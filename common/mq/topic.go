package mq

import (
	"hk4e/node/api"
)

func (m *MessageQueue) getOriginServer() (originServerType string, originServerAppId string) {
	originServerType = m.serverType
	originServerAppId = m.appId
	return originServerType, originServerAppId
}

func (m *MessageQueue) getTopic(serverType string, appId string) string {
	topic := serverType + "_" + appId + "_" + "HK4E"
	return topic
}

func (m *MessageQueue) SendToGate(appId string, netMsg *NetMsg) {
	netMsg.Topic = m.getTopic(api.GATE, appId)
	netMsg.ServerType = api.GATE
	netMsg.AppId = appId
	originServerType, originServerAppId := m.getOriginServer()
	netMsg.OriginServerType = originServerType
	netMsg.OriginServerAppId = originServerAppId
	m.netMsgInput <- netMsg
}

func (m *MessageQueue) SendToGs(appId string, netMsg *NetMsg) {
	netMsg.Topic = m.getTopic(api.GS, appId)
	netMsg.ServerType = api.GS
	netMsg.AppId = appId
	originServerType, originServerAppId := m.getOriginServer()
	netMsg.OriginServerType = originServerType
	netMsg.OriginServerAppId = originServerAppId
	m.netMsgInput <- netMsg
}

func (m *MessageQueue) SendToMulti(appId string, netMsg *NetMsg) {
	netMsg.Topic = m.getTopic(api.MULTI, appId)
	netMsg.ServerType = api.MULTI
	netMsg.AppId = appId
	originServerType, originServerAppId := m.getOriginServer()
	netMsg.OriginServerType = originServerType
	netMsg.OriginServerAppId = originServerAppId
	m.netMsgInput <- netMsg
}

func (m *MessageQueue) SendToRobot(appId string, netMsg *NetMsg) {
	netMsg.Topic = m.getTopic(api.ROBOT, appId)
	netMsg.ServerType = api.ROBOT
	netMsg.AppId = appId
	originServerType, originServerAppId := m.getOriginServer()
	netMsg.OriginServerType = originServerType
	netMsg.OriginServerAppId = originServerAppId
	m.netMsgInput <- netMsg
}

func (m *MessageQueue) SendToDispatch(appId string, netMsg *NetMsg) {
	netMsg.Topic = m.getTopic(api.DISPATCH, appId)
	netMsg.ServerType = api.DISPATCH
	netMsg.AppId = appId
	originServerType, originServerAppId := m.getOriginServer()
	netMsg.OriginServerType = originServerType
	netMsg.OriginServerAppId = originServerAppId
	m.netMsgInput <- netMsg
}

func (m *MessageQueue) SendToAll(netMsg *NetMsg) {
	netMsg.Topic = "ALL_SERVER_HK4E"
	netMsg.ServerType = "ALL_SERVER_HK4E"
	originServerType, originServerAppId := m.getOriginServer()
	netMsg.OriginServerType = originServerType
	netMsg.OriginServerAppId = originServerAppId
	m.netMsgInput <- netMsg
}
