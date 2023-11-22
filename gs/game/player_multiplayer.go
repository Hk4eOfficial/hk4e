package game

import (
	"time"

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

// PlayerApplyEnterMpReq 世界敲门请求
func (g *Game) PlayerApplyEnterMpReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PlayerApplyEnterMpReq)
	rsp := &proto.PlayerApplyEnterMpRsp{TargetUid: req.TargetUid}
	g.SendMsg(cmd.PlayerApplyEnterMpRsp, player.PlayerId, player.ClientSeq, rsp)
	g.PlayerApplyEnterWorld(player, req.TargetUid)
}

// PlayerApplyEnterMpResultReq 世界敲门处理请求
func (g *Game) PlayerApplyEnterMpResultReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PlayerApplyEnterMpResultReq)
	rsp := &proto.PlayerApplyEnterMpResultRsp{
		ApplyUid: req.ApplyUid,
		IsAgreed: req.IsAgreed,
	}
	g.SendMsg(cmd.PlayerApplyEnterMpResultRsp, player.PlayerId, player.ClientSeq, rsp)
	g.PlayerDealEnterWorld(player, req.ApplyUid, req.IsAgreed)
}

// PlayerGetForceQuitBanInfoReq 获取强退禁令信息请求
func (g *Game) PlayerGetForceQuitBanInfoReq(player *model.Player, payloadMsg pb.Message) {
	ok := true
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	for _, worldPlayer := range world.GetAllPlayer() {
		if worldPlayer.SceneLoadState != model.SceneEnterDone {
			ok = false
		}
	}
	if !ok {
		g.SendError(cmd.PlayerGetForceQuitBanInfoRsp, player, &proto.PlayerGetForceQuitBanInfoRsp{}, proto.Retcode_RET_MP_TARGET_PLAYER_IN_TRANSFER)
		return
	}
	g.SendSucc(cmd.PlayerGetForceQuitBanInfoRsp, player, &proto.PlayerGetForceQuitBanInfoRsp{})
}

// BackMyWorldReq 返回单人世界请求
func (g *Game) BackMyWorldReq(player *model.Player, payloadMsg pb.Message) {
	// 其他玩家
	ok := g.PlayerLeaveWorld(player, false, proto.PlayerQuitFromMpNotify_BACK_TO_MY_WORLD)
	if !ok {
		g.SendError(cmd.BackMyWorldRsp, player, &proto.BackMyWorldRsp{}, proto.Retcode_RET_MP_TARGET_PLAYER_IN_TRANSFER)
		return
	}
	g.SendSucc(cmd.BackMyWorldRsp, player, &proto.BackMyWorldRsp{})
}

// ChangeWorldToSingleModeReq 转换单人模式请求
func (g *Game) ChangeWorldToSingleModeReq(player *model.Player, payloadMsg pb.Message) {
	// 房主
	ok := g.PlayerLeaveWorld(player, false, proto.PlayerQuitFromMpNotify_HOST_NO_OTHER_PLAYER)
	if !ok {
		g.SendError(cmd.ChangeWorldToSingleModeRsp, player, &proto.ChangeWorldToSingleModeRsp{}, proto.Retcode_RET_MP_TARGET_PLAYER_IN_TRANSFER)
		return
	}
	g.SendSucc(cmd.ChangeWorldToSingleModeRsp, player, &proto.ChangeWorldToSingleModeRsp{})
}

// SceneKickPlayerReq 剔除玩家请求
func (g *Game) SceneKickPlayerReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SceneKickPlayerReq)
	targetUid := req.TargetUid
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	if player.PlayerId != world.GetOwner().PlayerId {
		g.SendError(cmd.SceneKickPlayerRsp, player, &proto.SceneKickPlayerRsp{})
		return
	}
	targetPlayer := USER_MANAGER.GetOnlineUser(targetUid)
	if targetPlayer == nil {
		logger.Error("player is nil, uid: %v", targetUid)
		return
	}
	ok := g.PlayerLeaveWorld(targetPlayer, false, proto.PlayerQuitFromMpNotify_KICK_BY_HOST)
	if !ok {
		g.SendError(cmd.SceneKickPlayerRsp, player, &proto.SceneKickPlayerRsp{}, proto.Retcode_RET_MP_TARGET_PLAYER_IN_TRANSFER)
		return
	}
	ntf := &proto.SceneKickPlayerNotify{
		TargetUid: targetUid,
		KickerUid: player.PlayerId,
	}
	for _, worldPlayer := range world.GetAllPlayer() {
		g.SendMsg(cmd.SceneKickPlayerNotify, worldPlayer.PlayerId, worldPlayer.ClientSeq, ntf)
	}
	rsp := &proto.SceneKickPlayerRsp{
		TargetUid: targetUid,
	}
	g.SendMsg(cmd.SceneKickPlayerRsp, player.PlayerId, player.ClientSeq, rsp)
}

// JoinPlayerSceneReq 进入他人世界请求
func (g *Game) JoinPlayerSceneReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.JoinPlayerSceneReq)
	rsp := &proto.JoinPlayerSceneRsp{
		Retcode: int32(proto.Retcode_RET_JOIN_OTHER_WAIT),
	}
	g.SendMsg(cmd.JoinPlayerSceneRsp, player.PlayerId, player.ClientSeq, rsp)

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	g.WorldRemovePlayer(world, player)

	g.SendMsg(cmd.LeaveWorldNotify, player.PlayerId, player.ClientSeq, new(proto.LeaveWorldNotify))

	hostPlayer := USER_MANAGER.GetOnlineUser(req.TargetUid)
	if hostPlayer == nil {
		// 要加入的世界属于非本地玩家
		if !USER_MANAGER.GetRemoteUserOnlineState(req.TargetUid) {
			// 全服不存在该在线玩家
			logger.Error("target player not online in any game server, uid: %v", req.TargetUid)
			return
		}
		// 走玩家在线跨服迁移流程
		g.OnOffline(player.PlayerId, &ChangeGsInfo{
			IsChangeGs:     true,
			JoinHostUserId: req.TargetUid,
		})
		return
	}

	g.LoginNotify(player.PlayerId, player.ClientSeq, player)

	g.JoinOtherWorld(player, hostPlayer)
}

// PlayerStartMatchReq 开始匹配请求
func (g *Game) PlayerStartMatchReq(player *model.Player, payloadMsg pb.Message) {
}

// PlayerCancelMatchReq 取消匹配请求
func (g *Game) PlayerCancelMatchReq(player *model.Player, payloadMsg pb.Message) {
}

// PlayerConfirmMatchReq 确认匹配请求
func (g *Game) PlayerConfirmMatchReq(player *model.Player, payloadMsg pb.Message) {
}

/************************************************** 游戏功能 **************************************************/

func (g *Game) PlayerApplyEnterWorld(player *model.Player, targetUid uint32) {
	applyFailNotify := func(reason proto.PlayerApplyEnterMpResultNotify_Reason) {
		playerApplyEnterMpResultNotify := &proto.PlayerApplyEnterMpResultNotify{
			TargetUid:      targetUid,
			TargetNickname: "",
			IsAgreed:       false,
			Reason:         reason,
		}
		g.SendMsg(cmd.PlayerApplyEnterMpResultNotify, player.PlayerId, player.ClientSeq, playerApplyEnterMpResultNotify)
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	if world.IsMultiplayerWorld() {
		applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
		return
	}
	targetPlayer := USER_MANAGER.GetOnlineUser(targetUid)
	if targetPlayer == nil {
		if !USER_MANAGER.GetRemoteUserOnlineState(targetUid) {
			// 全服不存在该在线玩家
			logger.Error("target player not online in any game server, uid: %v", targetUid)
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
			return
		}
		gsAppId := USER_MANAGER.GetRemoteUserGsAppId(targetUid)
		g.messageQueue.SendToGs(gsAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeServer,
			EventId: mq.ServerPlayerMpReq,
			ServerMsg: &mq.ServerMsg{
				PlayerMpInfo: &mq.PlayerMpInfo{
					OriginInfo: &mq.OriginInfo{
						CmdName: "PlayerApplyEnterMpReq",
						UserId:  player.PlayerId,
					},
					HostUserId:  targetUid,
					ApplyUserId: player.PlayerId,
					ApplyPlayerOnlineInfo: &mq.PlayerBaseInfo{
						UserId:         player.PlayerId,
						Nickname:       player.NickName,
						PlayerLevel:    player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
						MpSettingType:  uint8(player.PropMap[constant.PLAYER_PROP_PLAYER_MP_SETTING_TYPE]),
						NameCardId:     player.GetDbSocial().NameCard,
						Signature:      player.Signature,
						HeadImageId:    player.HeadImage,
						WorldPlayerNum: uint32(world.GetWorldPlayerNum()),
					},
				},
			},
		})
		return
	}
	if WORLD_MANAGER.GetMultiplayerWorldNum() >= MAX_MULTIPLAYER_WORLD_NUM {
		// 超过本服务器最大多人世界数量限制
		applyFailNotify(proto.PlayerApplyEnterMpResultNotify_MAX_PLAYER)
		return
	}
	targetWorld := WORLD_MANAGER.GetWorldById(targetPlayer.WorldId)
	if targetWorld == nil {
		// 目标玩家世界状态异常
		logger.Error("target world is nil, worldId: %v, uid: %v", targetPlayer.WorldId, player.PlayerId)
		applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
		return
	}
	if targetWorld.IsMultiplayerWorld() && targetWorld.GetOwner().PlayerId != targetPlayer.PlayerId {
		// 向同一世界内的非房主玩家申请时直接拒绝
		applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
		return
	}
	mpSetting := targetPlayer.PropMap[constant.PLAYER_PROP_PLAYER_MP_SETTING_TYPE]
	if mpSetting == 0 {
		// 房主玩家没开权限
		applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
		return
	} else if mpSetting == 1 {
		g.PlayerDealEnterWorld(targetPlayer, player.PlayerId, true)
		return
	}
	applyTime, exist := targetPlayer.CoopApplyMap[player.PlayerId]
	if exist && time.Now().UnixNano() < applyTime+int64(10*time.Second) {
		// 申请过期
		applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
		return
	}
	targetPlayer.CoopApplyMap[player.PlayerId] = time.Now().UnixNano()

	playerApplyEnterMpNotify := new(proto.PlayerApplyEnterMpNotify)
	playerApplyEnterMpNotify.SrcPlayerInfo = g.PacketOnlinePlayerInfo(player)
	g.SendMsg(cmd.PlayerApplyEnterMpNotify, targetPlayer.PlayerId, targetPlayer.ClientSeq, playerApplyEnterMpNotify)
}

func (g *Game) PlayerDealEnterWorld(hostPlayer *model.Player, otherUid uint32, agree bool) {
	applyTime, exist := hostPlayer.CoopApplyMap[otherUid]
	if !exist || time.Now().UnixNano() > applyTime+int64(10*time.Second) {
		return
	}
	delete(hostPlayer.CoopApplyMap, otherUid)
	if !agree {
		return
	}
	g.HostEnterMpWorld(hostPlayer)

	otherPlayer := USER_MANAGER.GetOnlineUser(otherUid)
	if otherPlayer == nil {
		if !USER_MANAGER.GetRemoteUserOnlineState(otherUid) {
			// 全服不存在该在线玩家
			logger.Error("target player not online in any game server, uid: %v", otherUid)
			return
		}
		gsAppId := USER_MANAGER.GetRemoteUserGsAppId(otherUid)
		g.messageQueue.SendToGs(gsAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeServer,
			EventId: mq.ServerPlayerMpReq,
			ServerMsg: &mq.ServerMsg{
				PlayerMpInfo: &mq.PlayerMpInfo{
					OriginInfo: &mq.OriginInfo{
						CmdName: "PlayerApplyEnterMpResultReq",
						UserId:  hostPlayer.PlayerId,
					},
					HostUserId:   hostPlayer.PlayerId,
					ApplyUserId:  otherUid,
					Agreed:       agree,
					HostNickname: hostPlayer.NickName,
				},
			},
		})
		return
	}

	otherPlayerWorld := WORLD_MANAGER.GetWorldById(otherPlayer.WorldId)
	if otherPlayerWorld == nil || otherPlayerWorld.IsMultiplayerWorld() {
		playerApplyEnterMpResultNotify := &proto.PlayerApplyEnterMpResultNotify{
			TargetUid:      hostPlayer.PlayerId,
			TargetNickname: hostPlayer.NickName,
			IsAgreed:       false,
			Reason:         proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP,
		}
		g.SendMsg(cmd.PlayerApplyEnterMpResultNotify, otherPlayer.PlayerId, otherPlayer.ClientSeq, playerApplyEnterMpResultNotify)
		return
	}

	playerApplyEnterMpResultNotify := &proto.PlayerApplyEnterMpResultNotify{
		TargetUid:      hostPlayer.PlayerId,
		TargetNickname: hostPlayer.NickName,
		IsAgreed:       agree,
		Reason:         proto.PlayerApplyEnterMpResultNotify_PLAYER_JUDGE,
	}
	g.SendMsg(cmd.PlayerApplyEnterMpResultNotify, otherPlayer.PlayerId, otherPlayer.ClientSeq, playerApplyEnterMpResultNotify)
}

func (g *Game) PlayerLeaveWorld(player *model.Player, force bool, reason proto.PlayerQuitFromMpNotify_QuitReason) bool {
	if force {
		g.SendMsg(cmd.PlayerQuitFromMpNotify, player.PlayerId, player.ClientSeq, &proto.PlayerQuitFromMpNotify{Reason: reason})
		g.ReLoginPlayer(player.PlayerId, true)
		return true
	}
	oldWorld := WORLD_MANAGER.GetWorldById(player.WorldId)
	if oldWorld == nil {
		logger.Error("world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return false
	}
	if !oldWorld.IsMultiplayerWorld() {
		return false
	}
	for _, worldPlayer := range oldWorld.GetAllPlayer() {
		if worldPlayer.SceneLoadState != model.SceneEnterDone {
			return false
		}
	}
	g.SendMsg(cmd.PlayerQuitFromMpNotify, player.PlayerId, player.ClientSeq, &proto.PlayerQuitFromMpNotify{Reason: reason})
	g.ReLoginPlayer(player.PlayerId, true)
	return true
}

func (g *Game) JoinOtherWorld(player *model.Player, hostPlayer *model.Player) {
	hostWorld := WORLD_MANAGER.GetWorldById(hostPlayer.WorldId)
	if hostWorld == nil {
		logger.Error("host world is nil, worldId: %v, uid: %v", hostPlayer.WorldId, player.PlayerId)
		return
	}
	if hostPlayer.SceneLoadState == model.SceneEnterDone {
		player.SceneJump = true
		player.SceneLoadState = model.SceneNone
		player.SceneEnterReason = uint32(proto.EnterReason_ENTER_REASON_TEAM_JOIN)
		player.IsInMp = hostWorld.IsMultiplayerWorld()
		player.SetSceneId(hostPlayer.GetSceneId())
		if WORLD_MANAGER.IsAiWorld(hostWorld) {
			player.SetPos(&model.Vector{X: 500.0, Y: 900.0, Z: -500.0})
			player.SetRot(new(model.Vector))
		} else {
			player.SetPos(hostPlayer.GetPos())
			player.SetRot(hostPlayer.GetRot())
		}
		g.WorldAddPlayer(hostWorld, player)

		enterSceneToken := hostWorld.AddEnterSceneContext(&EnterSceneContext{
			OldSceneId: 0,
			Uid:        player.PlayerId,
		})

		playerEnterSceneNotify := g.PacketPlayerEnterSceneNotifyMp(
			player,
			hostPlayer,
			proto.EnterType_ENTER_OTHER,
			hostPlayer.GetSceneId(),
			hostPlayer.GetPos(),
			enterSceneToken,
		)
		g.SendMsg(cmd.PlayerEnterSceneNotify, player.PlayerId, player.ClientSeq, playerEnterSceneNotify)
	} else {
		hostWorld.AddWaitPlayer(player.PlayerId)
	}
}

func (g *Game) HostEnterMpWorld(hostPlayer *model.Player) {
	world := WORLD_MANAGER.GetWorldById(hostPlayer.WorldId)
	if world == nil || world.IsMultiplayerWorld() {
		return
	}

	enterSceneId := hostPlayer.GetSceneId()
	enterPos := hostPlayer.GetPos()
	enterRot := hostPlayer.GetRot()
	world.ChangeToMultiplayer()
	hostPlayer.SetSceneId(enterSceneId)
	hostPlayer.SetPos(enterPos)
	hostPlayer.SetRot(enterRot)

	worldDataNotify := &proto.WorldDataNotify{
		WorldPropMap: make(map[uint32]*proto.PropValue),
	}
	// 是否多人游戏
	worldDataNotify.WorldPropMap[2] = g.PacketPropValue(2, object.ConvBoolToInt64(world.IsMultiplayerWorld()))
	g.SendMsg(cmd.WorldDataNotify, hostPlayer.PlayerId, hostPlayer.ClientSeq, worldDataNotify)

	hostPlayer.SceneJump = false
	hostPlayer.SceneLoadState = model.SceneNone
	hostPlayer.SceneEnterReason = uint32(proto.EnterReason_ENTER_REASON_HOST_FROM_SINGLE_TO_MP)

	currPos := g.GetPlayerPos(hostPlayer)

	enterSceneToken := world.AddEnterSceneContext(&EnterSceneContext{
		OldSceneId:        enterSceneId,
		OldPos:            currPos,
		NewSceneId:        enterSceneId,
		NewPos:            enterPos,
		NewRot:            enterRot,
		OldDungeonPointId: 0,
		Uid:               hostPlayer.PlayerId,
	})

	hostPlayerEnterSceneNotify := g.PacketPlayerEnterSceneNotifyMp(
		hostPlayer,
		hostPlayer,
		proto.EnterType_ENTER_GOTO,
		enterSceneId,
		enterPos,
		enterSceneToken,
	)
	g.SendMsg(cmd.PlayerEnterSceneNotify, hostPlayer.PlayerId, hostPlayer.ClientSeq, hostPlayerEnterSceneNotify)
}

func (g *Game) WorldAddPlayer(world *World, player *model.Player) {
	if world.GetWorldPlayerNum() >= 4 && !WORLD_MANAGER.IsAiWorld(world) {
		return
	}
	playerMap := world.GetAllPlayer()
	_, exist := playerMap[player.PlayerId]
	if exist {
		return
	}
	world.AddPlayer(player)
	player.WorldId = world.GetId()
	if world.IsMultiplayerWorld() && world.GetWorldPlayerNum() > 1 {
		g.UpdateWorldPlayerInfo(world, player)
	}
	world.GetOwner().RemoteWorldPlayerNum = uint32(world.GetWorldPlayerNum())
}

func (g *Game) WorldRemovePlayer(world *World, player *model.Player) {
	if WORLD_MANAGER.IsAiWorld(world) {
		aiWorldAoi := world.GetAiWorldAoi()
		pos := g.GetPlayerPos(player)
		logger.Debug("ai world aoi remove player, oldPos: %+v, uid: %v", pos, player.PlayerId)
		ok := aiWorldAoi.RemoveObjectFromGridByPos(int64(player.PlayerId), float32(pos.X), float32(pos.Y), float32(pos.Z))
		if !ok {
			logger.Error("ai world aoi remove player fail, uid: %v, pos: %+v", player.PlayerId, pos)
		}
	}

	if world.IsMultiplayerWorld() && player.PlayerId == world.GetOwner().PlayerId {
		// 多人世界房主离开剔除所有其他玩家
		for _, worldPlayer := range world.GetAllPlayer() {
			if worldPlayer.PlayerId == world.GetOwner().PlayerId {
				continue
			}
			g.PlayerLeaveWorld(worldPlayer, true, proto.PlayerQuitFromMpNotify_KICK_BY_HOST_LOGOUT)
		}
	}
	scene := world.GetSceneById(player.GetSceneId())

	// 清除玩家的载具
	for vehicleId, entityId := range player.VehicleInfo.CreateEntityIdMap {
		g.DestroyVehicleEntity(player, scene, vehicleId, entityId)
	}

	entityIdList := make([]uint32, 0)
	for _, entity := range g.GetVisionEntity(scene, g.GetPlayerPos(player)) {
		entityIdList = append(entityIdList, entity.GetId())
	}
	g.RemoveSceneEntityNotifyToPlayer(player, proto.VisionType_VISION_MISS, entityIdList)

	delTeamEntityNotify := g.PacketDelTeamEntityNotify(world, player)
	g.SendMsg(cmd.DelTeamEntityNotify, player.PlayerId, player.ClientSeq, delTeamEntityNotify)

	if world.IsMultiplayerWorld() {
		activeAvatarEntity := world.GetPlayerActiveAvatarEntity(player)
		g.RemoveSceneEntityNotifyBroadcast(scene, proto.VisionType_VISION_REMOVE, []uint32{activeAvatarEntity.GetId()}, 0)
	}

	world.RemovePlayer(player)

	player.WorldId = 0
	if world.GetOwner().PlayerId == player.PlayerId {
		// 房主离开销毁世界
		WORLD_MANAGER.DestroyWorld(world.GetId())
		return
	}
	if world.IsMultiplayerWorld() && world.GetWorldPlayerNum() > 0 {
		g.UpdateWorldPlayerInfo(world, player)
		world.GetOwner().RemoteWorldPlayerNum = uint32(world.GetWorldPlayerNum())
	}
}

func (g *Game) UpdateWorldPlayerInfo(hostWorld *World, excludePlayer *model.Player) {
	for _, worldPlayer := range hostWorld.GetAllPlayer() {
		if worldPlayer.PlayerId == excludePlayer.PlayerId {
			continue
		}

		worldPlayerInfoNotify := &proto.WorldPlayerInfoNotify{
			PlayerInfoList: make([]*proto.OnlinePlayerInfo, 0),
			PlayerUidList:  make([]uint32, 0),
		}
		for _, subWorldPlayer := range hostWorld.GetAllPlayer() {
			onlinePlayerInfo := &proto.OnlinePlayerInfo{
				Uid:                 subWorldPlayer.PlayerId,
				Nickname:            subWorldPlayer.NickName,
				PlayerLevel:         subWorldPlayer.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
				AvatarId:            subWorldPlayer.HeadImage,
				MpSettingType:       proto.MpSettingType(subWorldPlayer.PropMap[constant.PLAYER_PROP_PLAYER_MP_SETTING_TYPE]),
				NameCardId:          subWorldPlayer.GetDbSocial().NameCard,
				Signature:           subWorldPlayer.Signature,
				ProfilePicture:      &proto.ProfilePicture{AvatarId: subWorldPlayer.HeadImage},
				CurPlayerNumInWorld: uint32(hostWorld.GetWorldPlayerNum()),
			}

			worldPlayerInfoNotify.PlayerInfoList = append(worldPlayerInfoNotify.PlayerInfoList, onlinePlayerInfo)
			worldPlayerInfoNotify.PlayerUidList = append(worldPlayerInfoNotify.PlayerUidList, subWorldPlayer.PlayerId)
		}
		g.SendMsg(cmd.WorldPlayerInfoNotify, worldPlayer.PlayerId, worldPlayer.ClientSeq, worldPlayerInfoNotify)

		serverTimeNotify := &proto.ServerTimeNotify{
			ServerTime: uint64(time.Now().UnixMilli()),
		}
		g.SendMsg(cmd.ServerTimeNotify, worldPlayer.PlayerId, worldPlayer.ClientSeq, serverTimeNotify)

		g.UpdateWorldScenePlayerInfo(worldPlayer, hostWorld)
	}
}

func (g *Game) UpdateWorldScenePlayerInfo(player *model.Player, world *World) {
	scenePlayerInfoNotify := &proto.ScenePlayerInfoNotify{
		PlayerInfoList: make([]*proto.ScenePlayerInfo, 0),
	}
	for _, worldPlayer := range world.GetAllPlayer() {
		onlinePlayerInfo := &proto.OnlinePlayerInfo{
			Uid:                 worldPlayer.PlayerId,
			Nickname:            worldPlayer.NickName,
			PlayerLevel:         worldPlayer.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL],
			AvatarId:            worldPlayer.HeadImage,
			MpSettingType:       proto.MpSettingType(worldPlayer.PropMap[constant.PLAYER_PROP_PLAYER_MP_SETTING_TYPE]),
			NameCardId:          worldPlayer.GetDbSocial().NameCard,
			Signature:           worldPlayer.Signature,
			ProfilePicture:      &proto.ProfilePicture{AvatarId: worldPlayer.HeadImage},
			CurPlayerNumInWorld: uint32(world.GetWorldPlayerNum()),
		}
		scenePlayerInfoNotify.PlayerInfoList = append(scenePlayerInfoNotify.PlayerInfoList, &proto.ScenePlayerInfo{
			Uid:              worldPlayer.PlayerId,
			PeerId:           world.GetPlayerPeerId(worldPlayer),
			Name:             worldPlayer.NickName,
			SceneId:          worldPlayer.GetSceneId(),
			OnlinePlayerInfo: onlinePlayerInfo,
		})
	}
	g.SendMsg(cmd.ScenePlayerInfoNotify, player.PlayerId, player.ClientSeq, scenePlayerInfoNotify)

	sceneTeamUpdateNotify := g.PacketSceneTeamUpdateNotify(world, player)
	g.SendMsg(cmd.SceneTeamUpdateNotify, player.PlayerId, player.ClientSeq, sceneTeamUpdateNotify)

	syncTeamEntityNotify := &proto.SyncTeamEntityNotify{
		SceneId:            player.GetSceneId(),
		TeamEntityInfoList: make([]*proto.TeamEntityInfo, 0),
	}
	if world.IsMultiplayerWorld() {
		for _, worldPlayer := range world.GetAllPlayer() {
			if worldPlayer.PlayerId == player.PlayerId {
				continue
			}
			teamEntityInfo := &proto.TeamEntityInfo{
				TeamEntityId:    world.GetPlayerTeamEntityId(worldPlayer),
				AuthorityPeerId: world.GetPlayerPeerId(worldPlayer),
				TeamAbilityInfo: new(proto.AbilitySyncStateInfo),
			}
			syncTeamEntityNotify.TeamEntityInfoList = append(syncTeamEntityNotify.TeamEntityInfoList, teamEntityInfo)
		}
	}
	g.SendMsg(cmd.SyncTeamEntityNotify, player.PlayerId, player.ClientSeq, syncTeamEntityNotify)

	syncScenePlayTeamEntityNotify := &proto.SyncScenePlayTeamEntityNotify{
		SceneId: player.GetSceneId(),
	}
	g.SendMsg(cmd.SyncScenePlayTeamEntityNotify, player.PlayerId, player.ClientSeq, syncScenePlayTeamEntityNotify)
}

// 跨服玩家多人世界相关请求

func (g *Game) ServerPlayerMpReq(playerMpInfo *mq.PlayerMpInfo, gsAppId string) {
	switch playerMpInfo.OriginInfo.CmdName {
	case "PlayerApplyEnterMpReq":
		applyFailNotify := func(reason proto.PlayerApplyEnterMpResultNotify_Reason) {
			g.messageQueue.SendToGs(gsAppId, &mq.NetMsg{
				MsgType: mq.MsgTypeServer,
				EventId: mq.ServerPlayerMpRsp,
				ServerMsg: &mq.ServerMsg{
					PlayerMpInfo: &mq.PlayerMpInfo{
						OriginInfo: playerMpInfo.OriginInfo,
						HostUserId: playerMpInfo.HostUserId,
						ApplyOk:    false,
						Reason:     int32(reason),
					},
				},
			})
		}
		if g.dispatchCancel {
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
			return
		}
		hostPlayer := USER_MANAGER.GetOnlineUser(playerMpInfo.HostUserId)
		if hostPlayer == nil {
			logger.Error("player is nil, uid: %v", playerMpInfo.HostUserId)
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
			return
		}
		if WORLD_MANAGER.GetMultiplayerWorldNum() >= MAX_MULTIPLAYER_WORLD_NUM {
			// 超过本服务器最大多人世界数量限制
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_MAX_PLAYER)
			return
		}
		hostWorld := WORLD_MANAGER.GetWorldById(hostPlayer.WorldId)
		if hostWorld == nil {
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
			return
		}
		if hostWorld.IsMultiplayerWorld() && hostWorld.GetOwner().PlayerId != hostPlayer.PlayerId {
			// 向同一世界内的非房主玩家申请时直接拒绝
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
			return
		}
		mpSetting := hostPlayer.PropMap[constant.PLAYER_PROP_PLAYER_MP_SETTING_TYPE]
		if mpSetting == 0 {
			// 房主玩家没开权限
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
			return
		} else if mpSetting == 1 {
			g.PlayerDealEnterWorld(hostPlayer, playerMpInfo.ApplyUserId, true)
			return
		}
		applyTime, exist := hostPlayer.CoopApplyMap[playerMpInfo.ApplyUserId]
		if exist && time.Now().UnixNano() < applyTime+int64(10*time.Second) {
			applyFailNotify(proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP)
			return
		}
		hostPlayer.CoopApplyMap[playerMpInfo.ApplyUserId] = time.Now().UnixNano()

		playerApplyEnterMpNotify := new(proto.PlayerApplyEnterMpNotify)
		playerApplyEnterMpNotify.SrcPlayerInfo = &proto.OnlinePlayerInfo{
			Uid:                 playerMpInfo.ApplyPlayerOnlineInfo.UserId,
			Nickname:            playerMpInfo.ApplyPlayerOnlineInfo.Nickname,
			PlayerLevel:         playerMpInfo.ApplyPlayerOnlineInfo.PlayerLevel,
			AvatarId:            playerMpInfo.ApplyPlayerOnlineInfo.HeadImageId,
			MpSettingType:       proto.MpSettingType(playerMpInfo.ApplyPlayerOnlineInfo.MpSettingType),
			NameCardId:          playerMpInfo.ApplyPlayerOnlineInfo.NameCardId,
			Signature:           playerMpInfo.ApplyPlayerOnlineInfo.Signature,
			ProfilePicture:      &proto.ProfilePicture{AvatarId: playerMpInfo.ApplyPlayerOnlineInfo.HeadImageId},
			CurPlayerNumInWorld: playerMpInfo.ApplyPlayerOnlineInfo.WorldPlayerNum,
		}
		g.SendMsg(cmd.PlayerApplyEnterMpNotify, hostPlayer.PlayerId, hostPlayer.ClientSeq, playerApplyEnterMpNotify)

		g.messageQueue.SendToGs(gsAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeServer,
			EventId: mq.ServerPlayerMpRsp,
			ServerMsg: &mq.ServerMsg{
				PlayerMpInfo: &mq.PlayerMpInfo{
					OriginInfo: playerMpInfo.OriginInfo,
					HostUserId: playerMpInfo.HostUserId,
					ApplyOk:    true,
				},
			},
		})
	case "PlayerApplyEnterMpResultReq":
		applyPlayer := USER_MANAGER.GetOnlineUser(playerMpInfo.ApplyUserId)
		if applyPlayer == nil {
			logger.Error("player is nil, uid: %v", playerMpInfo.ApplyUserId)
			return
		}
		applyPlayerWorld := WORLD_MANAGER.GetWorldById(applyPlayer.WorldId)
		if applyPlayerWorld == nil || applyPlayerWorld.IsMultiplayerWorld() {
			playerApplyEnterMpResultNotify := &proto.PlayerApplyEnterMpResultNotify{
				TargetUid:      playerMpInfo.HostUserId,
				TargetNickname: playerMpInfo.HostNickname,
				IsAgreed:       false,
				Reason:         proto.PlayerApplyEnterMpResultNotify_PLAYER_CANNOT_ENTER_MP,
			}
			g.SendMsg(cmd.PlayerApplyEnterMpResultNotify, applyPlayer.PlayerId, applyPlayer.ClientSeq, playerApplyEnterMpResultNotify)
			return
		}

		playerApplyEnterMpResultNotify := &proto.PlayerApplyEnterMpResultNotify{
			TargetUid:      playerMpInfo.HostUserId,
			TargetNickname: playerMpInfo.HostNickname,
			IsAgreed:       playerMpInfo.Agreed,
			Reason:         proto.PlayerApplyEnterMpResultNotify_PLAYER_JUDGE,
		}
		g.SendMsg(cmd.PlayerApplyEnterMpResultNotify, applyPlayer.PlayerId, applyPlayer.ClientSeq, playerApplyEnterMpResultNotify)
	}
}

func (g *Game) ServerPlayerMpRsp(playerMpInfo *mq.PlayerMpInfo) {
	switch playerMpInfo.OriginInfo.CmdName {
	case "PlayerApplyEnterMpReq":
		player := USER_MANAGER.GetOnlineUser(playerMpInfo.OriginInfo.UserId)
		if player == nil {
			logger.Error("player is nil, uid: %v", playerMpInfo.OriginInfo.UserId)
			return
		}
		if !playerMpInfo.ApplyOk {
			playerApplyEnterMpResultNotify := &proto.PlayerApplyEnterMpResultNotify{
				TargetUid:      playerMpInfo.HostUserId,
				TargetNickname: "",
				IsAgreed:       false,
				Reason:         proto.PlayerApplyEnterMpResultNotify_Reason(playerMpInfo.Reason),
			}
			g.SendMsg(cmd.PlayerApplyEnterMpResultNotify, player.PlayerId, player.ClientSeq, playerApplyEnterMpResultNotify)
		}
	}
}

/************************************************** 打包封装 **************************************************/
