package game

import (
	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/alg"
	"hk4e/pkg/logger"
)

func forEachPlayerSceneGroup(player *model.Player, handleFunc func(suiteConfig *gdconf.Suite, groupConfig *gdconf.Group)) {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	for groupId, group := range scene.GetAllGroup() {
		groupConfig := gdconf.GetSceneGroup(int32(groupId))
		if groupConfig == nil {
			logger.Error("get group config is nil, groupId: %v, uid: %v", groupId, player.PlayerId)
			continue
		}
		for suiteId := range group.GetAllSuite() {
			suiteConfig := groupConfig.SuiteMap[int32(suiteId)]
			handleFunc(suiteConfig, groupConfig)
		}
	}
}

func forEachPlayerSceneGroupTrigger(player *model.Player, handleFunc func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group)) {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	for groupId, group := range scene.GetAllGroup() {
		groupConfig := gdconf.GetSceneGroup(int32(groupId))
		if groupConfig == nil {
			logger.Error("get group config is nil, groupId: %v, uid: %v", groupId, player.PlayerId)
			continue
		}
		for suiteId := range group.GetAllSuite() {
			suiteConfig := groupConfig.SuiteMap[int32(suiteId)]
			for _, triggerName := range suiteConfig.TriggerNameList {
				triggerConfig := groupConfig.TriggerMap[triggerName]
				handleFunc(triggerConfig, groupConfig)
			}
		}
	}
}

func forEachGroupTrigger(player *model.Player, group *Group, handleFunc func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group)) {
	groupConfig := gdconf.GetSceneGroup(int32(group.GetId()))
	if groupConfig == nil {
		logger.Error("get group config is nil, groupId: %v, uid: %v", group.GetId(), player.PlayerId)
		return
	}
	for suiteId := range group.GetAllSuite() {
		suiteConfig := groupConfig.SuiteMap[int32(suiteId)]
		for _, triggerName := range suiteConfig.TriggerNameList {
			triggerConfig := groupConfig.TriggerMap[triggerName]
			handleFunc(triggerConfig, groupConfig)
		}
	}
}

// SceneRegionTriggerCheck 场景区域触发器检测
func (g *Game) SceneRegionTriggerCheck(player *model.Player, oldPos *model.Vector, newPos *model.Vector, entityId uint32) {
	forEachPlayerSceneGroup(player, func(suiteConfig *gdconf.Suite, groupConfig *gdconf.Group) {
		for _, regionConfigId := range suiteConfig.RegionConfigIdList {
			regionConfig := groupConfig.RegionMap[regionConfigId]
			if regionConfig == nil {
				continue
			}
			shape := alg.NewShape()
			switch uint8(regionConfig.Shape) {
			case constant.REGION_SHAPE_SPHERE:
				shape.NewSphere(&alg.Vector3{X: regionConfig.Pos.X, Y: regionConfig.Pos.Y, Z: regionConfig.Pos.Z}, regionConfig.Radius)
			case constant.REGION_SHAPE_CUBIC:
				shape.NewCubic(&alg.Vector3{X: regionConfig.Pos.X, Y: regionConfig.Pos.Y, Z: regionConfig.Pos.Z},
					&alg.Vector3{X: regionConfig.Size.X, Y: regionConfig.Size.Y, Z: regionConfig.Size.Z})
			case constant.REGION_SHAPE_CYLINDER:
				shape.NewCylinder(&alg.Vector3{X: regionConfig.Pos.X, Y: regionConfig.Pos.Y, Z: regionConfig.Pos.Z},
					regionConfig.Radius, regionConfig.Height)
			case constant.REGION_SHAPE_POLYGON:
				vector2PointArray := make([]*alg.Vector2, 0)
				for _, vector := range regionConfig.PointArray {
					// z就是y
					vector2PointArray = append(vector2PointArray, &alg.Vector2{X: vector.X, Z: vector.Y})
				}
				shape.NewPolygon(&alg.Vector3{X: regionConfig.Pos.X, Y: regionConfig.Pos.Y, Z: regionConfig.Pos.Z},
					vector2PointArray, regionConfig.Height)
			}
			oldPosInRegion := shape.Contain(&alg.Vector3{X: float32(oldPos.X), Y: float32(oldPos.Y), Z: float32(oldPos.Z)})
			newPosInRegion := shape.Contain(&alg.Vector3{X: float32(newPos.X), Y: float32(newPos.Y), Z: float32(newPos.Z)})
			if !oldPosInRegion && newPosInRegion {
				logger.Debug("player enter region: %v, uid: %v", regionConfig, player.PlayerId)
				for _, triggerName := range suiteConfig.TriggerNameList {
					triggerConfig := groupConfig.TriggerMap[triggerName]
					if triggerConfig.Event != constant.LUA_EVENT_ENTER_REGION {
						continue
					}
					if triggerConfig.Condition != "" {
						cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
							&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
							&LuaEvt{param1: regionConfig.ConfigId, targetEntityId: entityId, sourceEntityId: uint32(regionConfig.ConfigId)})
						if !cond {
							continue
						}
					}
					logger.Debug("scene group trigger fire, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
					if triggerConfig.Action != "" {
						logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
						ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
							&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
							&LuaEvt{})
						if !ok {
							logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
						}
					}
					for _, triggerDataConfig := range gdconf.GetTriggerDataMap() {
						if triggerDataConfig.TriggerName == triggerConfig.Name {
							g.TriggerQuest(player, constant.QUEST_FINISH_COND_TYPE_TRIGGER_FIRE, "", triggerDataConfig.TriggerId)
						}
					}
				}
			} else if oldPosInRegion && !newPosInRegion {
				logger.Debug("player leave region: %v, uid: %v", regionConfig, player.PlayerId)
				for _, triggerName := range suiteConfig.TriggerNameList {
					triggerConfig := groupConfig.TriggerMap[triggerName]
					if triggerConfig.Event != constant.LUA_EVENT_LEAVE_REGION {
						continue
					}
					if triggerConfig.Condition != "" {
						cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
							&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
							&LuaEvt{param1: regionConfig.ConfigId, targetEntityId: entityId, sourceEntityId: uint32(regionConfig.ConfigId)})
						if !cond {
							continue
						}
					}
					logger.Debug("scene group trigger fire, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
					if triggerConfig.Action != "" {
						logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
						ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
							&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
							&LuaEvt{})
						if !ok {
							logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
						}
					}
				}
			}
		}
	})
}

// QuestStartTriggerCheck 任务开始触发器检测
func (g *Game) QuestStartTriggerCheck(player *model.Player, questId uint32) {
	forEachPlayerSceneGroupTrigger(player, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_QUEST_START {
			return
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{param1: int32(questId)})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}

// MonsterCreateTriggerCheck 怪物创建触发器检测
func (g *Game) MonsterCreateTriggerCheck(player *model.Player, group *Group, configId uint32) {
	forEachGroupTrigger(player, group, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_ANY_MONSTER_LIVE {
			return
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{param1: int32(configId)})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}

// MonsterDieTriggerCheck 怪物死亡触发器检测
func (g *Game) MonsterDieTriggerCheck(player *model.Player, group *Group) {
	forEachGroupTrigger(player, group, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_ANY_MONSTER_DIE {
			return
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}

// GadgetCreateTriggerCheck 物件创建触发器检测
func (g *Game) GadgetCreateTriggerCheck(player *model.Player, group *Group, configId uint32) {
	forEachGroupTrigger(player, group, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_GADGET_CREATE {
			return
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{param1: int32(configId)})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}

// GadgetStateChangeTriggerCheck 物件状态变更触发器检测
func (g *Game) GadgetStateChangeTriggerCheck(player *model.Player, group *Group, configId uint32, state uint8) {
	forEachGroupTrigger(player, group, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_GADGET_STATE_CHANGE {
			return
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{param1: int32(state), param2: int32(configId)})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}

// GadgetDieTriggerCheck 物件死亡触发器检测
func (g *Game) GadgetDieTriggerCheck(player *model.Player, group *Group, configId uint32) {
	forEachGroupTrigger(player, group, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_ANY_GADGET_DIE {
			return
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{param1: int32(configId)})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}

// GroupLoadTriggerCheck 场景组加载触发器检测
func (g *Game) GroupLoadTriggerCheck(player *model.Player, group *Group) {
	forEachGroupTrigger(player, group, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_GROUP_LOAD {
			return
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}

// TimerEventTriggerCheck 场景组定时事件触发器检测
func (g *Game) TimerEventTriggerCheck(player *model.Player, group *Group, source string) {
	forEachGroupTrigger(player, group, func(triggerConfig *gdconf.Trigger, groupConfig *gdconf.Group) {
		if triggerConfig.Event != constant.LUA_EVENT_TIMER_EVENT {
			return
		}
		if triggerConfig.Source != "" {
			if triggerConfig.Source != source {
				return
			}
		}
		if triggerConfig.Condition != "" {
			cond := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Condition,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{sourceName: source})
			if !cond {
				return
			}
		}
		if triggerConfig.Action != "" {
			logger.Debug("scene group trigger do action, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			ok := CallLuaFunc(groupConfig.GetLuaState(), triggerConfig.Action,
				&LuaCtx{uid: player.PlayerId, groupId: uint32(groupConfig.Id)},
				&LuaEvt{})
			if !ok {
				logger.Error("trigger action fail, trigger: %+v, uid: %v", triggerConfig, player.PlayerId)
			}
		}
	})
}
