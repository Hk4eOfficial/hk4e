package game

import (
	"math"
	"reflect"
	"strings"
	"time"

	"hk4e/common/constant"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

func (g *Game) PingReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PingReq)

	player.ClientTime = req.ClientTime
	now := uint32(time.Now().Unix())
	// 客户端与服务器时间相差太过严重
	max := math.Max(float64(now), float64(player.ClientTime))
	min := math.Min(float64(now), float64(player.ClientTime))
	if math.Abs(max-min) > 600.0 {
		logger.Error("abs of client time and server time above 600s, clientTime: %v, uid: %v", player.ClientTime, player.PlayerId)
	}
	player.LastKeepaliveTime = now

	g.SendMsg(cmd.PingRsp, player.PlayerId, player.ClientSeq, &proto.PingRsp{
		ClientTime: req.ClientTime,
		Seq:        req.Seq,
	})
}

func (g *Game) TowerAllDataReq(player *model.Player, payloadMsg pb.Message) {
	towerAllDataRsp := &proto.TowerAllDataRsp{
		TowerScheduleId:        29,
		TowerFloorRecordList:   []*proto.TowerFloorRecord{{FloorId: 1001}},
		CurLevelRecord:         &proto.TowerCurLevelRecord{IsEmpty: true},
		NextScheduleChangeTime: 4294967295,
		FloorOpenTimeMap: map[uint32]uint32{
			1024: 1630486800,
			1025: 1630486800,
			1026: 1630486800,
			1027: 1630486800,
		},
		ScheduleStartTime: 1630486800,
	}
	g.SendMsg(cmd.TowerAllDataRsp, player.PlayerId, player.ClientSeq, towerAllDataRsp)
}

func (g *Game) ClientRttNotify(userId uint32, clientRtt uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	// logger.Debug("client rtt notify, uid: %v, rtt: %v", userId, clientRtt)
	player.ClientRTT = clientRtt
}

func (g *Game) ServerAnnounceNotify(announceId uint32, announceMsg string) {
	for _, onlinePlayer := range USER_MANAGER.GetAllOnlineUserList() {
		now := uint32(time.Now().Unix())
		serverAnnounceNotify := &proto.ServerAnnounceNotify{
			AnnounceDataList: []*proto.AnnounceData{{
				ConfigId:              announceId,
				BeginTime:             now + 1,
				EndTime:               now + 2,
				CenterSystemText:      announceMsg,
				CenterSystemFrequency: 1,
			}},
		}
		g.SendMsg(cmd.ServerAnnounceNotify, onlinePlayer.PlayerId, 0, serverAnnounceNotify)
	}
}

func (g *Game) ServerAnnounceRevokeNotify(announceId uint32) {
	for _, onlinePlayer := range USER_MANAGER.GetAllOnlineUserList() {
		serverAnnounceRevokeNotify := &proto.ServerAnnounceRevokeNotify{
			ConfigIdList: []uint32{announceId},
		}
		g.SendMsg(cmd.ServerAnnounceRevokeNotify, onlinePlayer.PlayerId, 0, serverAnnounceRevokeNotify)
	}
}

func (g *Game) ToTheMoonEnterSceneReq(player *model.Player, payloadMsg pb.Message) {
	logger.Debug("player ttm enter scene, uid: %v", player.PlayerId)
	req := payloadMsg.(*proto.ToTheMoonEnterSceneReq)
	_ = req
	g.SendMsg(cmd.ToTheMoonEnterSceneRsp, player.PlayerId, player.ClientSeq, new(proto.ToTheMoonEnterSceneRsp))
}

func (g *Game) PathfindingEnterSceneReq(player *model.Player, payloadMsg pb.Message) {
	logger.Debug("player pf enter scene, uid: %v", player.PlayerId)
	req := payloadMsg.(*proto.PathfindingEnterSceneReq)
	_ = req
	g.SendMsg(cmd.PathfindingEnterSceneRsp, player.PlayerId, player.ClientSeq, new(proto.PathfindingEnterSceneRsp))
}

func (g *Game) QueryPathReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.QueryPathReq)
	queryPathRsp := &proto.QueryPathRsp{
		QueryId:     req.QueryId,
		QueryStatus: proto.QueryPathRsp_STATUS_SUCC,
		Corners:     []*proto.Vector{req.DestinationPos[0]},
	}
	g.SendMsg(cmd.QueryPathRsp, player.PlayerId, player.ClientSeq, queryPathRsp)
}

func (g *Game) ObstacleModifyNotify(player *model.Player, payloadMsg pb.Message) {
	ntf := payloadMsg.(*proto.ObstacleModifyNotify)
	_ = ntf
	// logger.Debug("ObstacleModifyNotify: %v, uid: %v", ntf, player.PlayerId)
}

// WorldPlayerRTTNotify 世界里所有玩家的网络延迟广播
func (g *Game) WorldPlayerRTTNotify(world *World) {
	ntf := &proto.WorldPlayerRTTNotify{
		PlayerRttList: make([]*proto.PlayerRTTInfo, 0),
	}
	for _, worldPlayer := range world.GetAllPlayer() {
		playerRTTInfo := &proto.PlayerRTTInfo{Uid: worldPlayer.PlayerId, Rtt: worldPlayer.ClientRTT}
		ntf.PlayerRttList = append(ntf.PlayerRttList, playerRTTInfo)
	}
	g.SendToWorldA(world, cmd.WorldPlayerRTTNotify, 0, ntf, 0)
}

// WorldPlayerLocationNotify 多人世界其他玩家的坐标位置广播
func (g *Game) WorldPlayerLocationNotify(world *World) {
	ntf := &proto.WorldPlayerLocationNotify{
		PlayerWorldLocList: make([]*proto.PlayerWorldLocationInfo, 0),
	}
	for _, worldPlayer := range world.GetAllPlayer() {
		pos := g.GetPlayerPos(worldPlayer)
		rot := g.GetPlayerRot(worldPlayer)
		playerWorldLocationInfo := &proto.PlayerWorldLocationInfo{
			SceneId: worldPlayer.GetSceneId(),
			PlayerLoc: &proto.PlayerLocationInfo{
				Uid: worldPlayer.PlayerId,
				Pos: &proto.Vector{X: float32(pos.X), Y: float32(pos.Y), Z: float32(pos.Z)},
				Rot: &proto.Vector{X: float32(rot.X), Y: float32(rot.Y), Z: float32(rot.Z)},
			},
		}

		if WORLD_MANAGER.IsAiWorld(world) {
			playerWorldLocationInfo.PlayerLoc.Pos = new(proto.Vector)
			playerWorldLocationInfo.PlayerLoc.Rot = new(proto.Vector)
		}

		ntf.PlayerWorldLocList = append(ntf.PlayerWorldLocList, playerWorldLocationInfo)
	}
	g.SendToWorldA(world, cmd.WorldPlayerLocationNotify, 0, ntf, 0)
}

func (g *Game) ScenePlayerLocationNotify(world *World) {
	for _, scene := range world.GetAllScene() {
		ntf := &proto.ScenePlayerLocationNotify{
			SceneId:        scene.GetId(),
			PlayerLocList:  make([]*proto.PlayerLocationInfo, 0),
			VehicleLocList: make([]*proto.VehicleLocationInfo, 0),
		}
		for _, scenePlayer := range scene.GetAllPlayer() {
			pos := g.GetPlayerPos(scenePlayer)
			rot := g.GetPlayerRot(scenePlayer)
			// 玩家位置
			playerLocationInfo := &proto.PlayerLocationInfo{
				Uid: scenePlayer.PlayerId,
				Pos: &proto.Vector{X: float32(pos.X), Y: float32(pos.Y), Z: float32(pos.Z)},
				Rot: &proto.Vector{X: float32(rot.X), Y: float32(rot.Y), Z: float32(rot.Z)},
			}

			if WORLD_MANAGER.IsAiWorld(world) {
				playerLocationInfo.Pos = new(proto.Vector)
				playerLocationInfo.Rot = new(proto.Vector)
			}

			ntf.PlayerLocList = append(ntf.PlayerLocList, playerLocationInfo)
			// 载具位置
			for _, entityId := range scenePlayer.VehicleInfo.CreateEntityIdMap {
				entity := scene.GetEntity(entityId)
				// 确保实体类型是否为载具
				if entity != nil && entity.GetEntityType() == constant.ENTITY_TYPE_GADGET && entity.GetGadgetEntity().GetGadgetVehicleEntity() != nil {
					vehicleLocationInfo := &proto.VehicleLocationInfo{
						Rot: &proto.Vector{
							X: float32(entity.GetRot().X),
							Y: float32(entity.GetRot().Y),
							Z: float32(entity.GetRot().Z),
						},
						EntityId: entity.GetId(),
						CurHp:    entity.GetFightProp()[constant.FIGHT_PROP_CUR_HP],
						OwnerUid: entity.GetGadgetEntity().GetGadgetVehicleEntity().GetOwnerUid(),
						Pos: &proto.Vector{
							X: float32(entity.GetPos().X),
							Y: float32(entity.GetPos().Y),
							Z: float32(entity.GetPos().Z),
						},
						UidList:  make([]uint32, 0, len(entity.GetGadgetEntity().GetGadgetVehicleEntity().GetMemberMap())),
						GadgetId: entity.GetGadgetEntity().GetGadgetVehicleEntity().GetVehicleId(),
						MaxHp:    entity.GetFightProp()[constant.FIGHT_PROP_MAX_HP],
					}
					for _, p := range entity.GetGadgetEntity().GetGadgetVehicleEntity().GetMemberMap() {
						vehicleLocationInfo.UidList = append(vehicleLocationInfo.UidList, p.PlayerId)
					}
					ntf.VehicleLocList = append(ntf.VehicleLocList, vehicleLocationInfo)
				}
			}
		}
		g.SendToSceneA(scene, cmd.ScenePlayerLocationNotify, 0, ntf, 0)
	}
}

func (g *Game) SceneTimeNotify(world *World) {
	for _, scene := range world.GetAllScene() {
		for _, player := range scene.GetAllPlayer() {
			sceneTimeNotify := &proto.SceneTimeNotify{
				SceneId:   player.GetSceneId(),
				SceneTime: uint64(scene.GetSceneTime()),
			}
			g.SendMsg(cmd.SceneTimeNotify, player.PlayerId, 0, sceneTimeNotify)
		}
	}
}

func (g *Game) PlayerTimeNotify(world *World) {
	for _, player := range world.GetAllPlayer() {
		playerTimeNotify := &proto.PlayerTimeNotify{
			IsPaused:   player.Pause,
			PlayerTime: uint64(player.TotalOnlineTime),
			ServerTime: uint64(time.Now().UnixMilli()),
		}
		g.SendMsg(cmd.PlayerTimeNotify, player.PlayerId, 0, playerTimeNotify)
	}
}

func (g *Game) PlayerGameTimeNotify(world *World) {
	for _, player := range world.GetAllPlayer() {
		scene := world.GetSceneById(player.GetSceneId())
		if scene == nil {
			logger.Error("scene is nil, sceneId: %v, uid: %v", player.GetSceneId(), player.PlayerId)
			return
		}
		for _, scenePlayer := range scene.GetAllPlayer() {
			playerGameTimeNotify := &proto.PlayerGameTimeNotify{
				GameTime: scene.GetGameTime(),
				Uid:      scenePlayer.PlayerId,
			}
			g.SendMsg(cmd.PlayerGameTimeNotify, scenePlayer.PlayerId, 0, playerGameTimeNotify)
			// 设置玩家天气
			climateType := GAME.GetWeatherAreaClimate(player.WeatherInfo.WeatherAreaId)
			// 跳过相同的天气
			if climateType == player.WeatherInfo.ClimateType {
				return
			}
			GAME.SetPlayerWeather(player, player.WeatherInfo.WeatherAreaId, climateType)
		}
	}
}

func (g *Game) GmTalkReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.GmTalkReq)
	logger.Info("GmTalkReq: %v", req.Msg)

	commandMessageInput := COMMAND_MANAGER.GetCommandMessageInput()
	if strings.Contains(req.Msg, "@@") {
		commandText := req.Msg
		commandText = strings.ReplaceAll(commandText, "@@", "")
		commandText = strings.ReplaceAll(commandText, " ", "")
		beginIndex := strings.Index(commandText, "(")
		endIndex := strings.Index(commandText, ")")
		if beginIndex == 0 || beginIndex == -1 || endIndex == -1 || beginIndex >= endIndex {
			g.SendMsg(cmd.GmTalkRsp, player.PlayerId, player.ClientSeq, &proto.GmTalkRsp{Retmsg: "GM函数解析失败", Msg: req.Msg})
			return
		}
		// 判断玩家权限是否足够
		if CommandPerm(player.CmdPerm) < CommandPermGM {
			g.SendMsg(cmd.GmTalkRsp, player.PlayerId, player.ClientSeq, &proto.GmTalkRsp{Retmsg: "权限不足", Msg: req.Msg})
			return
		}
		funcName := commandText[:beginIndex]
		paramList := strings.Split(commandText[beginIndex+1:endIndex], ",")
		commandMessageInput <- &CommandMessage{
			GMType:    SystemFuncGM,
			FuncName:  funcName,
			ParamList: paramList,
		}
	} else {
		commandMessageInput <- &CommandMessage{
			GMType:   DevClientGM,
			Executor: player,
			Text:     strings.ToLower(req.Msg),
		}
	}
	g.SendMsg(cmd.GmTalkRsp, player.PlayerId, player.ClientSeq, &proto.GmTalkRsp{Retmsg: "执行成功", Msg: req.Msg})
}

func (g *Game) PacketPropValue(key uint32, value any) *proto.PropValue {
	propValue := new(proto.PropValue)
	propValue.Type = key
	switch value.(type) {
	case int:
		v := value.(int)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	case int8:
		v := value.(int8)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	case int16:
		v := value.(int16)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	case int32:
		v := value.(int32)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	case int64:
		v := value.(int64)
		propValue.Val = v
		propValue.Value = &proto.PropValue_Ival{Ival: v}
	case float32:
		v := value.(float32)
		propValue.Value = &proto.PropValue_Fval{Fval: v}
	case float64:
		v := value.(float64)
		propValue.Value = &proto.PropValue_Fval{Fval: float32(v)}
	case uint8:
		v := value.(uint8)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	case uint16:
		v := value.(uint16)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	case uint32:
		v := value.(uint32)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	case uint64:
		v := value.(uint64)
		propValue.Val = int64(v)
		propValue.Value = &proto.PropValue_Ival{Ival: int64(v)}
	default:
		logger.Error("unknown value type: %v, value: %v", reflect.TypeOf(value), value)
		return nil
	}
	return propValue
}

// GetPlayerPos 获取玩家实时位置
func (g *Game) GetPlayerPos(player *model.Player) *model.Vector {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return player.GetPos()
	}
	entity := world.GetPlayerActiveAvatarEntity(player)
	if entity == nil {
		return player.GetPos()
	}
	return entity.GetPos()
}

// GetPlayerRot 获取玩家实时朝向
func (g *Game) GetPlayerRot(player *model.Player) *model.Vector {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return player.GetRot()
	}
	entity := world.GetPlayerActiveAvatarEntity(player)
	if entity == nil {
		return player.GetRot()
	}
	return entity.GetRot()
}
