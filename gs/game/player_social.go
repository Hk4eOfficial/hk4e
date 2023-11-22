package game

import (
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"hk4e/common/constant"
	"hk4e/common/mq"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/pkg/object"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

/************************************************** 接口请求 **************************************************/

func (g *Game) GetPlayerSocialDetailReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetPlayerSocialDetailReq)
	targetUid := req.Uid

	targetPlayer, _, _ := USER_MANAGER.LoadGlobalPlayer(targetUid)
	if targetPlayer == nil {
		g.SendError(cmd.GetPlayerSocialDetailRsp, player, &proto.GetPlayerSocialDetailRsp{}, proto.Retcode_RET_PLAYER_NOT_EXIST)
		return
	}
	dbSocial := player.GetDbSocial()
	socialDetail := &proto.SocialDetail{
		Uid:                  targetPlayer.PlayerId,
		ProfilePicture:       &proto.ProfilePicture{AvatarId: targetPlayer.HeadImage},
		Nickname:             targetPlayer.NickName,
		Signature:            targetPlayer.Signature,
		Level:                targetPlayer.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
		Birthday:             &proto.Birthday{Month: dbSocial.GetBirthdayMonth(), Day: dbSocial.GetBirthdayDay()},
		WorldLevel:           targetPlayer.PropMap[constant.PLAYER_PROP_PLAYER_WORLD_LEVEL],
		NameCardId:           dbSocial.NameCard,
		IsShowAvatar:         false,
		FinishAchievementNum: 0,
		IsFriend:             dbSocial.IsFriend(targetPlayer.PlayerId),
	}
	getPlayerSocialDetailRsp := &proto.GetPlayerSocialDetailRsp{
		DetailData: socialDetail,
	}
	g.SendMsg(cmd.GetPlayerSocialDetailRsp, player.PlayerId, player.ClientSeq, getPlayerSocialDetailRsp)
}

func (g *Game) SetPlayerBirthdayReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SetPlayerBirthdayReq)
	dbSocial := player.GetDbSocial()
	if dbSocial.IsSetBirthday() {
		g.SendError(cmd.SetPlayerBirthdayRsp, player, &proto.SetPlayerBirthdayRsp{})
		return
	}
	birthday := req.Birthday
	dbSocial.SetBirthday(birthday.Month, birthday.Day)
	g.SendMsg(cmd.SetPlayerBirthdayRsp, player.PlayerId, player.ClientSeq, &proto.SetPlayerBirthdayRsp{Birthday: req.Birthday})
}

func (g *Game) SetNameCardReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SetNameCardReq)
	nameCardId := req.NameCardId
	dbSocial := player.GetDbSocial()
	ok := dbSocial.UseNameCard(nameCardId)
	if !ok {
		logger.Error("name card not exist, uid: %v", player.PlayerId)
		return
	}
	g.SendMsg(cmd.SetNameCardRsp, player.PlayerId, player.ClientSeq, &proto.SetNameCardRsp{NameCardId: nameCardId})
}

func (g *Game) SetPlayerSignatureReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SetPlayerSignatureReq)
	signature := req.Signature

	setPlayerSignatureRsp := new(proto.SetPlayerSignatureRsp)
	if !object.IsUtf8String(signature) {
		setPlayerSignatureRsp.Retcode = int32(proto.Retcode_RET_SIGNATURE_ILLEGAL)
	} else if utf8.RuneCountInString(signature) > 50 {
		setPlayerSignatureRsp.Retcode = int32(proto.Retcode_RET_SIGNATURE_ILLEGAL)
	} else {
		player.Signature = signature
		setPlayerSignatureRsp.Signature = player.Signature
	}
	g.SendMsg(cmd.SetPlayerSignatureRsp, player.PlayerId, player.ClientSeq, setPlayerSignatureRsp)
}

func (g *Game) SetPlayerNameReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SetPlayerNameReq)
	nickName := req.NickName

	setPlayerNameRsp := new(proto.SetPlayerNameRsp)
	if len(nickName) == 0 {
		setPlayerNameRsp.Retcode = int32(proto.Retcode_RET_NICKNAME_IS_EMPTY)
	} else if !object.IsUtf8String(nickName) {
		setPlayerNameRsp.Retcode = int32(proto.Retcode_RET_NICKNAME_UTF8_ERROR)
	} else if utf8.RuneCountInString(nickName) > 14 {
		setPlayerNameRsp.Retcode = int32(proto.Retcode_RET_NICKNAME_TOO_LONG)
	} else if len(regexp.MustCompile(`\d`).FindAllString(nickName, -1)) > 6 {
		setPlayerNameRsp.Retcode = int32(proto.Retcode_RET_NICKNAME_TOO_MANY_DIGITS)
	} else {
		player.NickName = nickName
		setPlayerNameRsp.NickName = player.NickName
	}
	g.SendMsg(cmd.SetPlayerNameRsp, player.PlayerId, player.ClientSeq, setPlayerNameRsp)
}

func (g *Game) SetPlayerHeadImageReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SetPlayerHeadImageReq)
	avatarId := req.AvatarId
	dbAvatar := player.GetDbAvatar()
	avatar := dbAvatar.GetAvatarById(avatarId)
	if avatar == nil {
		logger.Error("the head img of the avatar not exist, uid: %v", player.PlayerId)
		return
	}
	player.HeadImage = avatarId

	setPlayerHeadImageRsp := &proto.SetPlayerHeadImageRsp{
		ProfilePicture: &proto.ProfilePicture{AvatarId: player.HeadImage},
	}
	g.SendMsg(cmd.SetPlayerHeadImageRsp, player.PlayerId, player.ClientSeq, setPlayerHeadImageRsp)
}

func (g *Game) GetAllUnlockNameCardReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetAllUnlockNameCardReq)
	_ = req
	dbSocial := player.GetDbSocial()
	g.SendMsg(cmd.GetAllUnlockNameCardRsp, player.PlayerId, player.ClientSeq, &proto.GetAllUnlockNameCardRsp{NameCardList: dbSocial.NameCardList})
}

func (g *Game) GetPlayerFriendListReq(player *model.Player, payloadMsg pb.Message) {
	getPlayerFriendListRsp := &proto.GetPlayerFriendListRsp{
		FriendList: make([]*proto.FriendBrief, 0),
	}

	// 添加好友到列表
	addFriendListFunc := func(uid uint32) {
		friendPlayer, online, _ := USER_MANAGER.LoadGlobalPlayer(uid)
		if friendPlayer == nil {
			logger.Error("target player is nil, uid: %v", player.PlayerId)
			return
		}
		var onlineState proto.FriendOnlineState = 0
		if online {
			onlineState = proto.FriendOnlineState_FRIEND_ONLINE
		} else {
			onlineState = proto.FriendOnlineState_FREIEND_DISCONNECT
		}
		friendBrief := &proto.FriendBrief{
			Uid:               friendPlayer.PlayerId,
			Nickname:          friendPlayer.NickName,
			Level:             friendPlayer.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
			ProfilePicture:    &proto.ProfilePicture{AvatarId: friendPlayer.HeadImage},
			WorldLevel:        friendPlayer.PropMap[constant.PLAYER_PROP_PLAYER_WORLD_LEVEL],
			Signature:         friendPlayer.Signature,
			OnlineState:       onlineState,
			IsMpModeAvailable: true,
			LastActiveTime:    player.OfflineTime,
			NameCardId:        friendPlayer.GetDbSocial().NameCard,
			Param:             (uint32(time.Now().Unix()) - player.OfflineTime) / 3600 / 24,
			IsGameSource:      true,
			PlatformType:      proto.PlatformType_PC,
		}
		getPlayerFriendListRsp.FriendList = append(getPlayerFriendListRsp.FriendList, friendBrief)
	}
	dbSocial := player.GetDbSocial()
	for uid := range dbSocial.FriendList {
		addFriendListFunc(uid)
	}
	// 命令管理器还需添加机器人的好友
	// 这样做是为了不修改用户好友列表的数据
	addFriendListFunc(COMMAND_MANAGER.system.PlayerId)

	g.SendMsg(cmd.GetPlayerFriendListRsp, player.PlayerId, player.ClientSeq, getPlayerFriendListRsp)
}

func (g *Game) GetPlayerAskFriendListReq(player *model.Player, payloadMsg pb.Message) {
	getPlayerAskFriendListRsp := &proto.GetPlayerAskFriendListRsp{
		AskFriendList: make([]*proto.FriendBrief, 0),
	}
	dbSocial := player.GetDbSocial()
	for uid := range dbSocial.FriendApplyList {
		friendPlayer, online, _ := USER_MANAGER.LoadGlobalPlayer(uid)
		if friendPlayer == nil {
			logger.Error("target player is nil, uid: %v", player.PlayerId)
			continue
		}
		var onlineState proto.FriendOnlineState
		if online {
			onlineState = proto.FriendOnlineState_FRIEND_ONLINE
		} else {
			onlineState = proto.FriendOnlineState_FREIEND_DISCONNECT
		}
		friendBrief := &proto.FriendBrief{
			Uid:               friendPlayer.PlayerId,
			Nickname:          friendPlayer.NickName,
			Level:             friendPlayer.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
			ProfilePicture:    &proto.ProfilePicture{AvatarId: friendPlayer.HeadImage},
			WorldLevel:        friendPlayer.PropMap[constant.PLAYER_PROP_PLAYER_WORLD_LEVEL],
			Signature:         friendPlayer.Signature,
			OnlineState:       onlineState,
			IsMpModeAvailable: true,
			LastActiveTime:    player.OfflineTime,
			NameCardId:        friendPlayer.GetDbSocial().NameCard,
			Param:             (uint32(time.Now().Unix()) - player.OfflineTime) / 3600 / 24,
			IsGameSource:      true,
			PlatformType:      proto.PlatformType_PC,
		}
		getPlayerAskFriendListRsp.AskFriendList = append(getPlayerAskFriendListRsp.AskFriendList, friendBrief)
	}
	g.SendMsg(cmd.GetPlayerAskFriendListRsp, player.PlayerId, player.ClientSeq, getPlayerAskFriendListRsp)
}

func (g *Game) AskAddFriendReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AskAddFriendReq)
	targetUid := req.TargetUid

	askAddFriendRsp := &proto.AskAddFriendRsp{
		TargetUid: targetUid,
	}
	g.SendMsg(cmd.AskAddFriendRsp, player.PlayerId, player.ClientSeq, askAddFriendRsp)

	targetPlayer := USER_MANAGER.GetOnlineUser(targetUid)
	if targetPlayer == nil {
		// 非本地玩家
		if USER_MANAGER.GetRemoteUserOnlineState(targetUid) {
			// 远程在线玩家
			gsAppId := USER_MANAGER.GetRemoteUserGsAppId(targetUid)
			g.messageQueue.SendToGs(gsAppId, &mq.NetMsg{
				MsgType: mq.MsgTypeServer,
				EventId: mq.ServerAddFriendNotify,
				ServerMsg: &mq.ServerMsg{
					AddFriendInfo: &mq.AddFriendInfo{
						OriginInfo: &mq.OriginInfo{
							CmdName: "AskAddFriendReq",
							UserId:  player.PlayerId,
						},
						TargetUserId: targetUid,
						ApplyPlayerOnlineInfo: &mq.PlayerBaseInfo{
							UserId:      player.PlayerId,
							Nickname:    player.NickName,
							PlayerLevel: player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
							NameCardId:  player.GetDbSocial().NameCard,
							Signature:   player.Signature,
							HeadImageId: player.HeadImage,
							WorldLevel:  player.PropMap[constant.PLAYER_PROP_PLAYER_WORLD_LEVEL],
						},
					},
				},
			})
		} else {
			// 全服离线玩家
			targetPlayer = USER_MANAGER.LoadTempOfflineUser(targetUid, true)
			if targetPlayer == nil {
				logger.Error("apply add friend target player is nil, uid: %v", targetUid)
				return
			}
			targetDbSocial := targetPlayer.GetDbSocial()
			if targetDbSocial.IsFriend(player.PlayerId) {
				logger.Error("friend or apply already exist, uid: %v", player.PlayerId)
				return
			}
			targetDbSocial.AddFriendApply(player.PlayerId)
			USER_MANAGER.SaveTempOfflineUser(targetPlayer)
		}
		return
	}

	targetDbSocial := targetPlayer.GetDbSocial()
	if targetDbSocial.IsFriend(player.PlayerId) {
		logger.Error("friend or apply already exist, uid: %v", player.PlayerId)
		return
	}
	targetDbSocial.AddFriendApply(player.PlayerId)

	// 目标玩家在线则通知
	askAddFriendNotify := &proto.AskAddFriendNotify{
		TargetUid: player.PlayerId,
	}
	askAddFriendNotify.TargetFriendBrief = &proto.FriendBrief{
		Uid:               player.PlayerId,
		Nickname:          player.NickName,
		Level:             player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
		ProfilePicture:    &proto.ProfilePicture{AvatarId: player.HeadImage},
		WorldLevel:        player.PropMap[constant.PLAYER_PROP_PLAYER_WORLD_LEVEL],
		Signature:         player.Signature,
		OnlineState:       proto.FriendOnlineState_FRIEND_ONLINE,
		IsMpModeAvailable: true,
		LastActiveTime:    player.OfflineTime,
		NameCardId:        player.GetDbSocial().NameCard,
		Param:             (uint32(time.Now().Unix()) - player.OfflineTime) / 3600 / 24,
		IsGameSource:      true,
		PlatformType:      proto.PlatformType_PC,
	}
	g.SendMsg(cmd.AskAddFriendNotify, targetPlayer.PlayerId, targetPlayer.ClientSeq, askAddFriendNotify)
}

func (g *Game) DealAddFriendReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.DealAddFriendReq)
	targetUid := req.TargetUid
	result := req.DealAddFriendResult

	agree := false
	if result == proto.DealAddFriendResultType_DEAL_ADD_FRIEND_ACCEPT {
		agree = true
	}
	dbSocial := player.GetDbSocial()
	if agree {
		dbSocial.AddFriend(targetUid)
	}
	dbSocial.DelFriendApply(targetUid)

	dealAddFriendRsp := &proto.DealAddFriendRsp{
		TargetUid:           targetUid,
		DealAddFriendResult: result,
	}
	g.SendMsg(cmd.DealAddFriendRsp, player.PlayerId, player.ClientSeq, dealAddFriendRsp)

	if agree {
		targetPlayer := USER_MANAGER.GetOnlineUser(targetUid)
		if targetPlayer == nil {
			// 非本地玩家
			if USER_MANAGER.GetRemoteUserOnlineState(targetUid) {
				// 远程在线玩家
				gsAppId := USER_MANAGER.GetRemoteUserGsAppId(targetUid)
				g.messageQueue.SendToGs(gsAppId, &mq.NetMsg{
					MsgType: mq.MsgTypeServer,
					EventId: mq.ServerAddFriendNotify,
					ServerMsg: &mq.ServerMsg{
						AddFriendInfo: &mq.AddFriendInfo{
							OriginInfo: &mq.OriginInfo{
								CmdName: "DealAddFriendReq",
								UserId:  player.PlayerId,
							},
							TargetUserId: targetUid,
							ApplyPlayerOnlineInfo: &mq.PlayerBaseInfo{
								UserId: player.PlayerId,
							},
						},
					},
				})
			} else {
				// 全服离线玩家
				targetPlayer = USER_MANAGER.LoadTempOfflineUser(targetUid, true)
				targetDbSocial := targetPlayer.GetDbSocial()
				if targetPlayer == nil {
					logger.Error("apply add friend target player is nil, uid: %v", targetUid)
					return
				}
				targetDbSocial.AddFriend(player.PlayerId)
				USER_MANAGER.SaveTempOfflineUser(targetPlayer)
			}
			return
		}
		targetDbSocial := targetPlayer.GetDbSocial()
		targetDbSocial.AddFriend(player.PlayerId)
	}
}

func (g *Game) GetOnlinePlayerListReq(player *model.Player, payloadMsg pb.Message) {
	count := 0
	rsp := &proto.GetOnlinePlayerListRsp{
		PlayerInfoList: make([]*proto.OnlinePlayerInfo, 0),
	}
	// 最先添加全服ai玩家
	aiUidList := USER_MANAGER.GetAllRemoteAiUidList()
	aiUidList = append(aiUidList, g.GetAi().PlayerId)
	for _, aiUid := range aiUidList {
		aiGsId := aiUid - AiBaseUid
		roomNumber := aiGsId - 1
		startMinute := roomNumber % 6 * 10
		name := fmt.Sprintf("房间：%v", roomNumber)
		sign := fmt.Sprintf("开启时间：%02d:%02d。", time.Now().Hour(), startMinute)
		rsp.PlayerInfoList = append(rsp.PlayerInfoList, &proto.OnlinePlayerInfo{
			Uid:                 aiUid,
			Nickname:            name,
			PlayerLevel:         1,
			AvatarId:            10000007,
			MpSettingType:       proto.MpSettingType_MP_SETTING_ENTER_AFTER_APPLY,
			NameCardId:          210001,
			Signature:           sign,
			ProfilePicture:      &proto.ProfilePicture{AvatarId: 10000007},
			CurPlayerNumInWorld: 1,
		})
		count++
	}
	onlinePlayerList := make([]*model.Player, 0)
	// 优先获取本地的在线玩家
	for _, onlinePlayer := range USER_MANAGER.GetAllOnlineUserList() {
		if onlinePlayer.PlayerId < PlayerBaseUid || onlinePlayer.PlayerId > MaxPlayerBaseUid {
			continue
		}
		if onlinePlayer.PlayerId == player.PlayerId {
			continue
		}
		onlinePlayerList = append(onlinePlayerList, onlinePlayer)
		count++
		if count >= 50 {
			break
		}
	}
	if count < 50 {
		// 本地不够时获取远程的在线玩家
		for _, onlinePlayer := range USER_MANAGER.GetRemoteOnlineUserList(50 - count) {
			if onlinePlayer.PlayerId == player.PlayerId {
				continue
			}
			onlinePlayerList = append(onlinePlayerList, onlinePlayer)
			count++
			if count >= 50 {
				break
			}
		}
	}

	for _, onlinePlayer := range onlinePlayerList {
		onlinePlayerInfo := g.PacketOnlinePlayerInfo(onlinePlayer)
		rsp.PlayerInfoList = append(rsp.PlayerInfoList, onlinePlayerInfo)
	}
	g.SendMsg(cmd.GetOnlinePlayerListRsp, player.PlayerId, player.ClientSeq, rsp)
}

func (g *Game) GetOnlinePlayerInfoReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetOnlinePlayerInfoReq)
	targetUid, ok := req.PlayerId.(*proto.GetOnlinePlayerInfoReq_TargetUid)
	if !ok {
		return
	}

	targetPlayer, online, _ := USER_MANAGER.LoadGlobalPlayer(targetUid.TargetUid)
	if targetPlayer == nil || !online {
		g.SendError(cmd.GetOnlinePlayerInfoRsp, player, &proto.GetOnlinePlayerInfoRsp{}, proto.Retcode_RET_PLAYER_NOT_ONLINE)
		return
	}

	g.SendMsg(cmd.GetOnlinePlayerInfoRsp, player.PlayerId, player.ClientSeq, &proto.GetOnlinePlayerInfoRsp{
		TargetUid:        targetUid.TargetUid,
		TargetPlayerInfo: g.PacketOnlinePlayerInfo(targetPlayer),
	})
}

func (g *Game) GetPlayerBlacklistReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetPlayerBlacklistReq)
	_ = req
	g.SendMsg(cmd.GetPlayerBlacklistRsp, player.PlayerId, player.ClientSeq, new(proto.GetPlayerBlacklistRsp))
}

/************************************************** 游戏功能 **************************************************/

// 跨服添加好友通知

func (g *Game) ServerAddFriendNotify(addFriendInfo *mq.AddFriendInfo) {
	switch addFriendInfo.OriginInfo.CmdName {
	case "AskAddFriendReq":
		targetPlayer := USER_MANAGER.GetOnlineUser(addFriendInfo.TargetUserId)
		if targetPlayer == nil {
			logger.Error("player is nil, uid: %v", addFriendInfo.TargetUserId)
			return
		}
		targetDbSocial := targetPlayer.GetDbSocial()
		if targetDbSocial.IsFriend(addFriendInfo.ApplyPlayerOnlineInfo.UserId) {
			logger.Error("friend or apply already exist, uid: %v", addFriendInfo.ApplyPlayerOnlineInfo.UserId)
			return
		}
		targetDbSocial.AddFriendApply(addFriendInfo.ApplyPlayerOnlineInfo.UserId)

		// 目标玩家在线则通知
		askAddFriendNotify := &proto.AskAddFriendNotify{
			TargetUid: addFriendInfo.ApplyPlayerOnlineInfo.UserId,
		}
		askAddFriendNotify.TargetFriendBrief = &proto.FriendBrief{
			Uid:               addFriendInfo.ApplyPlayerOnlineInfo.UserId,
			Nickname:          addFriendInfo.ApplyPlayerOnlineInfo.Nickname,
			Level:             addFriendInfo.ApplyPlayerOnlineInfo.PlayerLevel,
			ProfilePicture:    &proto.ProfilePicture{AvatarId: addFriendInfo.ApplyPlayerOnlineInfo.HeadImageId},
			WorldLevel:        addFriendInfo.ApplyPlayerOnlineInfo.WorldLevel,
			Signature:         addFriendInfo.ApplyPlayerOnlineInfo.Signature,
			OnlineState:       proto.FriendOnlineState_FRIEND_ONLINE,
			IsMpModeAvailable: true,
			LastActiveTime:    0,
			NameCardId:        addFriendInfo.ApplyPlayerOnlineInfo.NameCardId,
			Param:             0,
			IsGameSource:      true,
			PlatformType:      proto.PlatformType_PC,
		}
		g.SendMsg(cmd.AskAddFriendNotify, targetPlayer.PlayerId, targetPlayer.ClientSeq, askAddFriendNotify)
	case "DealAddFriendReq":
		targetPlayer := USER_MANAGER.GetOnlineUser(addFriendInfo.TargetUserId)
		if targetPlayer == nil {
			logger.Error("player is nil, uid: %v", addFriendInfo.TargetUserId)
			return
		}
		targetDbSocial := targetPlayer.GetDbSocial()
		targetDbSocial.AddFriend(addFriendInfo.ApplyPlayerOnlineInfo.UserId)
	}
}

/************************************************** 打包封装 **************************************************/

func (g *Game) PacketOnlinePlayerInfo(player *model.Player) *proto.OnlinePlayerInfo {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	worldPlayerNum := uint32(0)
	if world != nil {
		worldPlayerNum = uint32(world.GetWorldPlayerNum())
	} else {
		worldPlayerNum = player.RemoteWorldPlayerNum
	}
	onlinePlayerInfo := &proto.OnlinePlayerInfo{
		Uid:                 player.PlayerId,
		Nickname:            player.NickName,
		PlayerLevel:         player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
		AvatarId:            player.HeadImage,
		MpSettingType:       proto.MpSettingType(player.PropMap[constant.PLAYER_PROP_PLAYER_MP_SETTING_TYPE]),
		NameCardId:          player.GetDbSocial().NameCard,
		Signature:           player.Signature,
		ProfilePicture:      &proto.ProfilePicture{AvatarId: player.HeadImage},
		CurPlayerNumInWorld: worldPlayerNum,
	}
	return onlinePlayerInfo
}
