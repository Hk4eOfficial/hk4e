package net

import (
	"context"
	"encoding/binary"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"hk4e/common/config"
	"hk4e/common/mq"
	"hk4e/common/region"
	"hk4e/common/rpc"
	"hk4e/gate/client_proto"
	"hk4e/gate/dao"
	"hk4e/gate/kcp"
	"hk4e/node/api"
	"hk4e/pkg/logger"
	"hk4e/pkg/random"
	"hk4e/protocol/cmd"
)

// 网络连接管理

const (
	ConnEstFreqLimit      = 100        // 每秒连接建立频率限制
	RecvPacketFreqLimit   = 1000       // 客户端上行每秒发包频率限制
	SendPacketFreqLimit   = 1000       // 服务器下行每秒发包频率限制
	PacketMaxLen          = 343 * 1024 // 最大应用层包长度
	ConnRecvTimeout       = 30         // 收包超时时间 秒
	ConnSendTimeout       = 10         // 发包超时时间 秒
	MaxClientConnNumLimit = 1000       // 最大客户端连接数限制
	TcpNoDelay            = true       // 是否禁用tcp的nagle
	SessionSendChanLen    = 1000       // 会话发送管道缓存包容量
)

var CLIENT_CONN_NUM int32 = 0 // 当前客户端连接数

const (
	KcpConnEstNotify        = "KcpConnEstNotify"
	KcpConnAddrChangeNotify = "KcpConnAddrChangeNotify"
	KcpConnCloseNotify      = "KcpConnCloseNotify"
)

type KcpEvent struct {
	SessionId    uint32
	EventId      string
	EventMessage any
}

type KcpConnManager struct {
	kcpListener             *kcp.Listener
	db                      *dao.Dao
	discoveryClient         *rpc.DiscoveryClient // 节点服务器rpc客户端
	messageQueue            *mq.MessageQueue     // 消息队列
	globalGsOnlineMap       map[uint32]string    // 全服玩家在线表
	globalGsOnlineMapLock   sync.RWMutex
	minLoadGsServerAppId    string
	minLoadMultiServerAppId string
	stopServerInfo          *api.StopServerInfo
	whiteList               *api.GetWhiteListRsp
	// 会话
	sessionIdCounter uint32
	sessionMap       map[uint32]*Session
	sessionUserIdMap map[uint32]*Session
	sessionMapLock   sync.RWMutex
	// 事件
	createSessionChan        chan *Session
	destroySessionChan       chan *Session
	kcpEventChan             chan *KcpEvent
	reLoginRemoteKickRegChan chan *RemoteKick
	// 协议
	serverCmdProtoMap *cmd.CmdProtoMap
	clientCmdProtoMap *client_proto.ClientCmdProtoMap
	// 密钥
	signRsaKey   []byte
	encRsaKeyMap map[string][]byte
	dispatchKey  []byte
}

func NewKcpConnManager(db *dao.Dao, messageQueue *mq.MessageQueue, discovery *rpc.DiscoveryClient) (*KcpConnManager, error) {
	r := new(KcpConnManager)
	r.kcpListener = nil
	r.db = db
	r.discoveryClient = discovery
	r.messageQueue = messageQueue
	r.globalGsOnlineMap = make(map[uint32]string)
	r.minLoadGsServerAppId = ""
	r.minLoadMultiServerAppId = ""
	r.stopServerInfo = nil
	r.whiteList = nil
	r.sessionIdCounter = 0
	r.sessionMap = make(map[uint32]*Session)
	r.sessionUserIdMap = make(map[uint32]*Session)
	r.createSessionChan = make(chan *Session, 1000)
	r.destroySessionChan = make(chan *Session, 1000)
	r.kcpEventChan = make(chan *KcpEvent, 1000)
	r.reLoginRemoteKickRegChan = make(chan *RemoteKick, 1000)
	r.serverCmdProtoMap = cmd.NewCmdProtoMap()
	if config.GetConfig().Hk4e.ClientProtoProxyEnable {
		r.clientCmdProtoMap = client_proto.NewClientCmdProtoMap()
	}
	err := r.run()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (k *KcpConnManager) run() error {
	// 读取密钥相关文件
	k.signRsaKey, k.encRsaKeyMap, _ = region.LoadRegionRsaKey()
	// key
	rsp, err := k.discoveryClient.GetRegionEc2B(context.TODO(), &api.NullMsg{})
	if err != nil {
		logger.Error("get region ec2b error: %v", err)
		return err
	}
	ec2b, err := random.LoadEc2bKey(rsp.Data)
	if err != nil {
		logger.Error("parse region ec2b error: %v", err)
		return err
	}
	regionEc2b := random.NewEc2b()
	regionEc2b.SetSeed(ec2b.Seed())
	k.dispatchKey = regionEc2b.XorKey()
	// kcp
	addr := "0.0.0.0:" + strconv.Itoa(int(config.GetConfig().Hk4e.KcpPort))
	kcpListener, err := kcp.ListenWithOptions(addr)
	if err != nil {
		logger.Error("listen kcp err: %v", err)
		return err
	}
	k.kcpListener = kcpListener
	logger.Info("listen kcp at addr: %v", addr)
	go k.kcpNetInfo()
	go k.kcpEnetHandle(kcpListener)
	go k.acceptHandle(false, kcpListener, nil)
	if config.GetConfig().Hk4e.TcpModeEnable {
		// tcp
		addr := "0.0.0.0:" + strconv.Itoa(int(config.GetConfig().Hk4e.KcpPort))
		tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
		if err != nil {
			logger.Error("parse tcp addr err: %v", err)
			return err
		}
		tcpListener, err := net.ListenTCP("tcp4", tcpAddr)
		if err != nil {
			logger.Error("listen tcp err: %v", err)
			return err
		}
		logger.Info("listen tcp at addr: %v", addr)
		go k.acceptHandle(true, nil, tcpListener)
	}
	if !config.GetConfig().Hk4e.ForwardModeEnable {
		go k.forwardServerMsgToClientHandle()
	}
	k.syncGlobalGsOnlineMap()
	go k.autoSyncGlobalGsOnlineMap()
	k.syncMinLoadServerAppid()
	go k.autoSyncMinLoadServerAppid()
	k.syncWhiteList()
	go k.autoSyncWhiteList()
	k.syncStopServerInfo()
	go k.autoSyncStopServerInfo()
	go func() {
		for {
			kcpEvent := <-k.kcpEventChan
			logger.Info("[Kcp Event] kcpEvent: %+v", *kcpEvent)
		}
	}()
	return nil
}

func (k *KcpConnManager) Close() {
	k.closeAllKcpConn()
}

func (k *KcpConnManager) kcpNetInfo() {
	ticker := time.NewTicker(time.Second * 60)
	kcpErrorCount := uint64(0)
	for {
		<-ticker.C
		snmp := kcp.DefaultSnmp.Copy()
		kcpErrorCount += snmp.KCPInErrors
		logger.Info("kcp send: %v B/s, kcp recv: %v B/s", snmp.BytesSent/60, snmp.BytesReceived/60)
		logger.Info("udp send: %v B/s, udp recv: %v B/s", snmp.OutBytes/60, snmp.InBytes/60)
		logger.Info("udp send: %v pps, udp recv: %v pps", snmp.OutPkts/60, snmp.InPkts/60)
		clientConnNum := atomic.LoadInt32(&CLIENT_CONN_NUM)
		logger.Info("conn num: %v, new conn num: %v, kcp error num: %v", clientConnNum, snmp.CurrEstab, kcpErrorCount)
		kcp.DefaultSnmp.Reset()
	}
}

// 接收新连接协程
func (k *KcpConnManager) acceptHandle(tcpMode bool, kcpListener *kcp.Listener, tcpListener *net.TCPListener) {
	logger.Info("accept handle start, tcpMode: %v", tcpMode)
	connEstFreqLimitCounter := 0
	connEstFreqLimitTimer := time.Now().UnixNano()
	for {
		var conn *Conn = nil
		if !tcpMode {
			kcpConn, err := kcpListener.AcceptKCP()
			if err != nil {
				logger.Error("accept kcp err: %v", err)
				continue
			}
			kcpConn.SetACKNoDelay(true)
			kcpConn.SetWriteDelay(false)
			kcpConn.SetWindowSize(256, 256)
			kcpConn.SetMtu(1200)
			conn = NewKcpConn(kcpConn)
		} else {
			tcpConn, err := tcpListener.AcceptTCP()
			if err != nil {
				logger.Error("accept tcp err: %v", err)
				continue
			}
			if TcpNoDelay {
				_ = tcpConn.SetNoDelay(true)
			}
			conn = NewTcpConn(tcpConn)
		}
		// 连接建立频率限制
		connEstFreqLimitCounter++
		if connEstFreqLimitCounter > ConnEstFreqLimit {
			now := time.Now().UnixNano()
			if now-connEstFreqLimitTimer > int64(time.Second) {
				connEstFreqLimitCounter = 0
				connEstFreqLimitTimer = now
			} else {
				logger.Error("conn est freq limit, now: %v conn/s", connEstFreqLimitCounter)
				conn.Close()
				continue
			}
		}
		if config.GetConfig().Hk4e.ForwardModeEnable {
			clientConnNum := atomic.LoadInt32(&CLIENT_CONN_NUM)
			if clientConnNum != 0 {
				logger.Error("forward mode only support one client conn now")
				conn.Close()
				continue
			}
		}
		sessionId := uint32(0)
		if !tcpMode {
			sessionId = conn.GetSessionId()
		} else {
			sessionId = atomic.AddUint32(&k.sessionIdCounter, 1)
		}
		if !k.AddSession(sessionId) {
			logger.Error("session already exist, sessionId: %v", sessionId)
			conn.Close()
			continue
		}
		logger.Info("[ACCEPT] client connect, tcpMode: %v, sessionId: %v, conv: %v, addr: %v",
			tcpMode, sessionId, conn.GetConv(), conn.RemoteAddr())
		session := &Session{
			sessionId:          sessionId,
			conn:               conn,
			connState:          ConnEst,
			userId:             0,
			sendChan:           make(chan *ProtoMsg, SessionSendChanLen),
			seed:               0,
			xorKey:             k.dispatchKey,
			changeXorKeyFin:    false,
			gsServerAppId:      "",
			multiServerAppId:   "",
			robotServerAppId:   "",
			useMagicSeed:       false,
			keyId:              0,
			clientRandKey:      "",
			tcpRtt:             0,
			tcpRttLastSendTime: 0,
		}
		if config.GetConfig().Hk4e.ForwardModeEnable {
			robotServerAppId, err := k.discoveryClient.GetServerAppId(context.TODO(), &api.GetServerAppIdReq{
				ServerType: api.ROBOT,
			})
			if err != nil {
				logger.Error("get robot server appid error: %v", err)
				session.conn.Close()
				continue
			}
			session.robotServerAppId = robotServerAppId.AppId
			k.messageQueue.SendToRobot(session.robotServerAppId, &mq.NetMsg{
				MsgType: mq.MsgTypeServer,
				EventId: mq.ServerForwardModeClientConnNotify,
			})
		}
		go k.recvHandle(session)
		go k.sendHandle(session)
		if config.GetConfig().Hk4e.ForwardModeEnable {
			go k.forwardRobotMsgToClientHandle(session)
		}
		// 连接建立成功通知
		k.kcpEventChan <- &KcpEvent{
			SessionId:    session.sessionId,
			EventId:      KcpConnEstNotify,
			EventMessage: session.conn.RemoteAddr(),
		}
		atomic.AddInt32(&CLIENT_CONN_NUM, 1)
	}
}

// kcp连接事件处理函数
func (k *KcpConnManager) kcpEnetHandle(listener *kcp.Listener) {
	logger.Info("kcp enet handle start")
	for {
		enetNotify := <-listener.GetEnetNotifyChan()
		logger.Info("[Kcp Enet] addr: %v, conv: %v, sessionId: %v, connType: %v, enetType: %v",
			enetNotify.Addr, enetNotify.Conv, enetNotify.SessionId, enetNotify.ConnType, enetNotify.EnetType)
		switch enetNotify.ConnType {
		case kcp.ConnEnetSyn:
			if enetNotify.EnetType != kcp.EnetClientConnectKey {
				logger.Error("enet type not match, sessionId: %v", enetNotify.SessionId)
				continue
			}
			sessionId := atomic.AddUint32(&k.sessionIdCounter, 1)
			listener.SendEnetNotifyToPeer(&kcp.Enet{
				Addr:      enetNotify.Addr,
				SessionId: sessionId,
				Conv:      binary.BigEndian.Uint32(random.GetRandomByte(4)),
				ConnType:  kcp.ConnEnetEst,
				EnetType:  enetNotify.EnetType,
			})
		case kcp.ConnEnetFin:
			session := k.GetSession(enetNotify.SessionId)
			if session == nil {
				continue
			}
			if session.conn.GetConv() != enetNotify.Conv {
				logger.Error("conv not match, sessionId: %v", enetNotify.SessionId)
				continue
			}
			k.closeKcpConn(session, enetNotify.EnetType)
		case kcp.ConnEnetAddrChange:
			// 连接地址改变通知
			k.kcpEventChan <- &KcpEvent{
				SessionId:    enetNotify.SessionId,
				EventId:      KcpConnAddrChangeNotify,
				EventMessage: enetNotify.Addr,
			}
		default:
		}
	}
}

// Session 连接会话结构 只允许定义并发安全或者简单的基础数据结构
type Session struct {
	sessionId          uint32
	conn               *Conn
	connState          uint8
	userId             uint32
	sendChan           chan *ProtoMsg
	seed               uint64
	xorKey             []byte
	changeXorKeyFin    bool
	gsServerAppId      string
	multiServerAppId   string
	robotServerAppId   string
	useMagicSeed       bool
	keyId              uint32
	clientRandKey      string
	tcpRtt             uint32
	tcpRttLastSendTime int64
}

// 接收协程
func (k *KcpConnManager) recvHandle(session *Session) {
	logger.Info("recv handle start, sessionId: %v", session.sessionId)
	conn := session.conn
	header := make([]byte, 4)
	payload := make([]byte, PacketMaxLen)
	pktFreqLimitCounter := 0
	pktFreqLimitTimer := time.Now().UnixNano()
	for {
		var bin []byte = nil
		if !conn.IsTcpMode() {
			conn.SetReadDeadline(time.Now().Add(time.Second * ConnRecvTimeout))
			recvLen, err := conn.Read(payload)
			if err != nil {
				logger.Debug("exit recv loop, conn read err: %v, sessionId: %v", err, session.sessionId)
				k.closeKcpConn(session, kcp.EnetServerKick)
				return
			}
			bin = payload[:recvLen]
		} else {
			// tcp流分割解析
			recvLen := 0
			for recvLen < 4 {
				conn.SetReadDeadline(time.Now().Add(time.Second * ConnRecvTimeout))
				n, err := conn.Read(header[recvLen:])
				if err != nil {
					logger.Debug("exit recv loop, conn read err: %v, sessionId: %v", err, session.sessionId)
					k.closeKcpConn(session, kcp.EnetServerKick)
					return
				}
				recvLen += n
			}
			msgLen := binary.BigEndian.Uint32(header)
			// tcp rtt探测
			if msgLen == 0 {
				conn.SetWriteDeadline(time.Now().Add(time.Second * ConnSendTimeout))
				_, err := conn.Write([]byte{0x00, 0x00, 0x00, 0x00})
				if err != nil {
					logger.Debug("exit recv loop, conn write err: %v, sessionId: %v", err, session.sessionId)
					k.closeKcpConn(session, kcp.EnetServerKick)
					return
				}
				continue
			}
			if msgLen == 0xffffffff {
				now := time.Now().UnixMilli()
				session.tcpRtt = uint32(now - session.tcpRttLastSendTime)
				logger.Debug("[TCP RTT] sessionId: %v, rtt: %v ms", session.sessionId, session.tcpRtt)
				continue
			}
			if msgLen > PacketMaxLen {
				logger.Error("exit recv loop, msg len too long, sessionId: %v", session.sessionId)
				k.closeKcpConn(session, kcp.EnetServerKick)
				return
			}
			recvLen = 0
			for recvLen < int(msgLen) {
				conn.SetReadDeadline(time.Now().Add(time.Second * ConnRecvTimeout))
				n, err := conn.Read(payload[recvLen:msgLen])
				if err != nil {
					logger.Debug("exit recv loop, conn read err: %v, sessionId: %v", err, session.sessionId)
					k.closeKcpConn(session, kcp.EnetServerKick)
					return
				}
				recvLen += n
			}
			bin = payload[:msgLen]
		}
		// 收包频率限制
		pktFreqLimitCounter++
		if pktFreqLimitCounter > RecvPacketFreqLimit {
			now := time.Now().UnixNano()
			if now-pktFreqLimitTimer > int64(time.Second) {
				pktFreqLimitCounter = 0
				pktFreqLimitTimer = now
			} else {
				logger.Error("exit recv loop, client packet send freq too high, sessionId: %v, pps: %v",
					session.sessionId, pktFreqLimitCounter)
				k.closeKcpConn(session, kcp.EnetPacketFreqTooHigh)
				return
			}
		}
		kcpMsgList := make([]*KcpMsg, 0)
		DecodeBinToPayload(bin, session.sessionId, &kcpMsgList, session.xorKey)
		for _, v := range kcpMsgList {
			protoMsgList := ProtoDecode(v, k.serverCmdProtoMap, k.clientCmdProtoMap)
			for _, vv := range protoMsgList {
				if config.GetConfig().Hk4e.ForwardModeEnable {
					k.forwardClientMsgToRobotHandle(vv, session)
				} else {
					k.forwardClientMsgToServerHandle(vv, session)
				}
			}
		}
	}
}

// 发送协程
func (k *KcpConnManager) sendHandle(session *Session) {
	logger.Info("send handle start, sessionId: %v", session.sessionId)
	conn := session.conn
	pktFreqLimitCounter := 0
	pktFreqLimitTimer := time.Now().UnixNano()
	for {
		protoMsg, ok := <-session.sendChan
		if !ok {
			logger.Debug("exit send loop, send chan close, sessionId: %v", session.sessionId)
			k.closeKcpConn(session, kcp.EnetServerKick)
			return
		}
		kcpMsg := ProtoEncode(protoMsg, k.serverCmdProtoMap, k.clientCmdProtoMap)
		if kcpMsg == nil {
			logger.Error("encode kcp msg is nil, sessionId: %v", session.sessionId)
			continue
		}
		bin := EncodePayloadToBin(kcpMsg, session.xorKey)
		if conn.IsTcpMode() {
			// tcp流分割的4个字节payload长度头部
			headLenData := make([]byte, 4)
			binary.BigEndian.PutUint32(headLenData, uint32(len(bin)))
			conn.SetWriteDeadline(time.Now().Add(time.Second * ConnSendTimeout))
			_, err := conn.Write(headLenData)
			if err != nil {
				logger.Debug("exit send loop, conn write err: %v, sessionId: %v", err, session.sessionId)
				k.closeKcpConn(session, kcp.EnetServerKick)
				return
			}
		}
		conn.SetWriteDeadline(time.Now().Add(time.Second * ConnSendTimeout))
		_, err := conn.Write(bin)
		if err != nil {
			logger.Debug("exit send loop, conn write err: %v, sessionId: %v", err, session.sessionId)
			k.closeKcpConn(session, kcp.EnetServerKick)
			return
		}
		// 发包频率限制
		pktFreqLimitCounter++
		if pktFreqLimitCounter > SendPacketFreqLimit {
			now := time.Now().UnixNano()
			if now-pktFreqLimitTimer > int64(time.Second) {
				pktFreqLimitCounter = 0
				pktFreqLimitTimer = now
			} else {
				logger.Error("exit send loop, server packet send freq too high, sessionId: %v, pps: %v",
					session.sessionId, pktFreqLimitCounter)
				k.closeKcpConn(session, kcp.EnetPacketFreqTooHigh)
				return
			}
		}
		if session.changeXorKeyFin == false && protoMsg.CmdId == cmd.GetPlayerTokenRsp {
			// XOR密钥切换
			logger.Info("change session xor key, sessionId: %v", session.sessionId)
			session.changeXorKeyFin = true
			keyBlock := random.NewKeyBlock(session.seed, session.useMagicSeed)
			xorKey := keyBlock.XorKey()
			key := make([]byte, 4096)
			copy(key, xorKey[:])
			session.xorKey = key
		}
		if conn.IsTcpMode() {
			// tcp rtt探测
			now := time.Now().UnixMilli()
			if now-session.tcpRttLastSendTime > 1000 {
				conn.SetWriteDeadline(time.Now().Add(time.Second * ConnSendTimeout))
				_, err := conn.Write([]byte{0xff, 0xff, 0xff, 0xff})
				if err != nil {
					logger.Debug("exit send loop, conn write err: %v, sessionId: %v", err, session.sessionId)
					k.closeKcpConn(session, kcp.EnetServerKick)
					return
				}
				session.tcpRttLastSendTime = now
			}
		}
	}
}

// 关闭所有连接
func (k *KcpConnManager) closeAllKcpConn() {
	sessionList := make([]*Session, 0)
	k.sessionMapLock.RLock()
	for _, session := range k.sessionMap {
		sessionList = append(sessionList, session)
	}
	k.sessionMapLock.RUnlock()
	for _, session := range sessionList {
		if session == nil {
			continue
		}
		k.closeKcpConn(session, kcp.EnetServerShutdown)
	}
	logger.Info("all conn has been force close")
}

// 关闭指定连接
func (k *KcpConnManager) closeKcpConnBySessionId(sessionId uint32, reason uint32) {
	session := k.GetSession(sessionId)
	if session == nil {
		logger.Error("session not exist, sessionId: %v", sessionId)
		return
	}
	k.closeKcpConn(session, reason)
	logger.Info("conn has been close, sessionId: %v", sessionId)
}

// 关闭连接
func (k *KcpConnManager) closeKcpConn(session *Session, enetType uint32) {
	if session.connState == ConnClose {
		return
	}
	logger.Info("[CLOSE] client disconnect, tcpMode: %v, sessionId: %v, conv: %v, addr: %v",
		session.conn.IsTcpMode(), session.sessionId, session.conn.GetConv(), session.conn.RemoteAddr())
	session.connState = ConnClose
	// 清理数据
	k.DeleteSession(session.sessionId, session.userId)
	// 关闭连接
	if !session.conn.IsTcpMode() {
		k.kcpListener.SendEnetNotifyToPeer(&kcp.Enet{
			Addr:      session.conn.RemoteAddr(),
			SessionId: session.conn.GetSessionId(),
			Conv:      session.conn.GetConv(),
			ConnType:  kcp.ConnEnetFin,
			EnetType:  enetType,
		})
	}
	session.conn.Close()
	// 连接关闭通知
	k.kcpEventChan <- &KcpEvent{
		SessionId:    session.sessionId,
		EventId:      KcpConnCloseNotify,
		EventMessage: session.conn.RemoteAddr(),
	}
	if !config.GetConfig().Hk4e.ForwardModeEnable {
		// 通知GS玩家下线
		connCtrlMsg := new(mq.ConnCtrlMsg)
		connCtrlMsg.UserId = session.userId
		k.messageQueue.SendToGs(session.gsServerAppId, &mq.NetMsg{
			MsgType:     mq.MsgTypeConnCtrl,
			EventId:     mq.UserOfflineNotify,
			ConnCtrlMsg: connCtrlMsg,
		})
		logger.Info("send to gs user offline, sessionId: %v, uid: %v", session.sessionId, connCtrlMsg.UserId)
		k.destroySessionChan <- session
	} else {
		k.messageQueue.SendToRobot(session.robotServerAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeServer,
			EventId: mq.ServerForwardModeClientCloseNotify,
		})
	}
	atomic.AddInt32(&CLIENT_CONN_NUM, -1)
}

func (k *KcpConnManager) AddSession(sessionId uint32) bool {
	ok := false
	k.sessionMapLock.Lock()
	_, exist := k.sessionMap[sessionId]
	if !exist {
		k.sessionMap[sessionId] = nil
		ok = true
	}
	k.sessionMapLock.Unlock()
	return ok
}

func (k *KcpConnManager) GetSession(sessionId uint32) *Session {
	k.sessionMapLock.RLock()
	session, _ := k.sessionMap[sessionId]
	k.sessionMapLock.RUnlock()
	return session
}

func (k *KcpConnManager) GetSessionByUserId(userId uint32) *Session {
	k.sessionMapLock.RLock()
	session, _ := k.sessionUserIdMap[userId]
	k.sessionMapLock.RUnlock()
	return session
}

func (k *KcpConnManager) SetSession(session *Session, sessionId uint32, userId uint32) {
	k.sessionMapLock.Lock()
	k.sessionMap[sessionId] = session
	k.sessionUserIdMap[userId] = session
	k.sessionMapLock.Unlock()
}

func (k *KcpConnManager) DeleteSession(sessionId uint32, userId uint32) {
	k.sessionMapLock.Lock()
	delete(k.sessionMap, sessionId)
	delete(k.sessionUserIdMap, userId)
	k.sessionMapLock.Unlock()
}

func (k *KcpConnManager) autoSyncGlobalGsOnlineMap() {
	ticker := time.NewTicker(time.Second * 60)
	for {
		<-ticker.C
		k.syncGlobalGsOnlineMap()
	}
}

func (k *KcpConnManager) syncGlobalGsOnlineMap() {
	rsp, err := k.discoveryClient.GetGlobalGsOnlineMap(context.TODO(), nil)
	if err != nil {
		logger.Error("get global gs online map error: %v", err)
		return
	}
	copyMap := make(map[uint32]string)
	for k, v := range rsp.OnlineMap {
		copyMap[k] = v
	}
	copyMapLen := len(copyMap)
	k.globalGsOnlineMapLock.Lock()
	k.globalGsOnlineMap = copyMap
	k.globalGsOnlineMapLock.Unlock()
	logger.Info("sync global gs online map finish, len: %v", copyMapLen)
}

func (k *KcpConnManager) autoSyncMinLoadServerAppid() {
	ticker := time.NewTicker(time.Second * 15)
	for {
		<-ticker.C
		k.syncMinLoadServerAppid()
	}
}

func (k *KcpConnManager) syncMinLoadServerAppid() {
	gsServerAppId, err := k.discoveryClient.GetServerAppId(context.TODO(), &api.GetServerAppIdReq{
		ServerType: api.GS,
	})
	if err != nil {
		logger.Error("get gs server appid error: %v", err)
		k.minLoadGsServerAppId = ""
	} else {
		k.minLoadGsServerAppId = gsServerAppId.AppId
	}

	multiServerAppId, err := k.discoveryClient.GetServerAppId(context.TODO(), &api.GetServerAppIdReq{
		ServerType: api.MULTI,
	})
	if err != nil {
		k.minLoadMultiServerAppId = ""
	} else {
		k.minLoadMultiServerAppId = multiServerAppId.AppId
	}
}

func (k *KcpConnManager) autoSyncStopServerInfo() {
	ticker := time.NewTicker(time.Minute * 1)
	for {
		<-ticker.C
		k.syncStopServerInfo()
	}
}

func (k *KcpConnManager) syncStopServerInfo() {
	stopServerInfo, err := k.discoveryClient.GetStopServerInfo(context.TODO(), &api.NullMsg{})
	if err != nil {
		logger.Error("get stop server info error: %v", err)
		return
	}
	k.stopServerInfo = stopServerInfo
}

func (k *KcpConnManager) autoSyncWhiteList() {
	ticker := time.NewTicker(time.Minute * 1)
	for {
		<-ticker.C
		k.syncWhiteList()
	}
}

func (k *KcpConnManager) syncWhiteList() {
	whiteList, err := k.discoveryClient.GetWhiteList(context.TODO(), &api.NullMsg{})
	if err != nil {
		logger.Error("get white list error: %v", err)
		return
	}
	k.whiteList = whiteList
}
