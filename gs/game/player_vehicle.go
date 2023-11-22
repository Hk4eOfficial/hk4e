package game

import (
	"time"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

/************************************************** 接口请求 **************************************************/

// CreateVehicleReq 创建载具
func (g *Game) CreateVehicleReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.CreateVehicleReq)

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{})
		return
	}
	scene := world.GetSceneById(player.GetSceneId())

	// 创建载具冷却时间
	createVehicleCd := int64(5000) // TODO 冷却时间读取配置表
	if time.Now().UnixMilli()-player.VehicleInfo.LastCreateTime < createVehicleCd {
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{}, proto.Retcode_RET_CREATE_VEHICLE_IN_CD)
		return
	}

	// TODO req.ScenePointId 验证浪船锚点是否已解锁 Retcode_RET_VEHICLE_POINT_NOT_UNLOCK

	// TODO 验证将要创建的载具位置是否有效 Retcode_RET_CREATE_VEHICLE_POS_INVALID

	// 清除已创建的载具
	lastEntityId, ok := player.VehicleInfo.CreateEntityIdMap[req.VehicleId]
	if ok {
		g.DestroyVehicleEntity(player, scene, req.VehicleId, lastEntityId)
	}

	// 创建载具实体
	pos := &model.Vector{X: float64(req.Pos.X), Y: float64(req.Pos.Y), Z: float64(req.Pos.Z)}
	rot := &model.Vector{X: float64(req.Rot.X), Y: float64(req.Rot.Y), Z: float64(req.Rot.Z)}
	entityId := scene.CreateEntityGadgetVehicle(player.PlayerId, pos, rot, req.VehicleId)
	if entityId == 0 {
		logger.Error("vehicle entityId is 0, uid: %v", player.PlayerId)
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{})
		return
	}
	GAME.AddSceneEntityNotify(player, proto.VisionType_VISION_BORN, []uint32{entityId}, true, false)
	// 记录创建的载具信息
	player.VehicleInfo.CreateEntityIdMap[req.VehicleId] = entityId
	player.VehicleInfo.LastCreateTime = time.Now().UnixMilli()

	// PacketCreateVehicleRsp
	createVehicleRsp := &proto.CreateVehicleRsp{
		VehicleId: req.VehicleId,
		EntityId:  entityId,
	}
	g.SendMsg(cmd.CreateVehicleRsp, player.PlayerId, player.ClientSeq, createVehicleRsp)
}

// VehicleInteractReq 载具交互
func (g *Game) VehicleInteractReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.VehicleInteractReq)

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{})
		return
	}
	scene := world.GetSceneById(player.GetSceneId())

	// 获取载具实体
	entity := scene.GetEntity(req.EntityId)
	if entity == nil {
		logger.Error("vehicle entity is nil, entityId: %v", req.EntityId)
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{}, proto.Retcode_RET_ENTITY_NOT_EXIST)
		return
	}
	// 判断实体类型是否为载具
	gadgetEntity := entity.GetGadgetEntity()
	if gadgetEntity == nil || gadgetEntity.GetGadgetVehicleEntity() == nil {
		logger.Error("vehicle entity error, entityType: %v", entity.GetEntityType())
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{}, proto.Retcode_RET_GADGET_NOT_VEHICLE)
		return
	}

	dbTeam := player.GetDbTeam()
	dbAvatar := player.GetDbAvatar()
	avatarGuid := dbAvatar.GetAvatarById(dbTeam.GetActiveAvatarId()).Guid

	switch req.InteractType {
	case proto.VehicleInteractType_VEHICLE_INTERACT_IN:
		// 进入载具
		g.EnterVehicle(player, entity, avatarGuid)
	case proto.VehicleInteractType_VEHICLE_INTERACT_OUT:
		// 离开载具
		g.ExitVehicle(player, entity, avatarGuid)
	}
}

/************************************************** 游戏功能 **************************************************/

// VehicleDestroyMotion 载具销毁动作
func (g *Game) VehicleDestroyMotion(player *model.Player, entity *Entity, state proto.MotionState) {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	scene := world.GetSceneById(player.GetSceneId())

	// 状态等于 MOTION_STATE_DESTROY_VEHICLE 代表请求销毁
	if state == proto.MotionState_MOTION_DESTROY_VEHICLE {
		gadgetEntity := entity.GetGadgetEntity()
		g.DestroyVehicleEntity(player, scene, gadgetEntity.GetGadgetVehicleEntity().GetVehicleId(), entity.GetId())
	}
}

// IsPlayerInVehicle 判断玩家是否在载具中
func (g *Game) IsPlayerInVehicle(player *model.Player, entity *Entity) bool {
	if entity.GetEntityType() != constant.ENTITY_TYPE_GADGET {
		return false
	}
	gadgetVehicleEntity := entity.GetGadgetEntity().GetGadgetVehicleEntity()
	if gadgetVehicleEntity == nil {
		return false
	}
	for _, p := range gadgetVehicleEntity.GetMemberMap() {
		if p == player {
			return true
		}
	}
	return false
}

// DestroyVehicleEntity 删除载具实体
func (g *Game) DestroyVehicleEntity(player *model.Player, scene *Scene, vehicleId uint32, entityId uint32) {
	entity := scene.GetEntity(entityId)
	if entity == nil {
		return
	}
	// 确保实体类型是否为载具
	gadgetEntity := entity.GetGadgetEntity()
	if gadgetEntity == nil || gadgetEntity.GetGadgetVehicleEntity() == nil {
		return
	}
	// 目前原神仅有一种载具 多载具目前理论上是兼容了 到时候有问题再改
	// 确保载具Id为将要创建的 (每种载具允许存在1个)
	if gadgetEntity.GetGadgetVehicleEntity().GetVehicleId() != vehicleId {
		return
	}
	// 该载具是否为此玩家的
	if gadgetEntity.GetGadgetVehicleEntity().GetOwnerUid() != player.PlayerId {
		return
	}
	// 如果玩家正在载具中
	if g.IsPlayerInVehicle(player, entity) {
		// 离开载具
		dbTeam := player.GetDbTeam()
		dbAvatar := player.GetDbAvatar()
		g.ExitVehicle(player, entity, dbAvatar.GetAvatarById(dbTeam.GetActiveAvatarId()).Guid)
	}
	// 删除已创建的载具
	scene.DestroyEntity(entity.GetId())
	g.RemoveSceneEntityNotifyBroadcast(scene, proto.VisionType_VISION_MISS, []uint32{entity.GetId()}, 0)
	// 删除玩家载具在线数据
	delete(player.VehicleInfo.CreateEntityIdMap, vehicleId)
}

// EnterVehicle 进入载具
func (g *Game) EnterVehicle(player *model.Player, entity *Entity, avatarGuid uint64) {
	gadgetEntity := entity.GetGadgetEntity()
	// 获取载具配置表
	vehicleDataConfig := gdconf.GetVehicleDataById(int32(gadgetEntity.GetGadgetVehicleEntity().GetVehicleId()))
	if vehicleDataConfig == nil {
		logger.Error("vehicle config error, vehicleId: %v", gadgetEntity.GetGadgetVehicleEntity().GetVehicleId())
		return
	}
	maxSlot := int(vehicleDataConfig.ConfigGadgetVehicle.Vehicle.MaxSeatCount)
	// 判断载具是否已满
	if len(gadgetEntity.GetGadgetVehicleEntity().GetMemberMap()) >= maxSlot {
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{}, proto.Retcode_RET_VEHICLE_SLOT_OCCUPIED)
		return
	}

	// 找出载具空闲的位置
	var freePos uint32
	for i := uint32(0); i < uint32(maxSlot); i++ {
		p := gadgetEntity.GetGadgetVehicleEntity().GetMemberMap()[i]
		// 玩家如果已进入载具重复记录不进行报错
		if p == player || p == nil {
			// 载具成员记录玩家
			gadgetEntity.GetGadgetVehicleEntity().GetMemberMap()[i] = player
			freePos = i
		}
	}

	// 记录玩家所在的载具信息
	player.VehicleInfo.InVehicleEntityId = entity.GetId()

	// PacketVehicleInteractRsp
	vehicleInteractRsp := &proto.VehicleInteractRsp{
		InteractType: proto.VehicleInteractType_VEHICLE_INTERACT_IN,
		Member: &proto.VehicleMember{
			Uid:        player.PlayerId,
			AvatarGuid: avatarGuid,
			Pos:        freePos, // 应该是多人坐船时的位置?
		},
		EntityId: entity.GetId(),
	}
	g.SendMsg(cmd.VehicleInteractRsp, player.PlayerId, player.ClientSeq, vehicleInteractRsp)
}

// ExitVehicle 离开载具
func (g *Game) ExitVehicle(player *model.Player, entity *Entity, avatarGuid uint64) {
	// 玩家是否进入载具
	gadgetEntity := entity.GetGadgetEntity()
	if !g.IsPlayerInVehicle(player, entity) {
		logger.Error("vehicle not has player, uid: %v", player.PlayerId)
		g.SendError(cmd.VehicleInteractRsp, player, &proto.VehicleInteractRsp{}, proto.Retcode_RET_NOT_IN_VEHICLE)
		return
	}
	// 载具成员删除玩家
	var memberPos uint32
	memberMap := gadgetEntity.GetGadgetVehicleEntity().GetMemberMap()
	for pos, p := range memberMap {
		if p == player {
			memberPos = pos
			delete(memberMap, pos)
		}
	}
	// 清除记录的所在载具信息
	player.VehicleInfo.InVehicleEntityId = 0

	// PacketVehicleInteractRsp
	vehicleInteractRsp := &proto.VehicleInteractRsp{
		InteractType: proto.VehicleInteractType_VEHICLE_INTERACT_OUT,
		Member: &proto.VehicleMember{
			Uid:        player.PlayerId,
			AvatarGuid: avatarGuid,
			Pos:        memberPos, // 应该是多人坐船时的位置?
		},
		EntityId: entity.GetId(),
	}
	g.SendMsg(cmd.VehicleInteractRsp, player.PlayerId, player.ClientSeq, vehicleInteractRsp)
}

/************************************************** 打包封装 **************************************************/
