package mq

import (
	"hk4e/common/config"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"

	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	pb "google.golang.org/protobuf/proto"
)

type MessageQueue struct {
	natsConn     *nats.Conn
	natsMsgChan  chan *nats.Msg
	netMsgInput  chan *NetMsg
	netMsgOutput chan *NetMsg
	cmdProtoMap  *cmd.CmdProtoMap
}

func NewMessageQueue(serverType string, appId string) (r *MessageQueue) {
	r = new(MessageQueue)
	conn, err := nats.Connect(config.CONF.MQ.NatsUrl)
	if err != nil {
		logger.Error("connect nats error: %v", err)
		return nil
	}
	r.natsConn = conn
	r.natsMsgChan = make(chan *nats.Msg, 1000)
	_, err = r.natsConn.ChanSubscribe(r.getTopic(serverType, appId), r.natsMsgChan)
	if err != nil {
		logger.Error("nats subscribe error: %v", err)
		return nil
	}
	r.netMsgInput = make(chan *NetMsg, 1000)
	r.netMsgOutput = make(chan *NetMsg, 1000)
	r.cmdProtoMap = cmd.NewCmdProtoMap()
	go r.recvHandler()
	go r.sendHandler()
	return r
}

func (m *MessageQueue) Close() {
	m.natsConn.Close()
}

func (m *MessageQueue) GetNetMsg() chan *NetMsg {
	return m.netMsgOutput
}

func (m *MessageQueue) recvHandler() {
	for {
		natsMsg := <-m.natsMsgChan
		// msgpack NetMsg
		netMsg := new(NetMsg)
		err := msgpack.Unmarshal(natsMsg.Data, netMsg)
		if err != nil {
			logger.Error("parse bin to net msg error: %v", err)
			continue
		}
		switch netMsg.MsgType {
		case MsgTypeGame:
			gameMsg := netMsg.GameMsg
			if netMsg.EventId == NormalMsg || netMsg.EventId == UserRegNotify {
				// protobuf PayloadMessage
				payloadMessage := m.cmdProtoMap.GetProtoObjByCmdId(gameMsg.CmdId)
				if payloadMessage == nil {
					logger.Error("get protobuf obj by cmd id error: %v", err)
					continue
				}
				err = pb.Unmarshal(gameMsg.PayloadMessageData, payloadMessage)
				if err != nil {
					logger.Error("parse bin to payload msg error: %v", err)
					continue
				}
				gameMsg.PayloadMessage = payloadMessage
			}
		case MsgTypeFight:
		}
		m.netMsgOutput <- netMsg
	}
}

func (m *MessageQueue) sendHandler() {
	for {
		netMsg := <-m.netMsgInput
		switch netMsg.MsgType {
		case MsgTypeGame:
			gameMsg := netMsg.GameMsg
			if gameMsg.PayloadMessageData == nil {
				// protobuf PayloadMessage
				payloadMessageData, err := pb.Marshal(gameMsg.PayloadMessage)
				if err != nil {
					logger.Error("parse payload msg to bin error: %v", err)
					continue
				}
				gameMsg.PayloadMessageData = payloadMessageData
			}
		case MsgTypeFight:
		}
		// msgpack NetMsg
		netMsgData, err := msgpack.Marshal(netMsg)
		if err != nil {
			logger.Error("parse net msg to bin error: %v", err)
			continue
		}
		natsMsg := nats.NewMsg(netMsg.Topic)
		natsMsg.Data = netMsgData
		err = m.natsConn.PublishMsg(natsMsg)
		if err != nil {
			logger.Error("nats publish msg error: %v", err)
			continue
		}
	}
}