package game

import (
	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/alg"
	"hk4e/pkg/logger"
	"hk4e/pkg/object"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	lua "github.com/yuin/gopher-lua"
)

type LuaCtx struct {
	uid            uint32
	ownerUid       uint32
	sourceEntityId uint32
	targetEntityId uint32
	groupId        uint32
}

type LuaEvt struct {
	param1         int32
	param2         int32
	param3         int32
	param4         int32
	paramStr1      string
	evtType        int32
	uid            uint32
	sourceName     string
	sourceEntityId uint32
	targetEntityId uint32
}

// CallLuaFunc 调用LUA方法
func CallLuaFunc(luaState *lua.LState, luaFuncName string, luaCtx *LuaCtx, luaEvt *LuaEvt) bool {
	GAME.EndlessLoopCheck(EndlessLoopCheckTypeCallLuaFunc)
	ctx := luaState.NewTable()
	luaState.SetField(ctx, "uid", lua.LNumber(luaCtx.uid))
	luaState.SetField(ctx, "owner_uid", lua.LNumber(luaCtx.ownerUid))
	luaState.SetField(ctx, "source_entity_id", lua.LNumber(luaCtx.sourceEntityId))
	luaState.SetField(ctx, "target_entity_id", lua.LNumber(luaCtx.targetEntityId))
	luaState.SetField(ctx, "groupId", lua.LNumber(luaCtx.groupId))
	evt := luaState.NewTable()
	luaState.SetField(evt, "param1", lua.LNumber(luaEvt.param1))
	luaState.SetField(evt, "param2", lua.LNumber(luaEvt.param2))
	luaState.SetField(evt, "param3", lua.LNumber(luaEvt.param3))
	luaState.SetField(evt, "param4", lua.LNumber(luaEvt.param4))
	luaState.SetField(evt, "param_str1", lua.LString(luaEvt.paramStr1))
	luaState.SetField(evt, "type", lua.LNumber(luaEvt.evtType))
	luaState.SetField(evt, "uid", lua.LNumber(luaEvt.uid))
	luaState.SetField(evt, "source_name", lua.LString(luaEvt.sourceName))
	luaState.SetField(evt, "source_eid", lua.LNumber(luaEvt.sourceEntityId))
	luaState.SetField(evt, "target_eid", lua.LNumber(luaEvt.targetEntityId))
	err := luaState.CallByParam(lua.P{
		Fn:      luaState.GetGlobal(luaFuncName),
		NRet:    1,
		Protect: true,
	}, ctx, evt)
	if err != nil {
		logger.Error("call lua error, groupId: %v, func: %v, error: %v", luaCtx.groupId, luaFuncName, err)
		return false
	}
	luaRet := luaState.Get(-1)
	luaState.Pop(1)
	switch luaRet.(type) {
	case lua.LBool:
		return bool(luaRet.(lua.LBool))
	case lua.LNumber:
		return object.ConvRetCodeToBool(int64(luaRet.(lua.LNumber)))
	default:
		return false
	}
}

// GetContextPlayer 获取上下文中的玩家对象
func GetContextPlayer(ctx *lua.LTable, luaState *lua.LState) *model.Player {
	uid, ok := luaState.GetField(ctx, "uid").(lua.LNumber)
	if !ok {
		return nil
	}
	player := USER_MANAGER.GetOnlineUser(uint32(uid))
	return player
}

// GetContextGroup 获取上下文中的场景组对象
func GetContextGroup(player *model.Player, ctx *lua.LTable, luaState *lua.LState) *Group {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return nil
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		return nil
	}
	scene := world.GetSceneById(player.GetSceneId())
	group := scene.GetGroupById(uint32(groupId))
	if group == nil {
		return nil
	}
	return group
}

// GetContextDbSceneGroup 获取上下文中的场景组离线数据对象
func GetContextDbSceneGroup(player *model.Player, groupId uint32) *model.DbSceneGroup {
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return nil
	}
	owner := world.GetOwner()
	dbWorld := owner.GetDbWorld()
	dbScene := dbWorld.GetSceneById(player.GetSceneId())
	dbSceneGroup := dbScene.GetSceneGroupById(groupId)
	return dbSceneGroup
}

// RegLuaScriptLibFunc 注册LUA侧ScriptLib调用的Golang方法
func RegLuaScriptLibFunc() {
	gdconf.RegScriptLibFunc("GetEntityType", GetEntityType)
	gdconf.RegScriptLibFunc("GetQuestState", GetQuestState)
	gdconf.RegScriptLibFunc("PrintLog", PrintLog)
	gdconf.RegScriptLibFunc("PrintContextLog", PrintContextLog)
	gdconf.RegScriptLibFunc("BeginCameraSceneLook", BeginCameraSceneLook)
	gdconf.RegScriptLibFunc("GetGroupMonsterCount", GetGroupMonsterCount)
	gdconf.RegScriptLibFunc("GetGroupMonsterCountByGroupId", GetGroupMonsterCountByGroupId)
	gdconf.RegScriptLibFunc("CheckRemainGadgetCountByGroupId", CheckRemainGadgetCountByGroupId)
	gdconf.RegScriptLibFunc("ChangeGroupGadget", ChangeGroupGadget)
	gdconf.RegScriptLibFunc("GetGadgetStateByConfigId", GetGadgetStateByConfigId)
	gdconf.RegScriptLibFunc("SetGadgetStateByConfigId", SetGadgetStateByConfigId)
	gdconf.RegScriptLibFunc("MarkPlayerAction", MarkPlayerAction)
	gdconf.RegScriptLibFunc("AddQuestProgress", AddQuestProgress)
	gdconf.RegScriptLibFunc("CreateMonster", CreateMonster)
	gdconf.RegScriptLibFunc("CreateGadget", CreateGadget)
	gdconf.RegScriptLibFunc("KillEntityByConfigId", KillEntityByConfigId)
	gdconf.RegScriptLibFunc("AddExtraGroupSuite", AddExtraGroupSuite)
	gdconf.RegScriptLibFunc("GetGroupVariableValue", GetGroupVariableValue)
	gdconf.RegScriptLibFunc("GetGroupVariableValueByGroup", GetGroupVariableValueByGroup)
	gdconf.RegScriptLibFunc("SetGroupVariableValue", SetGroupVariableValue)
	gdconf.RegScriptLibFunc("SetGroupVariableValueByGroup", SetGroupVariableValueByGroup)
	gdconf.RegScriptLibFunc("ChangeGroupVariableValue", ChangeGroupVariableValue)
	gdconf.RegScriptLibFunc("ChangeGroupVariableValueByGroup", ChangeGroupVariableValueByGroup)
	gdconf.RegScriptLibFunc("GetRegionEntityCount", GetRegionEntityCount)
	gdconf.RegScriptLibFunc("CreateGroupTimerEvent", CreateGroupTimerEvent)
	gdconf.RegScriptLibFunc("EnterWeatherArea", EnterWeatherArea)
	gdconf.RegScriptLibFunc("SetWeatherAreaState", SetWeatherAreaState)
	gdconf.RegScriptLibFunc("RefreshGroup", RefreshGroup)
	gdconf.RegScriptLibFunc("RemoveExtraGroupSuite", RemoveExtraGroupSuite)
	gdconf.RegScriptLibFunc("ShowReminder", ShowReminder)
}

type CommonLuaTableParam struct {
	ConfigId   int32 `json:"config_id"`
	DelayTime  int32 `json:"delay_time"`
	RegionEid  int32 `json:"region_eid"`
	EntityType int32 `json:"entity_type"`
	GroupId    int32 `json:"group_id"`
	Suite      int32 `json:"suite"`
}

func GetEntityType(luaState *lua.LState) int {
	entityId := luaState.ToInt(1)
	luaState.Push(lua.LNumber(entityId >> 24))
	return 1
}

func GetQuestState(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(constant.QUEST_STATE_NONE))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(constant.QUEST_STATE_NONE))
		return 1
	}
	entityId := luaState.ToInt(2)
	_ = entityId
	questId := luaState.ToInt(3)
	dbQuest := player.GetDbQuest()
	quest := dbQuest.GetQuestById(uint32(questId))
	if quest == nil {
		luaState.Push(lua.LNumber(constant.QUEST_STATE_NONE))
		return 1
	}
	luaState.Push(lua.LNumber(quest.State))
	return 1
}

func PrintLog(luaState *lua.LState) int {
	logInfo := luaState.ToString(1)
	logger.Info("[LUA LOG] %v", logInfo)
	return 0
}

func PrintContextLog(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	uid, ok := luaState.GetField(ctx, "uid").(lua.LNumber)
	if !ok {
		return 0
	}
	logInfo := luaState.ToString(2)
	logger.Info("[LUA CTX LOG] %v [UID: %v]", logInfo, uid)
	return 0
}

func BeginCameraSceneLook(luaState *lua.LState) int {
	// TODO 由于解锁风之翼任务相关原因暂时屏蔽
	luaState.Push(lua.LNumber(0))
	return 1
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	cameraLockInfo, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	ntf := new(proto.BeginCameraSceneLookNotify)
	gdconf.ParseLuaTableToObject(cameraLockInfo, ntf)
	GAME.SendMsg(cmd.BeginCameraSceneLookNotify, player.PlayerId, player.ClientSeq, ntf)
	logger.Debug("BeginCameraSceneLook, ntf: %v, uid: %v", ntf, player.PlayerId)
	luaState.Push(lua.LNumber(0))
	return 1
}

func GetGroupMonsterCount(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	group := GetContextGroup(player, ctx, luaState)
	if group == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	monsterCount := 0
	for _, entity := range group.GetAllEntity() {
		if entity.GetEntityType() == constant.ENTITY_TYPE_MONSTER {
			monsterCount++
		}
	}
	luaState.Push(lua.LNumber(monsterCount))
	return 1
}

func GetGroupMonsterCountByGroupId(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	scene := world.GetSceneById(player.GetSceneId())
	groupId := luaState.ToInt(2)
	group := scene.GetGroupById(uint32(groupId))
	if group == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	monsterCount := 0
	for _, entity := range group.GetAllEntity() {
		if entity.GetEntityType() == constant.ENTITY_TYPE_MONSTER {
			monsterCount++
		}
	}
	luaState.Push(lua.LNumber(monsterCount))
	return 1
}

func CheckRemainGadgetCountByGroupId(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	scene := world.GetSceneById(player.GetSceneId())
	luaTable, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTableParam := new(CommonLuaTableParam)
	gdconf.ParseLuaTableToObject[*CommonLuaTableParam](luaTable, luaTableParam)
	group := scene.GetGroupById(uint32(luaTableParam.GroupId))
	if group == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	gadgetCount := 0
	for _, entity := range group.GetAllEntity() {
		if entity.GetEntityType() == constant.ENTITY_TYPE_GADGET {
			gadgetCount++
		}
	}
	luaState.Push(lua.LNumber(gadgetCount))
	return 1
}

func ChangeGroupGadget(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	group := GetContextGroup(player, ctx, luaState)
	if group == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	gadgetInfo, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	gadgetStateInfo := new(gdconf.Gadget)
	gdconf.ParseLuaTableToObject(gadgetInfo, gadgetStateInfo)
	entity := group.GetEntityByConfigId(uint32(gadgetStateInfo.ConfigId))
	if entity == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	GAME.ChangeGadgetState(player, entity.GetId(), uint32(gadgetStateInfo.State))
	luaState.Push(lua.LNumber(0))
	return 1
}

func GetGadgetStateByConfigId(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId := luaState.ToInt(2)
	configId := luaState.ToInt(3)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	scene := world.GetSceneById(player.GetSceneId())
	group := scene.GetGroupById(uint32(groupId))
	if group == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	entity := group.GetEntityByConfigId(uint32(configId))
	if entity == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	if entity.GetEntityType() != constant.ENTITY_TYPE_GADGET {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	gadgetEntity := entity.GetGadgetEntity()
	luaState.Push(lua.LNumber(gadgetEntity.GetGadgetState()))
	return 1
}

func SetGadgetStateByConfigId(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	group := GetContextGroup(player, ctx, luaState)
	if group == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	configId := luaState.ToInt(2)
	state := luaState.ToInt(3)
	entity := group.GetEntityByConfigId(uint32(configId))
	GAME.ChangeGadgetState(player, entity.GetId(), uint32(state))
	luaState.Push(lua.LNumber(0))
	return 1
}

func MarkPlayerAction(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	param1 := luaState.ToInt(2)
	param2 := luaState.ToInt(3)
	param3 := luaState.ToInt(4)
	logger.Debug("[MarkPlayerAction] [%v %v %v] [UID: %v]", param1, param2, param3, player.PlayerId)
	luaState.Push(lua.LNumber(0))
	return 1
}

func AddQuestProgress(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	complexParam := luaState.ToString(2)
	GAME.TriggerQuest(player, constant.QUEST_FINISH_COND_TYPE_LUA_NOTIFY, complexParam)
	luaState.Push(lua.LNumber(0))
	return 1
}

func CreateMonster(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTable, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTableParam := new(CommonLuaTableParam)
	gdconf.ParseLuaTableToObject[*CommonLuaTableParam](luaTable, luaTableParam)
	TICK_MANAGER.CreateUserTimer(player.PlayerId, UserTimerActionLuaCreateMonster, uint32(luaTableParam.DelayTime),
		uint32(groupId), uint32(luaTableParam.ConfigId))
	luaState.Push(lua.LNumber(0))
	return 1
}

func CreateGadget(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTable, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTableParam := new(CommonLuaTableParam)
	gdconf.ParseLuaTableToObject[*CommonLuaTableParam](luaTable, luaTableParam)
	GAME.SceneGroupCreateEntity(player, uint32(groupId), uint32(luaTableParam.ConfigId), constant.ENTITY_TYPE_GADGET)
	luaState.Push(lua.LNumber(0))
	return 1
}

func KillEntityByConfigId(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTable, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTableParam := new(CommonLuaTableParam)
	gdconf.ParseLuaTableToObject[*CommonLuaTableParam](luaTable, luaTableParam)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	scene := world.GetSceneById(player.GetSceneId())
	group := scene.GetGroupById(uint32(groupId))
	if group == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	entity := group.GetEntityByConfigId(uint32(luaTableParam.ConfigId))
	if entity == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	GAME.KillEntity(player, scene, entity.GetId(), proto.PlayerDieType_PLAYER_DIE_NONE)
	luaState.Push(lua.LNumber(0))
	return 1
}

func AddExtraGroupSuite(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId := luaState.ToInt(2)
	suiteId := luaState.ToInt(3)
	GAME.AddSceneGroupSuite(player, uint32(groupId), uint8(suiteId))
	luaState.Push(lua.LNumber(0))
	return 1
}

func GetGroupVariableValue(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	name := luaState.ToString(2)
	dbSceneGroup := GetContextDbSceneGroup(player, uint32(groupId))
	if dbSceneGroup == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	value := dbSceneGroup.GetVariableByName(name)
	luaState.Push(lua.LNumber(value))
	return 1
}

func GetGroupVariableValueByGroup(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	name := luaState.ToString(2)
	groupId := luaState.ToInt(3)
	dbSceneGroup := GetContextDbSceneGroup(player, uint32(groupId))
	if dbSceneGroup == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	value := dbSceneGroup.GetVariableByName(name)
	luaState.Push(lua.LNumber(value))
	return 1
}

func SetGroupVariableValue(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	name := luaState.ToString(2)
	value := luaState.ToInt(3)
	dbSceneGroup := GetContextDbSceneGroup(player, uint32(groupId))
	if dbSceneGroup == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	dbSceneGroup.SetVariable(name, int32(value))
	luaState.Push(lua.LNumber(0))
	return 1
}

func SetGroupVariableValueByGroup(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	name := luaState.ToString(2)
	value := luaState.ToInt(3)
	groupId := luaState.ToInt(4)
	dbSceneGroup := GetContextDbSceneGroup(player, uint32(groupId))
	if dbSceneGroup == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	dbSceneGroup.SetVariable(name, int32(value))
	luaState.Push(lua.LNumber(0))
	return 1
}

func ChangeGroupVariableValue(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	name := luaState.ToString(2)
	change := luaState.ToInt(3)
	dbSceneGroup := GetContextDbSceneGroup(player, uint32(groupId))
	if dbSceneGroup == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	value := dbSceneGroup.GetVariableByName(name)
	dbSceneGroup.SetVariable(name, value+int32(change))
	luaState.Push(lua.LNumber(0))
	return 1
}

func ChangeGroupVariableValueByGroup(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	name := luaState.ToString(2)
	change := luaState.ToInt(3)
	groupId := luaState.ToInt(4)
	dbSceneGroup := GetContextDbSceneGroup(player, uint32(groupId))
	if dbSceneGroup == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	value := dbSceneGroup.GetVariableByName(name)
	dbSceneGroup.SetVariable(name, value+int32(change))
	luaState.Push(lua.LNumber(0))
	return 1
}

func GetRegionEntityCount(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId, ok := luaState.GetField(ctx, "groupId").(lua.LNumber)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTable, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTableParam := new(CommonLuaTableParam)
	gdconf.ParseLuaTableToObject[*CommonLuaTableParam](luaTable, luaTableParam)
	groupConfig := gdconf.GetSceneGroup(int32(groupId))
	if groupConfig == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	regionConfig := groupConfig.RegionMap[luaTableParam.RegionEid]
	if regionConfig == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
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
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	scene := world.GetSceneById(player.GetSceneId())
	count := 0
	for _, entity := range scene.GetAllEntity() {
		contain := shape.Contain(&alg.Vector3{X: float32(entity.GetPos().X), Y: float32(entity.GetPos().Y), Z: float32(entity.GetPos().Z)})
		if !contain {
			continue
		}
		if entity.GetEntityType() != uint8(luaTableParam.EntityType) {
			continue
		}
		count++
	}
	luaState.Push(lua.LNumber(count))
	return 1
}

func CreateGroupTimerEvent(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId := luaState.ToInt(2)
	source := luaState.ToString(3)
	delay := luaState.ToInt(4)
	TICK_MANAGER.CreateUserTimer(player.PlayerId, UserTimerActionLuaGroupTimerEvent, uint32(delay),
		uint32(groupId), source)
	luaState.Push(lua.LNumber(0))
	return 1
}

func EnterWeatherArea(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	weatherAreaId := luaState.ToInt(2)
	// 设置玩家天气
	climateType := GAME.GetWeatherAreaClimate(uint32(weatherAreaId))
	GAME.SetPlayerWeather(player, uint32(weatherAreaId), climateType)
	luaState.Push(lua.LNumber(0))
	return 1
}

func SetWeatherAreaState(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	weatherAreaId := luaState.ToInt(2)
	climateType := luaState.ToInt(3)
	GAME.SetPlayerWeather(player, uint32(weatherAreaId), uint32(climateType))
	luaState.Push(lua.LNumber(0))
	return 1
}

func RefreshGroup(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTable, ok := luaState.Get(2).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	luaTableParam := new(CommonLuaTableParam)
	gdconf.ParseLuaTableToObject[*CommonLuaTableParam](luaTable, luaTableParam)
	GAME.RefreshSceneGroupSuite(player, uint32(luaTableParam.GroupId), uint8(luaTableParam.Suite))
	luaState.Push(lua.LNumber(0))
	return 1
}

func RemoveExtraGroupSuite(luaState *lua.LState) int {
	ctx, ok := luaState.Get(1).(*lua.LTable)
	if !ok {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	player := GetContextPlayer(ctx, luaState)
	if player == nil {
		luaState.Push(lua.LNumber(-1))
		return 1
	}
	groupId := luaState.ToInt(2)
	suiteId := luaState.ToInt(3)
	GAME.RemoveSceneGroupSuite(player, uint32(groupId), uint8(suiteId))
	luaState.Push(lua.LNumber(0))
	return 1
}

func ShowReminder(luaState *lua.LState) int {
	luaState.Push(lua.LNumber(0))
	return 1
}
