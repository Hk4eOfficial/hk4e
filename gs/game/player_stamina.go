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

// SceneAvatarStaminaStepReq 缓慢游泳或缓慢攀爬时消耗耐力
func (g *Game) SceneAvatarStaminaStepReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SceneAvatarStaminaStepReq)

	// 根据动作状态消耗耐力
	switch player.StaminaInfo.State {
	case proto.MotionState_MOTION_CLIMB:
		// 缓慢攀爬
		var angleRevise int32 // 角度修正值 归一化为-90到+90范围内的角
		// rotX ∈ [0,90) angle = rotX
		// rotX ∈ (270,360) angle = rotX - 360.0
		if req.Rot.X >= 0 && req.Rot.X < 90 {
			angleRevise = int32(req.Rot.X)
		} else if req.Rot.X > 270 && req.Rot.X < 360 {
			angleRevise = int32(req.Rot.X - 360.0)
		} else {
			logger.Error("invalid rot x angle: %v, uid: %v", req.Rot.X, player.PlayerId)
			g.SendError(cmd.SceneAvatarStaminaStepRsp, player, &proto.SceneAvatarStaminaStepRsp{})
			return
		}
		// 攀爬耐力修正曲线
		// angle >= 0 cost = -x + 10
		// angle < 0 cost = -2x + 10
		var costRevise int32 // 攀爬耐力修正值 在基础消耗值的水平上增加或减少
		if angleRevise >= 0 {
			// 普通或垂直斜坡
			costRevise = -angleRevise + 10
		} else {
			// 倒三角 非常消耗体力
			costRevise = -(angleRevise * 2) + 10
		}
		logger.Debug("stamina climbing, rotX: %v, costRevise: %v, cost: %v", req.Rot.X, costRevise, constant.STAMINA_COST_CLIMBING_BASE-costRevise)
		g.UpdatePlayerStamina(player, constant.STAMINA_COST_CLIMBING_BASE-costRevise)
	case proto.MotionState_MOTION_SWIM_MOVE:
		// 缓慢游泳
		g.UpdatePlayerStamina(player, constant.STAMINA_COST_SWIMMING)
	}

	sceneAvatarStaminaStepRsp := new(proto.SceneAvatarStaminaStepRsp)
	sceneAvatarStaminaStepRsp.UseClientRot = true
	sceneAvatarStaminaStepRsp.Rot = req.Rot
	g.SendMsg(cmd.SceneAvatarStaminaStepRsp, player.PlayerId, player.ClientSeq, sceneAvatarStaminaStepRsp)
}

/************************************************** 游戏功能 **************************************************/

// HandleAbilityStamina 处理来自ability的耐力消耗
func (g *Game) HandleAbilityStamina(player *model.Player, entry *proto.AbilityInvokeEntry) {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	// 获取世界中的角色实体
	worldAvatar := world.GetWorldAvatarByEntityId(entry.EntityId)
	if worldAvatar == nil {
		return
	}
	// 查找是不是属于该角色实体的ability id
	ability := worldAvatar.GetAbilityByInstanceId(entry.Head.InstancedAbilityId)
	if ability == nil {
		return
	}
	abilityNameHashCode := ability.AbilityName.GetHash()
	if abilityNameHashCode == 0 {
		return
	}
	// 根据ability name查找到对应的技能表里的技能配置
	staminaDataConfig := gdconf.GetSkillStaminaDataByAbilityHashCode(int32(abilityNameHashCode))
	if staminaDataConfig == nil {
		return
	}
	staminaInfo := player.StaminaInfo
	now := time.Now().UnixMilli()
	switch entry.ArgumentType {
	case proto.AbilityInvokeArgument_ABILITY_META_MODIFIER_CHANGE:
		// 普通角色重击耐力消耗
		// 距离技能开始过去的时间
		startPastTime := now - staminaInfo.LastSkillTime
		// 距离上次技能消耗的时间
		changePastTime := now - staminaInfo.LastCostStaminaTime
		// 法器角色轻击也会算触发重击消耗 胡桃等角色重击一次会多次消耗
		// 所以通过策略判断 必须距离技能开始过去200ms才算重击 两次技能耐力消耗之间需间隔500ms
		// 暂时就这样实现重击消耗 以后应该还会有更好的办法~
		if startPastTime > 200 && changePastTime > 500 {
			costStamina := -(staminaDataConfig.CostStamina * 100)
			logger.Debug("stamina cost, skillId: %v, cost: %v", staminaDataConfig.AvatarSkillId, costStamina)
			g.UpdatePlayerStamina(player, costStamina)
			staminaInfo.LastCostStaminaTime = now
		}
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_COST_STAMINA:
		// 大剑重击 或 持续技能 耐力消耗
		// 根据配置以及距离上次的时间计算消耗的耐力
		pastTime := now - staminaInfo.LastCostStaminaTime
		if pastTime > 500 {
			staminaInfo.LastCostStaminaTime = now
			pastTime = 0
		}
		costStamina := -(staminaDataConfig.CostStamina * 100)
		costStamina = int32(float64(pastTime) / 1000 * float64(costStamina))
		logger.Debug("stamina cost, skillId: %v, cost: %v", staminaDataConfig.AvatarSkillId, costStamina)
		g.UpdatePlayerStamina(player, costStamina)
		// 记录最后释放技能的时间
		staminaInfo.LastCostStaminaTime = now
	}
}

// ImmediateStamina 处理即时耐力消耗
func (g *Game) ImmediateStamina(player *model.Player, motionState proto.MotionState) {
	// 玩家暂停状态不更新耐力
	if player.Pause {
		return
	}
	staminaInfo := player.StaminaInfo
	// logger.Debug("stamina handle, uid: %v, motionState: %v", player.PlayerId, motionState)
	// 设置用于持续消耗或恢复耐力的值
	staminaInfo.SetStaminaCost(motionState)
	// 未改变状态不执行后面 有些仅在动作开始消耗耐力
	if motionState == staminaInfo.State {
		return
	}
	// 记录玩家的动作状态
	staminaInfo.State = motionState
	// 根据玩家的状态立刻消耗耐力
	switch motionState {
	case proto.MotionState_MOTION_CLIMB:
		// 攀爬开始
		g.UpdatePlayerStamina(player, constant.STAMINA_COST_CLIMB_START)
	case proto.MotionState_MOTION_DASH_BEFORE_SHAKE:
		// 冲刺
		g.UpdatePlayerStamina(player, constant.STAMINA_COST_SPRINT)
	case proto.MotionState_MOTION_CLIMB_JUMP:
		// 攀爬跳跃
		g.UpdatePlayerStamina(player, constant.STAMINA_COST_CLIMB_JUMP)
	case proto.MotionState_MOTION_SWIM_DASH:
		// 快速游泳开始
		g.UpdatePlayerStamina(player, constant.STAMINA_COST_SWIM_DASH_START)
	}
}

// SkillStartStamina 处理技能开始时的即时耐力消耗
func (g *Game) SkillStartStamina(player *model.Player, casterId uint32, skillId uint32) {
	staminaInfo := player.StaminaInfo
	// 记录最后释放的技能
	staminaInfo.LastSkillTime = time.Now().UnixMilli()
}

// RestoreCountStaminaHandler 处理耐力回复计数器
func (g *Game) RestoreCountStaminaHandler(player *model.Player) {
	// 玩家暂停状态不更新耐力
	if player.Pause {
		return
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	// 处理载具
	// 遍历玩家创建的载具实体
	for _, entityId := range player.VehicleInfo.CreateEntityIdMap {
		// 获取载具实体
		entity := scene.GetEntity(entityId)
		if entity == nil {
			continue
		}
		// 确保实体类型是否为载具
		gadgetEntity := entity.GetGadgetEntity()
		if gadgetEntity == nil || gadgetEntity.GetGadgetVehicleEntity() == nil {
			continue
		}
		// 获取载具配置表
		vehicleDataConfig := gdconf.GetVehicleDataById(int32(gadgetEntity.GetGadgetVehicleEntity().GetVehicleId()))
		if vehicleDataConfig == nil {
			logger.Error("vehicle config error, vehicleId: %v", gadgetEntity.GetGadgetVehicleEntity().GetVehicleId())
			continue
		}
		restoreDelay := gadgetEntity.GetGadgetVehicleEntity().GetRestoreDelay()
		// 做个限制不然一直加就panic了
		if restoreDelay < uint8(vehicleDataConfig.ConfigGadgetVehicle.Vehicle.Stamina.StaminaRecoverWaitTime*10) {
			gadgetEntity.GetGadgetVehicleEntity().SetRestoreDelay(restoreDelay + 1)
		}
	}
	// 处理玩家
	// 做个限制不然一直加就panic了
	if player.StaminaInfo.RestoreDelay < constant.STAMINA_PLAYER_RESTORE_DELAY {
		player.StaminaInfo.RestoreDelay++
	}
}

// VehicleRestoreStaminaHandler 处理载具持续回复耐力
func (g *Game) VehicleRestoreStaminaHandler(player *model.Player) {
	// 玩家暂停状态不更新耐力
	if player.Pause {
		return
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	// 遍历玩家创建的载具实体
	for _, entityId := range player.VehicleInfo.CreateEntityIdMap {
		// 获取载具实体
		entity := scene.GetEntity(entityId)
		if entity == nil {
			continue
		}
		// 判断玩家处于载具中
		if g.IsPlayerInVehicle(player, entity) {
			// 角色回复耐力
			g.UpdatePlayerStamina(player, constant.STAMINA_COST_IN_SKIFF)
		} else {
			// 载具回复耐力
			g.UpdateVehicleStamina(player, entity, constant.STAMINA_COST_SKIFF_NOBODY)
		}
	}
}

// SustainStaminaHandler 处理持续耐力消耗
func (g *Game) SustainStaminaHandler(player *model.Player) {
	// 玩家暂停状态不更新耐力
	if player.Pause {
		return
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	// 获取玩家处于的载具实体
	entity := scene.GetEntity(player.VehicleInfo.InVehicleEntityId)
	if entity == nil {
		// 更新玩家耐力
		g.UpdatePlayerStamina(player, player.StaminaInfo.CostStamina)
		return
	}
	// 确保实体类型是否为载具 且 根据玩家是否处于载具中更新耐力
	if g.IsPlayerInVehicle(player, entity) {
		// 更新载具耐力
		g.UpdateVehicleStamina(player, entity, player.StaminaInfo.CostStamina)
	} else {
		// 更新玩家耐力
		g.UpdatePlayerStamina(player, player.StaminaInfo.CostStamina)
	}
}

// GetChangeStamina 获取变更的耐力
// 当前耐力值 + 消耗的耐力值
func (g *Game) GetChangeStamina(curStamina int32, maxStamina int32, staminaCost int32) uint32 {
	// 即将更改为的耐力值
	stamina := curStamina + staminaCost
	// 确保耐力值不超出范围
	if stamina > maxStamina {
		stamina = maxStamina
	} else if stamina < 0 {
		stamina = 0
	}
	return uint32(stamina)
}

// UpdateVehicleStamina 更新载具耐力
func (g *Game) UpdateVehicleStamina(player *model.Player, vehicleEntity *Entity, staminaCost int32) {
	// 耐力消耗为0代表不更改 仍然执行后面的话会导致回复出问题
	if staminaCost == 0 {
		return
	}
	staminaInfo := player.StaminaInfo
	// 确保载具实体存在
	if vehicleEntity == nil {
		return
	}
	gadgetEntity := vehicleEntity.GetGadgetEntity()
	// 获取载具配置表
	vehicleDataConfig := gdconf.GetVehicleDataById(int32(gadgetEntity.GetGadgetVehicleEntity().GetVehicleId()))
	if vehicleDataConfig == nil {
		logger.Error("vehicle config error, vehicleId: %v", gadgetEntity.GetGadgetVehicleEntity().GetVehicleId())
		return
	}
	// 添加的耐力大于0为恢复
	if staminaCost > 0 {
		// 耐力延迟1.5s(15 ticks)恢复 动作状态为加速将立刻恢复耐力
		restoreDelay := gadgetEntity.GetGadgetVehicleEntity().GetRestoreDelay()
		if restoreDelay < uint8(vehicleDataConfig.ConfigGadgetVehicle.Vehicle.Stamina.StaminaRecoverWaitTime*10) && staminaInfo.State != proto.MotionState_MOTION_SKIFF_POWERED_DASH {
			return // 不恢复耐力
		}
	} else {
		// 消耗耐力重新计算恢复需要延迟的tick
		gadgetEntity.GetGadgetVehicleEntity().SetRestoreDelay(0)
	}
	// 因为载具的耐力需要换算
	// 这里先*100后面要用的时候再换算 为了确保精度
	// 最大耐力值
	maxStamina := int32(gadgetEntity.GetGadgetVehicleEntity().GetMaxStamina() * 100)
	// 现行耐力值
	curStamina := int32(gadgetEntity.GetGadgetVehicleEntity().GetCurStamina() * 100)
	// 将被变更的耐力
	stamina := g.GetChangeStamina(curStamina, maxStamina, staminaCost)
	// 当前无变动不要频繁发包
	if uint32(curStamina) == stamina {
		return
	}
	// 更改载具耐力 (换算)
	g.SetVehicleStamina(player, vehicleEntity, float32(stamina)/100)
}

// UpdatePlayerStamina 更新玩家耐力
func (g *Game) UpdatePlayerStamina(player *model.Player, staminaCost int32) {
	if player.StaminaInf && staminaCost < 0 {
		return
	}
	// 耐力消耗为0代表不更改 仍然执行后面的话会导致回复出问题
	if staminaCost == 0 {
		return
	}
	staminaInfo := player.StaminaInfo
	// 添加的耐力大于0为恢复
	if staminaCost > 0 {
		// 耐力延迟1.5s(15 ticks)恢复 动作状态为加速将立刻恢复耐力
		if staminaInfo.RestoreDelay < constant.STAMINA_PLAYER_RESTORE_DELAY && staminaInfo.State != proto.MotionState_MOTION_POWERED_FLY {
			return // 不恢复耐力
		}
	} else {
		// 消耗耐力重新计算恢复需要延迟的tick
		staminaInfo.RestoreDelay = 0
	}
	// 最大耐力值
	maxStamina := int32(player.PropMap[constant.PLAYER_PROP_MAX_STAMINA])
	// 现行耐力值
	curStamina := int32(player.PropMap[constant.PLAYER_PROP_CUR_PERSIST_STAMINA])
	// 将被变更的耐力
	stamina := g.GetChangeStamina(curStamina, maxStamina, staminaCost)
	// 检测玩家是否没耐力后执行溺水
	g.HandleDrown(player, stamina)
	// 当前无变动不要频繁发包
	if uint32(curStamina) == stamina {
		return
	}
	// 更改玩家的耐力
	g.SetPlayerStamina(player, stamina)
}

// HandleDrown 处理玩家溺水
func (g *Game) HandleDrown(player *model.Player, stamina uint32) {
	// 溺水需要耐力等于0
	if stamina != 0 {
		return
	}
	// 确保玩家正在游泳
	if player.StaminaInfo.State != proto.MotionState_MOTION_SWIM_MOVE && player.StaminaInfo.State != proto.MotionState_MOTION_SWIM_DASH {
		return
	}
	logger.Debug("player drown, curStamina: %v, state: %v", stamina, player.StaminaInfo.State)
	// 设置角色为死亡
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		return
	}
	for _, worldAvatar := range world.GetPlayerWorldAvatarList(player) {
		dbAvatar := player.GetDbAvatar()
		avatar := dbAvatar.GetAvatarById(worldAvatar.GetAvatarId())
		if avatar == nil {
			logger.Error("get avatar is nil, avatarId: %v", worldAvatar.GetAvatarId())
			continue
		}
		if avatar.LifeState != constant.LIFE_STATE_ALIVE {
			continue
		}
		g.KillPlayerAvatar(player, worldAvatar.GetAvatarId(), proto.PlayerDieType_PLAYER_DIE_DRAWN)
	}
}

// SetVehicleStamina 设置载具耐力
func (g *Game) SetVehicleStamina(player *model.Player, vehicleEntity *Entity, stamina float32) {
	// 设置载具的耐力
	gadgetEntity := vehicleEntity.GetGadgetEntity()
	gadgetEntity.GetGadgetVehicleEntity().SetCurStamina(stamina)
	// logger.Debug("vehicle stamina set, stamina: %v", stamina)

	vehicleStaminaNotify := new(proto.VehicleStaminaNotify)
	vehicleStaminaNotify.EntityId = vehicleEntity.GetId()
	vehicleStaminaNotify.CurStamina = stamina
	g.SendMsg(cmd.VehicleStaminaNotify, player.PlayerId, player.ClientSeq, vehicleStaminaNotify)
}

// SetPlayerStamina 设置玩家耐力
func (g *Game) SetPlayerStamina(player *model.Player, stamina uint32) {
	// 设置玩家的耐力
	player.PropMap[constant.PLAYER_PROP_CUR_PERSIST_STAMINA] = stamina
	// logger.Debug("player stamina set, stamina: %v", stamina)
	g.SendMsg(cmd.PlayerPropNotify, player.PlayerId, player.ClientSeq, g.PacketPlayerPropNotify(player, constant.PLAYER_PROP_CUR_PERSIST_STAMINA))
}

/************************************************** 打包封装 **************************************************/
