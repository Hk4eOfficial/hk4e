package game

import (
	"strconv"
	"strings"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/pkg/random"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

// 大地图模块 大世界相关的所有逻辑

/************************************************** 接口请求 **************************************************/

// SceneTransToPointReq 场景传送到传送点请求
func (g *Game) SceneTransToPointReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SceneTransToPointReq)
	if player.SceneLoadState != model.SceneEnterDone {
		g.SendError(cmd.SceneTransToPointRsp, player, &proto.SceneTransToPointRsp{}, proto.Retcode_RET_IN_TRANSFER)
		return
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		g.SendError(cmd.SceneTransToPointRsp, player, &proto.SceneTransToPointRsp{})
		return
	}
	owner := world.GetOwner()
	dbWorld := owner.GetDbWorld()
	dbScene := dbWorld.GetSceneById(req.SceneId)
	if dbScene == nil {
		g.SendError(cmd.SceneTransToPointRsp, player, &proto.SceneTransToPointRsp{}, proto.Retcode_RET_POINT_NOT_UNLOCKED)
		return
	}
	unlock := dbScene.CheckPointUnlock(req.PointId)
	if !unlock {
		g.SendError(cmd.SceneTransToPointRsp, player, &proto.SceneTransToPointRsp{}, proto.Retcode_RET_POINT_NOT_UNLOCKED)
		return
	}
	pointDataConfig := gdconf.GetScenePointBySceneIdAndPointId(int32(req.SceneId), int32(req.PointId))
	if pointDataConfig == nil {
		g.SendError(cmd.SceneTransToPointRsp, player, &proto.SceneTransToPointRsp{}, proto.Retcode_RET_POINT_NOT_UNLOCKED)
		return
	}

	// 传送玩家
	g.TeleportPlayer(
		player,
		proto.EnterReason_ENTER_REASON_TRANS_POINT,
		req.SceneId,
		&model.Vector{X: pointDataConfig.TranPos.X, Y: pointDataConfig.TranPos.Y, Z: pointDataConfig.TranPos.Z},
		&model.Vector{X: pointDataConfig.TranRot.X, Y: pointDataConfig.TranRot.Y, Z: pointDataConfig.TranRot.Z},
		0,
		0,
	)

	rsp := &proto.SceneTransToPointRsp{
		PointId: req.PointId,
		SceneId: req.SceneId,
	}
	g.SendMsg(cmd.SceneTransToPointRsp, player.PlayerId, player.ClientSeq, rsp)
}

// UnlockTransPointReq 解锁传送点请求
func (g *Game) UnlockTransPointReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.UnlockTransPointReq)

	ret := g.UnlockPlayerScenePoint(player, req.SceneId, req.PointId)
	if ret != proto.Retcode_RET_SUCC {
		g.SendError(cmd.UnlockTransPointRsp, player, &proto.UnlockTransPointRsp{}, ret)
		return
	}

	g.SendSucc(cmd.UnlockTransPointRsp, player, &proto.UnlockTransPointRsp{})
}

// GetScenePointReq 获取场景锚点请求
func (g *Game) GetScenePointReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetScenePointReq)

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", world.GetId(), player.PlayerId)
		g.SendError(cmd.GetScenePointRsp, player, &proto.GetScenePointRsp{})
		return
	}
	owner := world.GetOwner()
	if owner == nil {
		logger.Error("get owner is nil, worldId: %v", world.GetId())
		g.SendError(cmd.GetScenePointRsp, player, &proto.GetScenePointRsp{})
		return
	}
	dbWorld := owner.GetDbWorld()
	if dbWorld == nil {
		logger.Error("get dbWorld is nil, uid: %v", player.PlayerId)
		g.SendError(cmd.GetScenePointRsp, player, &proto.GetScenePointRsp{})
		return
	}
	dbScene := dbWorld.GetSceneById(req.SceneId)
	if dbScene == nil {
		logger.Error("get dbScene is nil, sceneId: %v, uid: %v", req.SceneId, player.PlayerId)
		g.SendError(cmd.GetScenePointRsp, player, &proto.GetScenePointRsp{})
		return
	}

	rsp := &proto.GetScenePointRsp{
		SceneId:           req.SceneId,
		UnlockAreaList:    dbScene.GetUnlockAreaList(),
		UnlockedPointList: dbScene.GetUnlockPointList(),
		UnhidePointList:   dbScene.GetUnHidePointList(),
	}
	g.SendMsg(cmd.GetScenePointRsp, player.PlayerId, player.ClientSeq, rsp)
}

// MarkMapReq 地图标点请求
func (g *Game) MarkMapReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.MarkMapReq)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}

	// 触发事件
	if PLUGIN_MANAGER.TriggerEvent(PluginEventIdMarkMap, &PluginEventMarkMap{
		PluginEvent: NewPluginEvent(),
		Player:      player,
		Req:         req,
	}) {
		return
	}

	// 地图标点传送
	if req.Op == proto.MarkMapReq_ADD && req.Mark.PointType == proto.MapMarkPointType_NPC && strings.Contains(req.Mark.Name, "@@") {
		posYStr := strings.ReplaceAll(req.Mark.Name, "@@", "")
		posY, err := strconv.Atoi(posYStr)
		if err != nil {
			logger.Error("parse pos y error: %v", err)
			posY = 300
		}
		g.TeleportPlayer(
			player,
			proto.EnterReason_ENTER_REASON_GM,
			req.Mark.SceneId,
			&model.Vector{X: float64(req.Mark.Pos.X), Y: float64(posY), Z: float64(req.Mark.Pos.Z)},
			new(model.Vector),
			0,
			0,
		)
		g.SendMsg(cmd.MarkMapRsp, player.PlayerId, player.ClientSeq, &proto.MarkMapRsp{MarkList: g.PacketMapMarkPointList(player)})
		return
	}
	dbWorld := player.GetDbWorld()
	switch req.Op {
	case proto.MarkMapReq_ADD:
		mark := &model.MapMark{
			SceneId: req.Mark.SceneId,
			Pos: &model.Vector{
				X: float64(req.Mark.Pos.X),
				Y: float64(req.Mark.Pos.Y),
				Z: float64(req.Mark.Pos.Z),
			},
			PointType: uint32(req.Mark.PointType),
			Name:      req.Mark.Name,
		}
		dbWorld.MapMarkList = append(dbWorld.MapMarkList, mark)
	case proto.MarkMapReq_DEL:
		newMapMarkList := make([]*model.MapMark, 0, len(dbWorld.MapMarkList))
		for _, mapMark := range dbWorld.MapMarkList {
			if mapMark.SceneId == req.Mark.SceneId &&
				int32(mapMark.Pos.X) == int32(req.Mark.Pos.X) &&
				int32(mapMark.Pos.Y) == int32(req.Mark.Pos.Y) &&
				int32(mapMark.Pos.Z) == int32(req.Mark.Pos.Z) {
				continue
			}
			newMapMarkList = append(newMapMarkList, mapMark)
		}
		dbWorld.MapMarkList = newMapMarkList
	case proto.MarkMapReq_MOD:
		newMapMarkList := make([]*model.MapMark, 0, len(dbWorld.MapMarkList))
		for _, mapMark := range dbWorld.MapMarkList {
			if mapMark.SceneId == req.Old.SceneId &&
				int32(mapMark.Pos.X) == int32(req.Old.Pos.X) &&
				int32(mapMark.Pos.Y) == int32(req.Old.Pos.Y) &&
				int32(mapMark.Pos.Z) == int32(req.Old.Pos.Z) {
				mapMark = &model.MapMark{
					SceneId: req.Mark.SceneId,
					Pos: &model.Vector{
						X: float64(req.Mark.Pos.X),
						Y: float64(req.Mark.Pos.Y),
						Z: float64(req.Mark.Pos.Z),
					},
					PointType: uint32(req.Mark.PointType),
					Name:      req.Mark.Name,
				}
			}
			newMapMarkList = append(newMapMarkList, mapMark)
		}
		dbWorld.MapMarkList = newMapMarkList
	case proto.MarkMapReq_GET:
	}
	g.SendMsg(cmd.MarkMapRsp, player.PlayerId, player.ClientSeq, &proto.MarkMapRsp{MarkList: g.PacketMapMarkPointList(player)})
}

// GetSceneAreaReq 获取场景区域请求
func (g *Game) GetSceneAreaReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GetSceneAreaReq)

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", world.GetId(), player.PlayerId)
		g.SendError(cmd.GetSceneAreaRsp, player, &proto.GetSceneAreaRsp{})
		return
	}
	owner := world.GetOwner()
	if owner == nil {
		logger.Error("get owner is nil, worldId: %v", world.GetId())
		g.SendError(cmd.GetSceneAreaRsp, player, &proto.GetSceneAreaRsp{})
		return
	}
	dbWorld := owner.GetDbWorld()
	if dbWorld == nil {
		logger.Error("get dbWorld is nil, uid: %v", player.PlayerId)
		g.SendError(cmd.GetSceneAreaRsp, player, &proto.GetSceneAreaRsp{})
		return
	}
	dbScene := dbWorld.GetSceneById(req.SceneId)
	if dbScene == nil {
		logger.Error("get dbScene is nil, sceneId: %v, uid: %v", req.SceneId, player.PlayerId)
		g.SendError(cmd.GetSceneAreaRsp, player, &proto.GetSceneAreaRsp{})
		return
	}

	rsp := &proto.GetSceneAreaRsp{
		SceneId:      req.SceneId,
		AreaIdList:   dbScene.GetUnlockAreaList(),
		CityInfoList: nil,
	}
	if req.SceneId == 3 {
		rsp.CityInfoList = []*proto.CityInfo{
			{CityId: 1, Level: 10},
			{CityId: 2, Level: 10},
			{CityId: 3, Level: 10},
			{CityId: 4, Level: 10},
			{CityId: 5, Level: 10},
		}
	}
	g.SendMsg(cmd.GetSceneAreaRsp, player.PlayerId, player.ClientSeq, rsp)
}

// EnterWorldAreaReq 进入世界区域请求
func (g *Game) EnterWorldAreaReq(player *model.Player, payloadMsg pb.Message) {
	logger.Debug("player enter world area, uid: %v", player.PlayerId)
	req := payloadMsg.(*proto.EnterWorldAreaReq)

	logger.Debug("EnterWorldAreaReq: %v", req)

	rsp := &proto.EnterWorldAreaRsp{
		AreaType: req.AreaType,
		AreaId:   req.AreaId,
	}
	g.SendMsg(cmd.EnterWorldAreaRsp, player.PlayerId, player.ClientSeq, rsp)
}

// ChangeGameTimeReq 修改游戏时间请求
func (g *Game) ChangeGameTimeReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.ChangeGameTimeReq)
	gameTime := req.GameTime
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	if scene == nil {
		logger.Error("scene is nil, sceneId: %v, uid: %v", player.GetSceneId(), player.PlayerId)
		return
	}
	logger.Debug("change game time, gameTime: %v, uid: %v", gameTime, player.PlayerId)
	g.ChangeGameTime(scene, gameTime)

	// 设置玩家天气
	climateType := GAME.GetWeatherAreaClimate(player.WeatherInfo.WeatherAreaId)
	// 跳过相同的天气
	if climateType != player.WeatherInfo.ClimateType {
		g.SetPlayerWeather(player, player.WeatherInfo.WeatherAreaId, climateType)
	}

	rsp := &proto.ChangeGameTimeRsp{
		CurGameTime: scene.GetGameTime(),
	}
	g.SendMsg(cmd.ChangeGameTimeRsp, player.PlayerId, player.ClientSeq, rsp)
}

// NpcTalkReq npc对话请求
func (g *Game) NpcTalkReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.NpcTalkReq)

	g.TriggerQuest(player, constant.QUEST_FINISH_COND_TYPE_COMPLETE_TALK, "", int32(req.TalkId))

	rsp := &proto.NpcTalkRsp{
		CurTalkId:   req.TalkId,
		NpcEntityId: req.NpcEntityId,
		EntityId:    req.EntityId,
	}
	g.SendMsg(cmd.NpcTalkRsp, player.PlayerId, player.ClientSeq, rsp)
}

// DungeonEntryInfoReq 秘境信息请求
func (g *Game) DungeonEntryInfoReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.DungeonEntryInfoReq)

	pointDataConfig := gdconf.GetScenePointBySceneIdAndPointId(int32(req.SceneId), int32(req.PointId))
	if pointDataConfig == nil {
		g.SendError(cmd.DungeonEntryInfoRsp, player, &proto.DungeonEntryInfoRsp{})
		return
	}

	rsp := &proto.DungeonEntryInfoRsp{
		DungeonEntryList: make([]*proto.DungeonEntryInfo, 0),
		PointId:          req.PointId,
	}
	for _, dungeonId := range pointDataConfig.DungeonIds {
		rsp.DungeonEntryList = append(rsp.DungeonEntryList, &proto.DungeonEntryInfo{
			DungeonId: uint32(dungeonId),
		})
	}
	g.SendMsg(cmd.DungeonEntryInfoRsp, player.PlayerId, player.ClientSeq, rsp)
}

// PlayerEnterDungeonReq 玩家进入秘境请求
func (g *Game) PlayerEnterDungeonReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PlayerEnterDungeonReq)
	dungeonDataConfig := gdconf.GetDungeonDataById(int32(req.DungeonId))
	if dungeonDataConfig == nil {
		logger.Error("get dungeon data config is nil, dungeonId: %v, uid: %v", req.DungeonId, player.PlayerId)
		return
	}
	sceneLuaConfig := gdconf.GetSceneLuaConfigById(dungeonDataConfig.SceneId)
	if sceneLuaConfig == nil {
		logger.Error("get scene lua config is nil, sceneId: %v, uid: %v", dungeonDataConfig.SceneId, player.PlayerId)
		return
	}
	sceneConfig := sceneLuaConfig.SceneConfig
	g.TeleportPlayer(
		player,
		proto.EnterReason_ENTER_REASON_DUNGEON_ENTER,
		uint32(dungeonDataConfig.SceneId),
		&model.Vector{X: float64(sceneConfig.BornPos.X), Y: float64(sceneConfig.BornPos.Y), Z: float64(sceneConfig.BornPos.Z)},
		&model.Vector{X: float64(sceneConfig.BornRot.X), Y: float64(sceneConfig.BornRot.Y), Z: float64(sceneConfig.BornRot.Z)},
		req.DungeonId,
		req.PointId,
	)

	rsp := &proto.PlayerEnterDungeonRsp{
		DungeonId: req.DungeonId,
		PointId:   req.PointId,
	}
	g.SendMsg(cmd.PlayerEnterDungeonRsp, player.PlayerId, player.ClientSeq, rsp)
}

// PlayerQuitDungeonReq 玩家离开秘境请求
func (g *Game) PlayerQuitDungeonReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PlayerQuitDungeonReq)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	ctx := world.GetLastEnterSceneContextByUid(player.PlayerId)
	if ctx == nil {
		return
	}
	pointDataConfig := gdconf.GetScenePointBySceneIdAndPointId(int32(ctx.OldSceneId), int32(ctx.OldDungeonPointId))
	if pointDataConfig == nil {
		return
	}
	g.TeleportPlayer(
		player,
		proto.EnterReason_ENTER_REASON_DUNGEON_QUIT,
		ctx.OldSceneId,
		&model.Vector{X: pointDataConfig.TranPos.X, Y: pointDataConfig.TranPos.Y, Z: pointDataConfig.TranPos.Z},
		&model.Vector{X: pointDataConfig.TranRot.X, Y: pointDataConfig.TranRot.Y, Z: pointDataConfig.TranRot.Z},
		0,
		0,
	)

	rsp := &proto.PlayerQuitDungeonRsp{
		PointId: req.PointId,
	}
	g.SendMsg(cmd.PlayerQuitDungeonRsp, player.PlayerId, player.ClientSeq, rsp)
}

// GadgetInteractReq gadget交互请求
func (g *Game) GadgetInteractReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GadgetInteractReq)

	// 触发事件
	if PLUGIN_MANAGER.TriggerEvent(PluginEventIdGadgetInteract, &PluginEventGadgetInteract{
		PluginEvent: NewPluginEvent(),
		Player:      player,
		Req:         req,
	}) {
		return
	}

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	entity := scene.GetEntity(req.GadgetEntityId)
	if entity == nil {
		logger.Error("get entity is nil, entityId: %v, uid: %v", req.GadgetEntityId, player.PlayerId)
		return
	}

	interactType := proto.InteractType_INTERACT_NONE
	switch entity.GetEntityType() {
	case constant.ENTITY_TYPE_GADGET:
		gadgetEntity := entity.GetGadgetEntity()
		gadgetDataConfig := gdconf.GetGadgetDataById(int32(gadgetEntity.GetGadgetId()))
		if gadgetDataConfig == nil {
			logger.Error("get gadget data config is nil, gadgetId: %v, uid: %v", gadgetEntity.GetGadgetId(), player.PlayerId)
			return
		}
		logger.Debug("[GadgetInteractReq] GadgetData: %+v, EntityId: %v, uid: %v", gadgetDataConfig, entity.GetId(), player.PlayerId)
		switch gadgetDataConfig.Type {
		case constant.GADGET_TYPE_GADGET, constant.GADGET_TYPE_EQUIP, constant.GADGET_TYPE_ENERGY_BALL:
			// 掉落物捡起
			interactType = proto.InteractType_INTERACT_PICK_ITEM
			gadgetNormalEntity := gadgetEntity.GetGadgetNormalEntity()
			g.AddPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: gadgetNormalEntity.GetItemId(), ChangeCount: 1}}, proto.ActionReasonType_ACTION_REASON_SUBFIELD_DROP)
			g.KillEntity(player, scene, entity.GetId(), proto.PlayerDieType_PLAYER_DIE_NONE)
		case constant.GADGET_TYPE_GATHER_OBJECT:
			// 采集物摘取
			interactType = proto.InteractType_INTERACT_GATHER
			gadgetNormalEntity := gadgetEntity.GetGadgetNormalEntity()
			g.AddPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: gadgetNormalEntity.GetItemId(), ChangeCount: 1}}, proto.ActionReasonType_ACTION_REASON_GATHER)
			g.KillEntity(player, scene, entity.GetId(), proto.PlayerDieType_PLAYER_DIE_NONE)
		case constant.GADGET_TYPE_CHEST:
			// 宝箱开启
			interactType = proto.InteractType_INTERACT_OPEN_CHEST
			// 宝箱交互结束 开启宝箱
			if req.OpType == proto.InterOpType_INTER_OP_FINISH {
				// 随机掉落
				g.chestDrop(player, entity)
				// 更新宝箱状态
				g.SendMsg(cmd.WorldChestOpenNotify, player.PlayerId, player.ClientSeq, &proto.WorldChestOpenNotify{
					GroupId:  entity.GetGroupId(),
					SceneId:  scene.GetId(),
					ConfigId: entity.GetConfigId(),
				})
				g.ChangeGadgetState(player, entity.GetId(), constant.GADGET_STATE_CHEST_OPENED)
				g.KillEntity(player, scene, entity.GetId(), proto.PlayerDieType_PLAYER_DIE_NONE)
			}
		default:
			logger.Error("not support gadget type: %v, uid: %v", gadgetDataConfig.Type, player.PlayerId)
		}
	case constant.ENTITY_TYPE_MONSTER:
		// TODO 环境动物掉落道具
		g.KillEntity(player, scene, entity.GetId(), proto.PlayerDieType_PLAYER_DIE_NONE)
	default:
		logger.Error("not support entity type: %v, uid: %v", entity.GetEntityType(), player.PlayerId)
	}

	rsp := &proto.GadgetInteractRsp{
		GadgetEntityId: req.GadgetEntityId,
		GadgetId:       req.GadgetId,
		OpType:         req.OpType,
		InteractType:   interactType,
	}
	g.SendMsg(cmd.GadgetInteractRsp, player.PlayerId, player.ClientSeq, rsp)
}

func (g *Game) EnterTransPointRegionNotify(player *model.Player, payloadMsg pb.Message) {
	ntf := payloadMsg.(*proto.EnterTransPointRegionNotify)
	_ = ntf

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}

	if WORLD_MANAGER.IsAiWorld(world) {
		return
	}

	dbAvatar := player.GetDbAvatar()
	for _, worldAvatar := range world.GetPlayerWorldAvatarList(player) {
		avatar := dbAvatar.GetAvatarById(worldAvatar.GetAvatarId())
		if avatar.LifeState == constant.LIFE_STATE_DEAD {
			g.RevivePlayerAvatar(player, worldAvatar.GetAvatarId())
		}
		g.AddPlayerAvatarHp(player.PlayerId, worldAvatar.GetAvatarId(), 0.0, true, proto.ChangHpReason_CHANGE_HP_ADD_GM)
	}
}

func (g *Game) ExitTransPointRegionNotify(player *model.Player, payloadMsg pb.Message) {
	ntf := payloadMsg.(*proto.ExitTransPointRegionNotify)
	_ = ntf
}

/************************************************** 游戏功能 **************************************************/

// UnlockPlayerScenePoint 解锁场景锚点
func (g *Game) UnlockPlayerScenePoint(player *model.Player, sceneId uint32, pointId uint32) proto.Retcode {
	dbWorld := player.GetDbWorld()
	dbScene := dbWorld.GetSceneById(sceneId)
	if dbScene == nil {
		logger.Error("get dbScene is nil, sceneId: %v, uid: %v", sceneId, player.PlayerId)
		return proto.Retcode_RET_SVR_ERROR
	}
	unlock := dbScene.CheckPointUnlock(pointId)
	if unlock {
		logger.Error("point already unlock, sceneId: %v, pointId: %v, uid: %v", sceneId, pointId, player.PlayerId)
		return proto.Retcode_RET_POINT_ALREAY_UNLOCKED
	}
	dbScene.UnlockPoint(pointId)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return proto.Retcode_RET_SVR_ERROR
	}
	scene := world.GetSceneById(player.SceneId)
	g.SendToSceneA(scene, cmd.ScenePointUnlockNotify, player.ClientSeq, &proto.ScenePointUnlockNotify{
		SceneId:   sceneId,
		PointList: []uint32{pointId},
	}, 0)
	g.TriggerQuest(player, constant.QUEST_FINISH_COND_TYPE_UNLOCK_TRANS_POINT, "", int32(sceneId), int32(pointId))
	return proto.Retcode_RET_SUCC
}

// UnlockPlayerSceneArea 解锁场景区域
func (g *Game) UnlockPlayerSceneArea(player *model.Player, sceneId uint32, areaId uint32) {
	dbWorld := player.GetDbWorld()
	dbScene := dbWorld.GetSceneById(sceneId)
	if dbScene == nil {
		logger.Error("get dbScene is nil, sceneId: %v, uid: %v", sceneId, player.PlayerId)
		return
	}
	unlock := dbScene.CheckAreaUnlock(areaId)
	if unlock {
		logger.Error("area already unlock, sceneId: %v, areaId: %v, uid: %v", sceneId, areaId, player.PlayerId)
		return
	}
	dbScene.UnlockArea(areaId)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.SceneId)
	g.SendToSceneA(scene, cmd.SceneAreaUnlockNotify, player.ClientSeq, &proto.SceneAreaUnlockNotify{
		SceneId:  sceneId,
		AreaList: []uint32{areaId},
	}, 0)
	g.TriggerQuest(player, constant.QUEST_FINISH_COND_TYPE_UNLOCK_AREA, "", int32(sceneId), int32(areaId))
}

// ChangeGameTime 修改游戏场景时间
func (g *Game) ChangeGameTime(scene *Scene, gameTime uint32) {
	scene.ChangeGameTime(gameTime)
	for _, scenePlayer := range scene.GetAllPlayer() {
		playerGameTimeNotify := &proto.PlayerGameTimeNotify{
			GameTime: scene.GetGameTime(),
			Uid:      scenePlayer.PlayerId,
		}
		g.SendMsg(cmd.PlayerGameTimeNotify, scenePlayer.PlayerId, scenePlayer.ClientSeq, playerGameTimeNotify)
	}
}

const (
	MonsterDropTypeHp = iota
	MonsterDropTypeKill
)

func (g *Game) monsterDrop(player *model.Player, monsterDropType int, hpDropId int32, entity *Entity) {
	dropId := int32(0)
	dropCount := int32(0)
	switch monsterDropType {
	case MonsterDropTypeHp:
		dropId = hpDropId
		dropCount = 1
	case MonsterDropTypeKill:
		sceneGroupConfig := gdconf.GetSceneGroup(int32(entity.GetGroupId()))
		if sceneGroupConfig == nil {
			logger.Error("get scene group config is nil, groupId: %v, uid: %v", entity.GetGroupId(), player.PlayerId)
			return
		}
		monsterConfig := sceneGroupConfig.MonsterMap[int32(entity.GetConfigId())]
		if monsterConfig.DropId != 0 {
			dropId = monsterConfig.DropId
			dropCount = 1
		} else {
			dropTag := ""
			if monsterConfig.DropTag != "" {
				dropTag = monsterConfig.DropTag
			} else {
				monsterDataConfig := gdconf.GetMonsterDataById(monsterConfig.MonsterId)
				if monsterDataConfig == nil {
					logger.Error("get monster data config is nil, monsterId: %v, uid: %v", monsterConfig.MonsterId, player.PlayerId)
					return
				}
				dropTag = gdconf.GetDropModelByMonsterModel(monsterDataConfig.Name)
			}
			monsterDropDataConfig := gdconf.GetMonsterDropDataByDropTagAndLevel(dropTag, monsterConfig.Level)
			if monsterDropDataConfig == nil {
				logger.Error("get monster drop data config is nil, monsterConfig: %+v, uid: %v", monsterConfig, player.PlayerId)
				return
			}
			dropId = monsterDropDataConfig.DropId
			dropCount = monsterDropDataConfig.DropCount
		}
	}
	dropDataConfig := gdconf.GetDropDataById(dropId)
	if dropDataConfig == nil {
		logger.Error("get drop data config is nil, dropId: %v, uid: %v", dropId, player.PlayerId)
		return
	}
	totalItemMap := g.doRandDropFullTimes(dropDataConfig, int(dropCount))
	for itemId, count := range totalItemMap {
		itemDataConfig := gdconf.GetItemDataById(int32(itemId))
		if itemDataConfig == nil {
			logger.Error("get item data config is nil, itemId: %v, uid: %v", itemId, player.PlayerId)
			continue
		}
		if itemDataConfig.GadgetId != 0 {
			g.CreateDropGadget(player, entity.GetPos(), uint32(itemDataConfig.GadgetId), itemId, count)
		} else {
			g.AddPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: itemId, ChangeCount: count}}, proto.ActionReasonType_ACTION_REASON_SUBFIELD_DROP)
		}
	}
}

func (g *Game) chestDrop(player *model.Player, entity *Entity) {
	sceneGroupConfig := gdconf.GetSceneGroup(int32(entity.GetGroupId()))
	if sceneGroupConfig == nil {
		logger.Error("get scene group config is nil, groupId: %v, uid: %v", entity.GetGroupId(), player.PlayerId)
		return
	}
	gadgetConfig := sceneGroupConfig.GadgetMap[int32(entity.GetConfigId())]
	dropId := int32(0)
	dropCount := int32(0)
	if gadgetConfig.ChestDropId != 0 {
		dropId = gadgetConfig.ChestDropId
		dropCount = 1
	} else {
		chestDropDataConfig := gdconf.GetChestDropDataByDropTagAndLevel(gadgetConfig.DropTag, gadgetConfig.Level)
		if chestDropDataConfig == nil {
			logger.Error("get chest drop data config is nil, gadgetConfig: %+v, uid: %v", gadgetConfig, player.PlayerId)
			return
		}
		dropId = chestDropDataConfig.DropId
		dropCount = chestDropDataConfig.DropCount
	}
	dropDataConfig := gdconf.GetDropDataById(dropId)
	if dropDataConfig == nil {
		logger.Error("get drop data config is nil, dropId: %v, uid: %v", dropId, player.PlayerId)
		return
	}
	totalItemMap := g.doRandDropFullTimes(dropDataConfig, int(dropCount))
	for itemId, count := range totalItemMap {
		itemDataConfig := gdconf.GetItemDataById(int32(itemId))
		if itemDataConfig == nil {
			logger.Error("get item data config is nil, itemId: %v, uid: %v", itemId, player.PlayerId)
			continue
		}
		if itemDataConfig.GadgetId != 0 {
			g.CreateDropGadget(player, entity.GetPos(), uint32(itemDataConfig.GadgetId), itemId, count)
		} else {
			g.AddPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: itemId, ChangeCount: count}}, proto.ActionReasonType_ACTION_REASON_SUBFIELD_DROP)
		}
	}
}

func (g *Game) doRandDropFullTimes(dropDataConfig *gdconf.DropData, times int) map[uint32]uint32 {
	totalItemMap := make(map[uint32]uint32)
	for i := 0; i < times; i++ {
		itemMap := g.doRandDropFull(dropDataConfig)
		if itemMap == nil {
			continue
		}
		for itemId, count := range itemMap {
			totalItemMap[itemId] += count
		}
	}
	return totalItemMap
}

func (g *Game) doRandDropFull(dropDataConfig *gdconf.DropData) map[uint32]uint32 {
	itemMap := make(map[uint32]uint32)
	dropList := make([]*gdconf.DropData, 0)
	dropList = append(dropList, dropDataConfig)
	for i := 0; i < 1000; i++ {
		if len(dropList) == 0 {
			// 掉落结束
			return itemMap
		}
		dropMap := g.doRandDropOnce(dropList[0])
		dropList = dropList[1:]
		for dropId, count := range dropMap {
			// 掉落id优先在掉落表里找 找不到就去道具表里找
			subDropDataConfig := gdconf.GetDropDataById(dropId)
			if subDropDataConfig != nil {
				// 添加子掉落
				dropList = append(dropList, subDropDataConfig)
			} else {
				// 添加道具
				itemMap[uint32(dropId)] += uint32(count)
			}
		}
	}
	logger.Error("drop overtimes, drop config: %v", dropDataConfig)
	return nil
}

func (g *Game) doRandDropOnce(dropDataConfig *gdconf.DropData) map[int32]int32 {
	dropMap := make(map[int32]int32)
	switch dropDataConfig.RandomType {
	case gdconf.RandomTypeChoose:
		// RWS随机
		randNum := random.GetRandomInt32(0, dropDataConfig.SubDropTotalWeight-1)
		sumWeight := int32(0)
		for _, subDrop := range dropDataConfig.SubDropList {
			sumWeight += subDrop.Weight
			if sumWeight > randNum {
				count := random.GetRandomInt32(subDrop.CountRange[0], subDrop.CountRange[1])
				if count > 0 {
					dropMap[subDrop.Id] = count
				}
				break
			}
		}
	case gdconf.RandomTypeIndep:
		// 独立随机
		randNum := random.GetRandomInt32(0, gdconf.RandomTypeIndepWeight-1)
		for _, subDrop := range dropDataConfig.SubDropList {
			if subDrop.Weight > randNum {
				count := random.GetRandomInt32(subDrop.CountRange[0], subDrop.CountRange[1])
				if count > 0 {
					dropMap[subDrop.Id] += count
				}
			}
		}
	}
	return dropMap
}

// TeleportPlayer 传送玩家通用接口
func (g *Game) TeleportPlayer(
	player *model.Player, enterReason proto.EnterReason,
	sceneId uint32, pos, rot *model.Vector,
	dungeonId, dungeonPointId uint32,
) {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	if WORLD_MANAGER.IsAiWorld(world) {
		return
	}

	oldSceneId := player.GetSceneId()
	oldPos := g.GetPlayerPos(player)
	newSceneId := sceneId
	newPos := pos
	newRot := rot

	var enterType proto.EnterType
	if newSceneId != oldSceneId {
		player.SceneJump = true
		logger.Debug("player jump scene, scene: %v, pos: %v", player.GetSceneId(), newPos)
		enterType = proto.EnterType_ENTER_JUMP
		if enterReason == proto.EnterReason_ENTER_REASON_DUNGEON_ENTER {
			logger.Debug("player tp to dungeon scene, sceneId: %v, pos: %v", player.GetSceneId(), newPos)
			enterType = proto.EnterType_ENTER_DUNGEON
		}
		delTeamEntityNotify := g.PacketDelTeamEntityNotify(world, player)
		g.SendMsg(cmd.DelTeamEntityNotify, player.PlayerId, player.ClientSeq, delTeamEntityNotify)
	} else {
		player.SceneJump = false
		logger.Debug("player goto scene, scene: %v, pos: %v", player.GetSceneId(), newPos)
		enterType = proto.EnterType_ENTER_GOTO
	}

	player.SceneEnterReason = uint32(enterReason)

	enterSceneToken := world.AddEnterSceneContext(&EnterSceneContext{
		OldSceneId:        oldSceneId,
		OldPos:            oldPos,
		NewSceneId:        newSceneId,
		NewPos:            newPos,
		NewRot:            newRot,
		OldDungeonPointId: dungeonPointId,
		Uid:               player.PlayerId,
	})

	playerEnterSceneNotify := g.PacketPlayerEnterSceneNotifyTp(player, enterType, newSceneId, newPos, dungeonId, enterSceneToken)
	g.SendMsg(cmd.PlayerEnterSceneNotify, player.PlayerId, player.ClientSeq, playerEnterSceneNotify)
}

/************************************************** 打包封装 **************************************************/

func (g *Game) PacketMapMarkPointList(player *model.Player) []*proto.MapMarkPoint {
	pbMarkList := make([]*proto.MapMarkPoint, 0)
	dbWorld := player.GetDbWorld()
	for _, mapMark := range dbWorld.MapMarkList {
		pbMarkList = append(pbMarkList, &proto.MapMarkPoint{
			SceneId: mapMark.SceneId,
			Name:    mapMark.Name,
			Pos: &proto.Vector{
				X: float32(mapMark.Pos.X),
				Y: float32(mapMark.Pos.Y),
				Z: float32(mapMark.Pos.Z),
			},
			PointType: proto.MapMarkPointType(mapMark.PointType),
		})
	}
	return pbMarkList
}
