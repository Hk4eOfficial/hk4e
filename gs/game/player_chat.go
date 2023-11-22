package game

import (
	"time"

	"hk4e/common/mq"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

/************************************************** 接口请求 **************************************************/

const (
	MaxMsgListLen = 100 // 与某人的最大聊天记录条数
)

func (g *Game) PullRecentChatReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PullRecentChatReq)
	// 经研究发现 原神现网环境 客户端仅拉取最新的5条未读聊天消息 所以人太多的话小姐姐不回你消息是有原因的
	// 因此 阿米你这样做真的合适吗 不过现在代码到了我手上我想怎么写就怎么写 我才不会重蹈覆辙
	_ = req.PullNum

	retMsgList := make([]*proto.ChatInfo, 0)
	for _, msgList := range player.ChatMsgMap {
		for _, chatMsg := range msgList {
			// 反手就是一个遍历
			if chatMsg.IsRead {
				continue
			}
			retMsgList = append(retMsgList, g.ConvChatMsgToChatInfo(chatMsg))
		}
	}

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	if world.IsMultiplayerWorld() {
		chatList := world.GetChatList()
		count := len(chatList)
		if count > 10 {
			count = 10
		}
		for i := len(chatList) - count; i < len(chatList); i++ {
			playerChatNotify := &proto.PlayerChatNotify{
				ChannelId: 0,
				ChatInfo:  chatList[i],
			}
			g.SendMsg(cmd.PlayerChatNotify, player.PlayerId, player.ClientSeq, playerChatNotify)
		}
	}

	pullRecentChatRsp := &proto.PullRecentChatRsp{
		ChatInfo: retMsgList,
	}
	g.SendMsg(cmd.PullRecentChatRsp, player.PlayerId, player.ClientSeq, pullRecentChatRsp)
}

func (g *Game) PullPrivateChatReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PullPrivateChatReq)
	targetUid := req.TargetUid
	pullNum := req.PullNum
	fromSequence := req.FromSequence

	msgList, exist := player.ChatMsgMap[targetUid]
	if !exist {
		return
	}
	if pullNum+fromSequence > uint32(len(msgList)) {
		pullNum = uint32(len(msgList)) - fromSequence
	}
	recentMsgList := msgList[fromSequence : fromSequence+pullNum]
	retMsgList := make([]*proto.ChatInfo, 0)
	for _, chatMsg := range recentMsgList {
		retMsgList = append(retMsgList, g.ConvChatMsgToChatInfo(chatMsg))
	}

	pullPrivateChatRsp := &proto.PullPrivateChatRsp{
		ChatInfo: retMsgList,
	}
	g.SendMsg(cmd.PullPrivateChatRsp, player.PlayerId, player.ClientSeq, pullPrivateChatRsp)
}

func (g *Game) PrivateChatReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PrivateChatReq)
	targetUid := req.TargetUid
	content := req.Content

	// 根据发送的类型发送消息
	switch content.(type) {
	case *proto.PrivateChatReq_Text:
		text := content.(*proto.PrivateChatReq_Text).Text
		if len(text) == 0 || len(text) > 80 {
			g.SendError(cmd.PrivateChatRsp, player, &proto.PrivateChatRsp{}, proto.Retcode_RET_PRIVATE_CHAT_CONTENT_TOO_LONG)
			return
		}
		// 发送私聊文本消息
		g.SendPrivateChat(player, targetUid, text)
		// 输入命令 会检测是否为命令的
		COMMAND_MANAGER.PlayerInputCommand(player, targetUid, text)
	case *proto.PrivateChatReq_Icon:
		icon := content.(*proto.PrivateChatReq_Icon).Icon
		// 发送私聊图标消息
		g.SendPrivateChat(player, targetUid, icon)
	default:
		return
	}

	g.SendMsg(cmd.PrivateChatRsp, player.PlayerId, player.ClientSeq, new(proto.PrivateChatRsp))
}

func (g *Game) ReadPrivateChatReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.ReadPrivateChatReq)
	targetUid := req.TargetUid

	msgList, exist := player.ChatMsgMap[targetUid]
	if !exist {
		return
	}
	for index, chatMsg := range msgList {
		chatMsg.IsRead = true
		msgList[index] = chatMsg
	}
	player.ChatMsgMap[targetUid] = msgList

	// 更新db
	go USER_MANAGER.ReadUserChatMsgToDbSync(player.PlayerId, targetUid)

	g.SendMsg(cmd.ReadPrivateChatRsp, player.PlayerId, player.ClientSeq, new(proto.ReadPrivateChatRsp))
}

func (g *Game) PlayerChatReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PlayerChatReq)
	channelId := req.ChannelId
	chatInfo := req.ChatInfo

	sendChatInfo := &proto.ChatInfo{
		Time:    uint32(time.Now().Unix()),
		Uid:     player.PlayerId,
		Content: nil,
	}
	switch chatInfo.Content.(type) {
	case *proto.ChatInfo_Text:
		text := chatInfo.Content.(*proto.ChatInfo_Text).Text
		if len(text) == 0 {
			return
		}
		sendChatInfo.Content = &proto.ChatInfo_Text{
			Text: text,
		}
	case *proto.ChatInfo_Icon:
		icon := chatInfo.Content.(*proto.ChatInfo_Icon).Icon
		sendChatInfo.Content = &proto.ChatInfo_Icon{
			Icon: icon,
		}
	default:
		return
	}

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	world.AddChat(sendChatInfo)

	ntf := &proto.PlayerChatNotify{
		ChannelId: channelId,
		ChatInfo:  sendChatInfo,
	}
	g.SendToWorldA(world, cmd.PlayerChatNotify, player.ClientSeq, ntf, 0)

	g.SendMsg(cmd.PlayerChatRsp, player.PlayerId, player.ClientSeq, new(proto.PlayerChatRsp))
}

func (g *Game) GetChatEmojiCollectionReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetChatEmojiCollectionReq)
	_ = req
	g.SendMsg(cmd.GetChatEmojiCollectionRsp, player.PlayerId, player.ClientSeq, new(proto.GetChatEmojiCollectionRsp))
}

/************************************************** 游戏功能 **************************************************/

// SendPrivateChat 发送私聊文本消息给玩家
func (g *Game) SendPrivateChat(player *model.Player, targetUid uint32, content any) {
	chatMsg := &model.ChatMsg{
		Sequence: 0,
		Time:     uint32(time.Now().Unix()),
		ToUid:    targetUid,
		Uid:      player.PlayerId,
		IsRead:   false,
		IsDelete: false,
	}
	// 根据传入的值判断消息类型
	switch content.(type) {
	case string:
		// 文本消息
		chatMsg.MsgType = model.ChatMsgTypeText
		chatMsg.Text = content.(string)
	case uint32:
		// 图标消息
		chatMsg.MsgType = model.ChatMsgTypeIcon
		chatMsg.Icon = content.(uint32)
	}

	if player.PlayerId != COMMAND_MANAGER.system.PlayerId && targetUid != COMMAND_MANAGER.system.PlayerId {
		// 写入db
		go USER_MANAGER.SaveUserChatMsgToDbSync(chatMsg)
	}

	// 消息加入自己的队列
	msgList, exist := player.ChatMsgMap[targetUid]
	// 处理序号
	if !exist {
		msgList = make([]*model.ChatMsg, 0)
		chatMsg.Sequence = 101
	} else {
		chatMsg.Sequence = uint32(len(msgList)) + 101
	}
	if len(msgList) > MaxMsgListLen {
		msgList = msgList[1:]
	}
	msgList = append(msgList, chatMsg)
	player.ChatMsgMap[targetUid] = msgList

	chatInfo := g.ConvChatMsgToChatInfo(chatMsg)

	privateChatNotify := &proto.PrivateChatNotify{
		ChatInfo: chatInfo,
	}
	g.SendMsg(cmd.PrivateChatNotify, player.PlayerId, player.ClientSeq, privateChatNotify)

	targetPlayer := USER_MANAGER.GetOnlineUser(targetUid)
	if targetPlayer == nil {
		if USER_MANAGER.GetRemoteUserOnlineState(targetUid) {
			// 目标玩家在别的服在线
			gsAppId := USER_MANAGER.GetRemoteUserGsAppId(targetUid)
			g.messageQueue.SendToGs(gsAppId, &mq.NetMsg{
				MsgType: mq.MsgTypeServer,
				EventId: mq.ServerChatMsgNotify,
				ServerMsg: &mq.ServerMsg{
					ChatMsgInfo: &mq.ChatMsgInfo{
						Time:     chatMsg.Time,
						ToUid:    chatMsg.ToUid,
						Uid:      chatMsg.Uid,
						IsRead:   chatMsg.IsRead,
						MsgType:  chatMsg.MsgType,
						Text:     chatMsg.Text,
						Icon:     chatMsg.Icon,
						IsDelete: chatMsg.IsDelete,
					},
				},
			})
		}
		return
	}

	// 消息加入目标玩家的队列
	msgList, exist = targetPlayer.ChatMsgMap[player.PlayerId]
	if !exist {
		msgList = make([]*model.ChatMsg, 0)
	}
	if len(msgList) > MaxMsgListLen {
		msgList = msgList[1:]
	}
	msgList = append(msgList, chatMsg)
	targetPlayer.ChatMsgMap[player.PlayerId] = msgList

	// 如果目标玩家在线发送消息
	if targetPlayer.Online {
		privateChatNotify := &proto.PrivateChatNotify{
			ChatInfo: chatInfo,
		}
		g.SendMsg(cmd.PrivateChatNotify, targetPlayer.PlayerId, targetPlayer.ClientSeq, privateChatNotify)
	}
}

func (g *Game) ConvChatInfoToChatMsg(chatInfo *proto.ChatInfo) (chatMsg *model.ChatMsg) {
	chatMsg = &model.ChatMsg{
		Sequence: chatInfo.Sequence,
		Time:     chatInfo.Time,
		ToUid:    chatInfo.ToUid,
		Uid:      chatInfo.Uid,
		IsRead:   chatInfo.IsRead,
		MsgType:  0,
		Text:     "",
		Icon:     0,
		IsDelete: false,
	}
	switch chatInfo.Content.(type) {
	case *proto.ChatInfo_Text:
		chatMsg.MsgType = model.ChatMsgTypeText
		chatMsg.Text = chatInfo.Content.(*proto.ChatInfo_Text).Text
	case *proto.ChatInfo_Icon:
		chatMsg.MsgType = model.ChatMsgTypeIcon
		chatMsg.Icon = chatInfo.Content.(*proto.ChatInfo_Icon).Icon
	default:
		logger.Error("chat info content type error, contentType: %T", chatInfo.Content)
	}
	return chatMsg
}

func (g *Game) ConvChatMsgToChatInfo(chatMsg *model.ChatMsg) (chatInfo *proto.ChatInfo) {
	chatInfo = &proto.ChatInfo{
		Time:     chatMsg.Time,
		Sequence: chatMsg.Sequence,
		ToUid:    chatMsg.ToUid,
		Uid:      chatMsg.Uid,
		IsRead:   chatMsg.IsRead,
		Content:  nil,
	}
	switch chatMsg.MsgType {
	case model.ChatMsgTypeText:
		chatInfo.Content = &proto.ChatInfo_Text{
			Text: chatMsg.Text,
		}
	case model.ChatMsgTypeIcon:
		chatInfo.Content = &proto.ChatInfo_Icon{
			Icon: chatMsg.Icon,
		}
	default:
		logger.Error("chat info content type error, msgType: %v", chatMsg.MsgType)
	}
	return chatInfo
}

// 跨服玩家聊天通知

func (g *Game) ServerChatMsgNotify(chatMsgInfo *mq.ChatMsgInfo) {
	targetPlayer := USER_MANAGER.GetOnlineUser(chatMsgInfo.ToUid)
	if targetPlayer == nil {
		logger.Error("player is nil, uid: %v", chatMsgInfo.ToUid)
		return
	}
	chatMsg := &model.ChatMsg{
		Time:     chatMsgInfo.Time,
		ToUid:    chatMsgInfo.ToUid,
		Uid:      chatMsgInfo.Uid,
		IsRead:   chatMsgInfo.IsRead,
		MsgType:  chatMsgInfo.MsgType,
		Text:     chatMsgInfo.Text,
		Icon:     chatMsgInfo.Icon,
		IsDelete: chatMsgInfo.IsDelete,
	}
	// 消息加入目标玩家的队列
	msgList, exist := targetPlayer.ChatMsgMap[chatMsgInfo.Uid]
	if !exist {
		msgList = make([]*model.ChatMsg, 0)
	}
	if len(msgList) > MaxMsgListLen {
		msgList = msgList[1:]
	}
	msgList = append(msgList, chatMsg)
	targetPlayer.ChatMsgMap[chatMsgInfo.Uid] = msgList

	// 如果目标玩家在线发送消息
	if targetPlayer.Online {
		privateChatNotify := &proto.PrivateChatNotify{
			ChatInfo: g.ConvChatMsgToChatInfo(chatMsg),
		}
		g.SendMsg(cmd.PrivateChatNotify, targetPlayer.PlayerId, targetPlayer.ClientSeq, privateChatNotify)
	}
}

/************************************************** 打包封装 **************************************************/
