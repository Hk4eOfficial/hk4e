package net

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"hk4e/common/config"

	"hk4e/gate/client_proto"
	"hk4e/gate/kcp"
	hk4egatenet "hk4e/gate/net"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	"github.com/gorilla/websocket"
	pb "google.golang.org/protobuf/proto"
)

type Packet struct {
	Time       uint64 `json:"time"`
	Dir        string `json:"dir"`
	CmdId      uint32 `json:"cmd_id"`
	CmdName    string `json:"cmd_name"`
	HeadMsg    string `json:"head_msg"`
	PayloadMsg string `json:"payload_msg"`
}

type Session struct {
	Conn                   *kcp.UDPSession
	XorKey                 []byte
	SendChan               chan *hk4egatenet.ProtoMsg
	RecvChan               chan *hk4egatenet.ProtoMsg
	ServerCmdProtoMap      *cmd.CmdProtoMap
	ClientCmdProtoMap      *client_proto.ClientCmdProtoMap
	ClientSeq              uint32
	DeadEvent              chan bool
	ClientVersionRandomKey string
	SecurityCmdBuffer      []byte
	Uid                    uint32
	IsClose                bool
	PktList                []*Packet
	PktCapWsConn           *websocket.Conn
	PktLock                sync.Mutex
}

func NewSession(gateAddr string, dispatchKey []byte) (*Session, error) {
	conn, err := kcp.DialWithOptions(gateAddr)
	if err != nil {
		logger.Error("kcp client conn to server error: %v", err)
		return nil, err
	}
	conn.SetACKNoDelay(true)
	conn.SetWriteDelay(false)
	conn.SetWindowSize(256, 256)
	conn.SetMtu(1200)
	r := &Session{
		Conn:                   conn,
		XorKey:                 dispatchKey,
		SendChan:               make(chan *hk4egatenet.ProtoMsg, 1000),
		RecvChan:               make(chan *hk4egatenet.ProtoMsg, 1000),
		ServerCmdProtoMap:      cmd.NewCmdProtoMap(),
		ClientCmdProtoMap:      nil,
		ClientSeq:              0,
		DeadEvent:              make(chan bool, 1),
		ClientVersionRandomKey: "",
		SecurityCmdBuffer:      nil,
		Uid:                    0,
		IsClose:                false,
		PktList:                make([]*Packet, 0),
		PktCapWsConn:           nil,
	}
	if config.GetConfig().Hk4e.ClientProtoProxyEnable {
		r.ClientCmdProtoMap = client_proto.NewClientCmdProtoMap()
	}
	go r.recvHandle()
	go r.sendHandle()
	return r, nil
}

func (s *Session) SendMsg(cmdId uint16, msg pb.Message) {
	s.SendChan <- &hk4egatenet.ProtoMsg{
		SessionId: 0,
		CmdId:     cmdId,
		HeadMessage: &proto.PacketHead{
			ClientSequenceId: atomic.AddUint32(&s.ClientSeq, 1),
			SentMs:           uint64(time.Now().UnixMilli()),
			EnetIsReliable:   1,
		},
		PayloadMessage: msg,
	}
}

func (s *Session) SendMsgFwd(cmdId uint16, clientSeq uint32, msg pb.Message) {
	s.SendChan <- &hk4egatenet.ProtoMsg{
		SessionId: 0,
		CmdId:     cmdId,
		HeadMessage: &proto.PacketHead{
			ClientSequenceId: clientSeq,
			SentMs:           uint64(time.Now().UnixMilli()),
			EnetIsReliable:   1,
		},
		PayloadMessage: msg,
	}
}

func (s *Session) Close() {
	if s.IsClose {
		return
	}
	s.IsClose = true
	_ = s.Conn.Close()
	s.DeadEvent <- true
}

func (s *Session) recvHandle() {
	logger.Info("recv handle start")
	conn := s.Conn
	convId := conn.GetConv()
	recvBuf := make([]byte, hk4egatenet.PacketMaxLen)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * hk4egatenet.ConnRecvTimeout))
		recvLen, err := conn.Read(recvBuf)
		if err != nil {
			logger.Error("exit recv loop, conn read err: %v, convId: %v", err, convId)
			s.Close()
			break
		}
		recvData := recvBuf[:recvLen]
		kcpMsgList := make([]*hk4egatenet.KcpMsg, 0)
		hk4egatenet.DecodeBinToPayload(recvData, convId, &kcpMsgList, s.XorKey)
		for _, v := range kcpMsgList {
			protoMsgList := hk4egatenet.ProtoDecode(v, s.ServerCmdProtoMap, s.ClientCmdProtoMap)
			for _, vv := range protoMsgList {
				s.RecvChan <- vv
				cmdName := string(vv.PayloadMessage.ProtoReflect().Descriptor().FullName())
				headMsg, _ := json.Marshal(vv.HeadMessage)
				payloadMsg, _ := json.Marshal(vv.PayloadMessage)
				packet := &Packet{
					Time:       uint64(time.Now().UnixMilli()),
					Dir:        "RECV",
					CmdId:      uint32(vv.CmdId),
					CmdName:    cmdName,
					HeadMsg:    string(headMsg),
					PayloadMsg: string(payloadMsg),
				}
				packetData, _ := json.Marshal(packet)
				s.PktLock.Lock()
				s.PktList = append(s.PktList, packet)
				if s.PktCapWsConn != nil {
					err := s.PktCapWsConn.WriteMessage(websocket.TextMessage, packetData)
					if err != nil {
						s.PktCapWsConn = nil
					}
				}
				s.PktLock.Unlock()
			}
		}
	}
}

func (s *Session) sendHandle() {
	logger.Info("send handle start")
	conn := s.Conn
	convId := conn.GetConv()
	for {
		protoMsg, ok := <-s.SendChan
		if !ok {
			logger.Error("exit send loop, send chan close, convId: %v", convId)
			s.Close()
			break
		}
		kcpMsg := hk4egatenet.ProtoEncode(protoMsg, s.ServerCmdProtoMap, s.ClientCmdProtoMap)
		if kcpMsg == nil {
			logger.Error("decode kcp msg is nil, convId: %v", convId)
			continue
		}
		bin := hk4egatenet.EncodePayloadToBin(kcpMsg, s.XorKey)
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second * hk4egatenet.ConnSendTimeout))
		_, err := conn.Write(bin)
		if err != nil {
			logger.Error("exit send loop, conn write err: %v, convId: %v", err, convId)
			s.Close()
			break
		}
		cmdName := string(protoMsg.PayloadMessage.ProtoReflect().Descriptor().FullName())
		headMsg, _ := json.Marshal(protoMsg.HeadMessage)
		payloadMsg, _ := json.Marshal(protoMsg.PayloadMessage)
		packet := &Packet{
			Time:       uint64(time.Now().UnixMilli()),
			Dir:        "SEND",
			CmdId:      uint32(protoMsg.CmdId),
			CmdName:    cmdName,
			HeadMsg:    string(headMsg),
			PayloadMsg: string(payloadMsg),
		}
		packetData, _ := json.Marshal(packet)
		s.PktLock.Lock()
		s.PktList = append(s.PktList, packet)
		if s.PktCapWsConn != nil {
			err := s.PktCapWsConn.WriteMessage(websocket.TextMessage, packetData)
			if err != nil {
				s.PktCapWsConn = nil
			}
		}
		s.PktLock.Unlock()
	}
}
