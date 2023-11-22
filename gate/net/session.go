package net

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"hk4e/common/config"
	"hk4e/common/mq"
	"hk4e/dispatch/controller"
	"hk4e/gate/dao"
	"hk4e/gate/kcp"
	"hk4e/node/api"
	"hk4e/pkg/endec"
	"hk4e/pkg/httpclient"
	"hk4e/pkg/logger"
	"hk4e/pkg/random"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

// 会话管理

const (
	ConnEst = iota
	ConnWaitLogin
	ConnActive
	ConnClose
)

// 转发客户端消息到其他服务器 每个连接独立协程
func (k *KcpConnManager) forwardClientMsgToServerHandle(protoMsg *ProtoMsg, session *Session) {
	if session.connState == ConnClose {
		return
	}
	if protoMsg.HeadMessage == nil {
		logger.Error("recv null head msg: %v", protoMsg)
		return
	}
	// 网关服务器本地处理的请求
	switch protoMsg.CmdId {
	case cmd.GetPlayerTokenReq:
		// GATE登录包
		if session.connState != ConnEst {
			return
		}
		session.connState = ConnWaitLogin
		req := protoMsg.PayloadMessage.(*proto.GetPlayerTokenReq)
		rsp := k.doGateLogin(req, session)
		// 返回数据到客户端
		msg := &ProtoMsg{
			SessionId:      protoMsg.SessionId,
			CmdId:          cmd.GetPlayerTokenRsp,
			HeadMessage:    k.getHeadMsg(protoMsg.HeadMessage.ClientSequenceId),
			PayloadMessage: rsp,
		}
		session.sendChan <- msg
	case cmd.PlayerForceExitReq:
		// 退出游戏
		if session.connState != ConnActive {
			return
		}
		// 关闭连接
		k.closeKcpConnBySessionId(protoMsg.SessionId, kcp.EnetClientClose)
	case cmd.PlayerLoginReq:
		// GS登录包
		if session.connState != ConnWaitLogin {
			return
		}
		req := protoMsg.PayloadMessage.(*proto.PlayerLoginReq)
		req.TargetUid = 0
		req.TargetHomeOwnerUid = 0
		gameMsg := &mq.GameMsg{
			UserId:         session.userId,
			CmdId:          protoMsg.CmdId,
			ClientSeq:      protoMsg.HeadMessage.ClientSequenceId,
			PayloadMessage: req,
		}
		// 转发到GS
		k.messageQueue.SendToGs(session.gsServerAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeGame,
			EventId: mq.NormalMsg,
			GameMsg: gameMsg,
		})
	default:
		if session.connState != ConnActive {
			logger.Error("conn not active so drop packet, cmdId: %v, uid: %v, sessionId: %v",
				protoMsg.CmdId, session.userId, protoMsg.SessionId)
			return
		}
		gameMsg := &mq.GameMsg{
			UserId:             session.userId,
			CmdId:              protoMsg.CmdId,
			ClientSeq:          protoMsg.HeadMessage.ClientSequenceId,
			PayloadMessageData: nil,
		}
		// 在这里直接序列化成二进制数据 终结PayloadMessage的生命周期并回收进缓存池
		payloadMessageData, err := pb.Marshal(protoMsg.PayloadMessage)
		if err != nil {
			logger.Error("parse payload msg to bin error: %v, stack: %v", err, logger.Stack())
			return
		}
		k.serverCmdProtoMap.PutProtoObjCache(protoMsg.CmdId, protoMsg.PayloadMessage)
		gameMsg.PayloadMessageData = payloadMessageData
		// 转发到寻路服务
		if session.multiServerAppId != "" {
			if protoMsg.CmdId == cmd.QueryPathReq ||
				protoMsg.CmdId == cmd.ObstacleModifyNotify {
				k.messageQueue.SendToMulti(session.multiServerAppId, &mq.NetMsg{
					MsgType: mq.MsgTypeGame,
					EventId: mq.NormalMsg,
					GameMsg: gameMsg,
				})
			}
		}
		// 转发到GS
		k.messageQueue.SendToGs(session.gsServerAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeGame,
			EventId: mq.NormalMsg,
			GameMsg: gameMsg,
		})
		// 通知GS玩家客户端往返时延
		if protoMsg.CmdId == cmd.PingReq {
			rtt := uint32(0)
			if !session.conn.IsTcpMode() {
				logger.Debug("sessionId: %v, KcpRTO: %v, KcpSRTT: %v, KcpRTTVar: %v",
					protoMsg.SessionId, session.conn.GetKcpRTO(), session.conn.GetKcpSRTT(), session.conn.GetKcpSRTTVar())
				rtt = uint32(session.conn.GetKcpSRTT())
			} else {
				rtt = session.tcpRtt
			}
			connCtrlMsg := &mq.ConnCtrlMsg{
				UserId:    session.userId,
				ClientRtt: rtt,
			}
			k.messageQueue.SendToGs(session.gsServerAppId, &mq.NetMsg{
				MsgType:     mq.MsgTypeConnCtrl,
				EventId:     mq.ClientRttNotify,
				ConnCtrlMsg: connCtrlMsg,
			})
		}
	}
}

// 转发其他服务器的消息到客户端 所有连接共享一个协程
func (k *KcpConnManager) forwardServerMsgToClientHandle() {
	logger.Debug("server msg forward handle start")
	// 函数栈内缓存 添加删除事件走chan 避免频繁加锁
	sessionMap := make(map[uint32]*Session)
	userIdSessionIdMap := make(map[uint32]uint32)
	// 远程全局顶号注册列表
	reLoginRemoteKickRegMap := make(map[uint32]chan bool)
	for {
		select {
		case session := <-k.createSessionChan:
			sessionMap[session.sessionId] = session
			userIdSessionIdMap[session.userId] = session.sessionId
		case session := <-k.destroySessionChan:
			delete(sessionMap, session.sessionId)
			delete(userIdSessionIdMap, session.userId)
			close(session.sendChan)
		case remoteKick := <-k.reLoginRemoteKickRegChan:
			reLoginRemoteKickRegMap[remoteKick.userId] = remoteKick.kickFinishNotifyChan
			remoteKick.regFinishNotifyChan <- true
		case netMsg := <-k.messageQueue.GetNetMsg():
			switch netMsg.MsgType {
			case mq.MsgTypeGame:
				k.gameMsgHandle(netMsg, sessionMap, userIdSessionIdMap)
			case mq.MsgTypeConnCtrl:
				k.connCtrlMsgHandle(netMsg, userIdSessionIdMap)
			case mq.MsgTypeServer:
				k.serverMsgHandle(netMsg, sessionMap, userIdSessionIdMap, reLoginRemoteKickRegMap)
			}
		}
	}
}

func (k *KcpConnManager) gameMsgHandle(
	netMsg *mq.NetMsg,
	sessionMap map[uint32]*Session, userIdSessionIdMap map[uint32]uint32,
) {
	gameMsg := netMsg.GameMsg
	switch netMsg.EventId {
	case mq.NormalMsg:
		// 分发到每个连接具体的发送协程
		sessionId, exist := userIdSessionIdMap[gameMsg.UserId]
		if !exist {
			logger.Error("can not find sessionId by uid: %v, cmdId: %v", gameMsg.UserId, gameMsg.CmdId)
			return
		}
		protoMsg := &ProtoMsg{
			SessionId:      sessionId,
			CmdId:          gameMsg.CmdId,
			HeadMessage:    k.getHeadMsg(gameMsg.ClientSeq),
			PayloadMessage: gameMsg.PayloadMessage,
		}
		session := sessionMap[protoMsg.SessionId]
		if session == nil {
			logger.Error("session is nil, sessionId: %v", protoMsg.SessionId)
			return
		}
		if session.connState == ConnClose {
			return
		}
		if protoMsg.CmdId == cmd.PlayerLoginRsp {
			rsp := protoMsg.PayloadMessage.(*proto.PlayerLoginRsp)
			if rsp.Retcode == 0 {
				logger.Debug("session active, sessionId: %v", protoMsg.SessionId)
				session.connState = ConnActive
				// 通知GS玩家各个服务器的appid
				serverMsg := &mq.ServerMsg{
					UserId:           session.userId,
					MultiServerAppId: session.multiServerAppId,
				}
				k.messageQueue.SendToGs(session.gsServerAppId, &mq.NetMsg{
					MsgType:   mq.MsgTypeServer,
					EventId:   mq.ServerAppidBindNotify,
					ServerMsg: serverMsg,
				})
			}
		}
		if len(session.sendChan) == SessionSendChanLen {
			logger.Error("session send chan is full, sessionId: %v", protoMsg.SessionId)
			k.closeKcpConnBySessionId(sessionId, kcp.EnetWaitSndMax)
			return
		}
		session.sendChan <- protoMsg
	}
}

func (k *KcpConnManager) connCtrlMsgHandle(
	netMsg *mq.NetMsg,
	userIdSessionIdMap map[uint32]uint32,
) {
	connCtrlMsg := netMsg.ConnCtrlMsg
	switch netMsg.EventId {
	case mq.KickPlayerNotify:
		sessionId, exist := userIdSessionIdMap[connCtrlMsg.KickUserId]
		if !exist {
			logger.Error("can not find sessionId by uid: %v", connCtrlMsg.KickUserId)
			return
		}
		k.closeKcpConnBySessionId(sessionId, connCtrlMsg.KickReason)
	}
}

func (k *KcpConnManager) serverMsgHandle(
	netMsg *mq.NetMsg,
	sessionMap map[uint32]*Session, userIdSessionIdMap map[uint32]uint32,
	reLoginRemoteKickRegMap map[uint32]chan bool,
) {
	serverMsg := netMsg.ServerMsg
	switch netMsg.EventId {
	case mq.ServerUserGsChangeNotify:
		sessionId, exist := userIdSessionIdMap[serverMsg.UserId]
		if !exist {
			logger.Error("can not find sessionId by uid: %v", serverMsg.UserId)
			return
		}
		session := sessionMap[sessionId]
		if session == nil {
			logger.Error("session is nil, sessionId: %v", sessionId)
			return
		}
		session.gsServerAppId = serverMsg.GameServerAppId
		session.multiServerAppId = ""
		// 网关代发登录请求到新的GS
		gameMsg := &mq.GameMsg{
			UserId:    serverMsg.UserId,
			CmdId:     cmd.PlayerLoginReq,
			ClientSeq: 0,
			PayloadMessage: &proto.PlayerLoginReq{
				TargetUid:          serverMsg.JoinHostUserId,
				TargetHomeOwnerUid: 0,
			},
		}
		k.messageQueue.SendToGs(session.gsServerAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeGame,
			EventId: mq.NormalMsg,
			GameMsg: gameMsg,
		})
	case mq.ServerUserOnlineStateChangeNotify:
		// 收到GS玩家离线完成通知
		logger.Debug("global player online state change, uid: %v, online: %v, gs appid: %v",
			serverMsg.UserId, serverMsg.IsOnline, netMsg.OriginServerAppId)
		if serverMsg.IsOnline {
			k.globalGsOnlineMapLock.Lock()
			k.globalGsOnlineMap[serverMsg.UserId] = netMsg.OriginServerAppId
			k.globalGsOnlineMapLock.Unlock()
		} else {
			k.globalGsOnlineMapLock.Lock()
			delete(k.globalGsOnlineMap, serverMsg.UserId)
			k.globalGsOnlineMapLock.Unlock()
			kickFinishNotifyChan, exist := reLoginRemoteKickRegMap[serverMsg.UserId]
			if !exist {
				return
			}
			// 唤醒存在的顶号登录流程
			logger.Info("awake interrupt login, uid: %v", serverMsg.UserId)
			kickFinishNotifyChan <- true
			delete(reLoginRemoteKickRegMap, serverMsg.UserId)
		}
	}
}

// 转发客户端消息到robot
func (k *KcpConnManager) forwardClientMsgToRobotHandle(protoMsg *ProtoMsg, session *Session) {
	if protoMsg.HeadMessage == nil {
		logger.Error("recv null head msg: %v", protoMsg)
		return
	}
	// 网关服务器本地处理的请求
	switch protoMsg.CmdId {
	case cmd.GetPlayerTokenReq:
		// GATE登录包
		if session.connState != ConnEst {
			return
		}
		session.connState = ConnWaitLogin
		req := protoMsg.PayloadMessage.(*proto.GetPlayerTokenReq)
		session.keyId = req.KeyId
		session.clientRandKey = req.ClientRandKey
	case cmd.PlayerForceExitReq:
		// 退出游戏
		if session.connState != ConnActive {
			return
		}
	case cmd.PlayerLoginReq:
		// GS登录包
		if session.connState != ConnWaitLogin {
			return
		}
	}
	gameMsg := &mq.GameMsg{
		UserId:             session.userId,
		CmdId:              protoMsg.CmdId,
		ClientSeq:          protoMsg.HeadMessage.ClientSequenceId,
		PayloadMessageData: nil,
	}
	// 在这里直接序列化成二进制数据 终结PayloadMessage的生命周期并回收进缓存池
	payloadMessageData, err := pb.Marshal(protoMsg.PayloadMessage)
	if err != nil {
		logger.Error("parse payload msg to bin error: %v, stack: %v", err, logger.Stack())
		return
	}
	k.serverCmdProtoMap.PutProtoObjCache(protoMsg.CmdId, protoMsg.PayloadMessage)
	gameMsg.PayloadMessageData = payloadMessageData
	// 转发到Robot
	k.messageQueue.SendToRobot(session.robotServerAppId, &mq.NetMsg{
		MsgType: mq.MsgTypeGame,
		EventId: mq.NormalMsg,
		GameMsg: gameMsg,
	})
}

// 转发robot的消息到客户端
func (k *KcpConnManager) forwardRobotMsgToClientHandle(session *Session) {
	logger.Debug("robot msg forward handle start")
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			if session.connState == ConnClose {
				logger.Debug("robot msg forward handle stop")
				ticker.Stop()
				return
			}
		case netMsg := <-k.messageQueue.GetNetMsg():
			if netMsg.OriginServerType != api.ROBOT {
				continue
			}
			switch netMsg.MsgType {
			case mq.MsgTypeGame:
				gameMsg := netMsg.GameMsg
				switch netMsg.EventId {
				case mq.NormalMsg:
					switch gameMsg.CmdId {
					case cmd.GetPlayerTokenRsp:
						rsp := gameMsg.PayloadMessage.(*proto.GetPlayerTokenRsp)
						ok := k.keyExchange(session, rsp.Uid, rsp)
						if !ok {
							logger.Error("key exchange error, uid: %v", rsp.Uid)
							session.conn.Close()
							continue
						}
						// 关联玩家uid和连接信息
						session.userId = rsp.Uid
						k.SetSession(session, session.sessionId, session.userId)
						gameMsg.PayloadMessage = rsp
					case cmd.PlayerLoginRsp:
						logger.Debug("session active, uid: %v", gameMsg.UserId)
						session.connState = ConnActive
					}
					protoMsg := &ProtoMsg{
						SessionId:      session.sessionId,
						CmdId:          gameMsg.CmdId,
						HeadMessage:    k.getHeadMsg(gameMsg.ClientSeq),
						PayloadMessage: gameMsg.PayloadMessage,
					}
					if len(session.sendChan) == SessionSendChanLen {
						logger.Error("session send chan is full, sessionId: %v", protoMsg.SessionId)
						session.conn.Close()
						continue
					}
					session.sendChan <- protoMsg
				}
			case mq.MsgTypeServer:
				switch netMsg.EventId {
				case mq.ServerForwardModeServerCloseNotify:
					session.conn.Close()
				}
			}
		}
	}
}

func (k *KcpConnManager) getHeadMsg(clientSeq uint32) *proto.PacketHead {
	headMsg := new(proto.PacketHead)
	if clientSeq != 0 {
		headMsg.ClientSequenceId = clientSeq
		headMsg.SentMs = uint64(time.Now().UnixMilli())
	}
	return headMsg
}

type RemoteKick struct {
	regFinishNotifyChan  chan bool
	userId               uint32
	kickFinishNotifyChan chan bool
}

func (k *KcpConnManager) loginFailRsp(uid uint32, retCode proto.Retcode, isForbid bool, forbidEndTime uint32) *proto.GetPlayerTokenRsp {
	rsp := new(proto.GetPlayerTokenRsp)
	rsp.Uid = uid
	rsp.Retcode = int32(retCode)
	if isForbid {
		rsp.Msg = "FORBID_CHEATING_PLUGINS"
		rsp.BlackUidEndTime = forbidEndTime
		if rsp.BlackUidEndTime == 0 {
			rsp.BlackUidEndTime = 2051193600 // 2035-01-01 00:00:00
		}
	}
	return rsp
}

func (k *KcpConnManager) doGateLogin(req *proto.GetPlayerTokenReq, session *Session) *proto.GetPlayerTokenRsp {
	// 验证token
	signStr := fmt.Sprintf("app_id=%d&channel_id=%d&combo_token=%s&open_id=%s", 1, 1, req.AccountToken, req.AccountUid)
	signHash := hmac.New(sha256.New, []byte(config.GetConfig().Hk4e.LoginSdkAccountKey))
	signHash.Write([]byte(signStr))
	signData := signHash.Sum(nil)
	sign := hex.EncodeToString(signData)
	tokenVerifyRsp, err := httpclient.PostJson[controller.TokenVerifyRsp](
		config.GetConfig().Hk4e.LoginSdkUrl,
		&controller.TokenVerifyReq{
			AppID:      1,
			ChannelID:  1,
			OpenID:     req.AccountUid,
			ComboToken: req.AccountToken,
			Sign:       sign,
			Region:     "",
		})
	if err != nil {
		logger.Error("verify token http error: %v, openId: %v", err, req.AccountUid)
		return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
	}
	if tokenVerifyRsp.RetCode != 0 {
		logger.Error("verify token error, openId: %v", req.AccountUid)
		return k.loginFailRsp(0, proto.Retcode_RET_ACCOUNT_VEIRFY_ERROR, false, 0)
	}
	account, err := k.db.QueryAccountByOpenId(req.AccountUid)
	if err != nil {
		logger.Error("query account error: %v, openId: %v", err, req.AccountUid)
		return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
	}
	if account == nil {
		// 注册账号与uid关联
		getNextUidRsp, err := k.discoveryClient.GetNextUid(context.TODO(), &api.NullMsg{})
		if err != nil {
			logger.Error("get next uid error: %v, openId: %v", err, req.AccountUid)
			return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
		}
		account = &dao.Account{
			OpenId:        req.AccountUid,
			Uid:           getNextUidRsp.Uid,
			IsForbid:      false,
			ForbidEndTime: 0,
		}
		_, err = k.db.InsertAccount(account)
		if err != nil {
			logger.Error("insert account error: %v, openId: %v", err, req.AccountUid)
			return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
		}
	}
	uid := account.Uid
	if account.IsForbid {
		// 封号
		return k.loginFailRsp(uid, proto.Retcode_RET_BLACK_UID, true, account.ForbidEndTime)
	}
	addr := session.conn.RemoteAddr()
	addrSplit := strings.Split(addr, ":")
	clientIp := addrSplit[0]
	if k.stopServerInfo.StopServer {
		isWhiteList := false
		for _, ipAddr := range k.whiteList.IpAddrList {
			if ipAddr == clientIp {
				isWhiteList = true
				break
			}
		}
		if !isWhiteList {
			return k.loginFailRsp(uid, proto.Retcode_RET_STOP_SERVER, false, 0)
		}
	}
	clientConnNum := atomic.LoadInt32(&CLIENT_CONN_NUM)
	if clientConnNum > MaxClientConnNumLimit {
		logger.Error("gate conn num limit, uid: %v", uid)
		return k.loginFailRsp(uid, proto.Retcode_RET_MAX_PLAYER, false, 0)
	}
	k.globalGsOnlineMapLock.RLock()
	_, exist := k.globalGsOnlineMap[uid]
	k.globalGsOnlineMapLock.RUnlock()
	if exist {
		// 注册回调通知
		regFinishNotifyChan := make(chan bool, 1)
		kickFinishNotifyChan := make(chan bool, 1)
		k.reLoginRemoteKickRegChan <- &RemoteKick{
			regFinishNotifyChan:  regFinishNotifyChan,
			userId:               uid,
			kickFinishNotifyChan: kickFinishNotifyChan,
		}
		// 注册等待
		logger.Info("run global interrupt login reg wait, uid: %v", uid)
		timer := time.NewTimer(time.Second * 1)
		select {
		case <-timer.C:
			logger.Error("global interrupt login reg wait timeout, uid: %v", uid)
			timer.Stop()
			return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
		case <-regFinishNotifyChan:
			timer.Stop()
		}
		oldSession := k.GetSessionByUserId(uid)
		if oldSession != nil {
			// 本地顶号
			k.closeKcpConnBySessionId(oldSession.sessionId, kcp.EnetServerRelogin)
		} else {
			// 远程顶号
			connCtrlMsg := new(mq.ConnCtrlMsg)
			connCtrlMsg.KickUserId = uid
			connCtrlMsg.KickReason = kcp.EnetServerRelogin
			k.messageQueue.SendToAll(&mq.NetMsg{
				MsgType:     mq.MsgTypeConnCtrl,
				EventId:     mq.KickPlayerNotify,
				ConnCtrlMsg: connCtrlMsg,
			})
		}
		// 顶号等待
		logger.Info("run global interrupt login kick wait, uid: %v", uid)
		timer = time.NewTimer(time.Second * 10)
		select {
		case <-timer.C:
			logger.Error("global interrupt login kick wait timeout, uid: %v", uid)
			timer.Stop()
			return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
		case <-kickFinishNotifyChan:
			timer.Stop()
		}
	}
	// 关联玩家uid和连接信息
	session.userId = uid
	k.SetSession(session, session.sessionId, session.userId)
	k.createSessionChan <- session
	// 绑定各个服务器appid
	if k.minLoadGsServerAppId == "" {
		return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
	}
	session.gsServerAppId = k.minLoadGsServerAppId
	session.multiServerAppId = k.minLoadMultiServerAppId
	logger.Debug("session gs appid: %v, uid: %v", session.gsServerAppId, uid)
	logger.Debug("session multi appid: %v, uid: %v", session.multiServerAppId, uid)
	// 构造响应
	rsp := k.buildGateLoginRsp(uid, req.AccountUid, req.AccountToken, clientIp)
	// 密钥交换
	session.keyId = req.KeyId
	session.clientRandKey = req.ClientRandKey
	ok := k.keyExchange(session, uid, rsp)
	if !ok {
		logger.Error("key exchange error, uid: %v", uid)
		return k.loginFailRsp(0, proto.Retcode_RET_SVR_ERROR, false, 0)
	}
	return rsp
}

func (k *KcpConnManager) buildGateLoginRsp(uid uint32, accountUid string, token string, clientIp string) *proto.GetPlayerTokenRsp {
	rsp := &proto.GetPlayerTokenRsp{
		Uid:                    uid,
		AccountUid:             accountUid,
		Token:                  token,
		AccountType:            1,
		ChannelId:              1,
		SubChannelId:           1,
		PlatformType:           3,
		RegPlatform:            3,
		IsProficientPlayer:     false,
		CountryCode:            "US",
		Birthday:               "2000-01-01",
		ClientIpStr:            clientIp,
		SecurityCmdBuffer:      random.GetRandomByte(32),
		ClientVersionRandomKey: fmt.Sprintf("%03x-%012x", random.GetRandomByte(3), random.GetRandomByte(12)),
	}
	return rsp
}

func (k *KcpConnManager) keyExchange(session *Session, uid uint32, rsp *proto.GetPlayerTokenRsp) bool {
	timeRand := random.GetTimeRand()
	serverSeedUint64 := timeRand.Uint64()
	session.seed = serverSeedUint64
	if session.keyId != 0 {
		session.useMagicSeed = true
		keyId := strconv.Itoa(int(session.keyId))
		encPubPrivKey, exist := k.encRsaKeyMap[keyId]
		if !exist {
			logger.Error("can not found key id: %v, uid: %v", keyId, uid)
			return false
		}
		pubKey, err := endec.RsaParsePubKeyByPrivKey(encPubPrivKey)
		if err != nil {
			logger.Error("parse rsa pub key error: %v, uid: %v", err, uid)
			return false
		}
		signPrivkey, err := endec.RsaParsePrivKey(k.signRsaKey)
		if err != nil {
			logger.Error("parse rsa priv key error: %v, uid: %v", err, uid)
			return false
		}
		clientSeedBase64 := session.clientRandKey
		clientSeedEnc, err := base64.StdEncoding.DecodeString(clientSeedBase64)
		if err != nil {
			logger.Error("parse client seed base64 error: %v, uid: %v", err, uid)
			return false
		}
		clientSeed, err := endec.RsaDecrypt(clientSeedEnc, signPrivkey)
		if err != nil {
			logger.Error("rsa dec error: %v, uid: %v", err, uid)
			return false
		}
		clientSeedUint64 := uint64(0)
		err = binary.Read(bytes.NewReader(clientSeed), binary.BigEndian, &clientSeedUint64)
		if err != nil {
			logger.Error("parse client seed to uint64 error: %v, uid: %v", err, uid)
			return false
		}
		logger.Debug("clientSeed: %v, clientSeedUint64: %v", clientSeed, clientSeedUint64)
		logger.Debug("serverSeedUint64: %v", serverSeedUint64)
		seedUint64 := serverSeedUint64 ^ clientSeedUint64
		seedBuf := new(bytes.Buffer)
		err = binary.Write(seedBuf, binary.BigEndian, seedUint64)
		if err != nil {
			logger.Error("write seed uint64 to bytes error: %v, uid: %v", err, uid)
			return false
		}
		seed := seedBuf.Bytes()
		logger.Debug("seed: %v, seedUint64: %v", seed, seedUint64)
		seedEnc, err := endec.RsaEncrypt(seed, pubKey)
		if err != nil {
			logger.Error("rsa enc error: %v, uid: %v", err, uid)
			return false
		}
		seedSign, err := endec.RsaSign(seed, signPrivkey)
		if err != nil {
			logger.Error("rsa sign error: %v, uid: %v", err, uid)
			return false
		}
		rsp.KeyId = session.keyId
		rsp.ServerRandKey = base64.StdEncoding.EncodeToString(seedEnc)
		rsp.Sign = base64.StdEncoding.EncodeToString(seedSign)
	} else {
		session.useMagicSeed = false
		rsp.SecretKeySeed = serverSeedUint64
		rsp.SecretKey = fmt.Sprintf("%03x-%012x", random.GetRandomByte(3), random.GetRandomByte(12))
	}
	return true
}
