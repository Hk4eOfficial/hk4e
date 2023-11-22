package game

import (
	"time"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"
)

// 游戏服务器定时帧管理器

const (
	ServerTickTime = 50  // 服务器全局tick最小间隔毫秒
	UserTickTime   = 100 // 玩家自身tick最小间隔毫秒
)

type UserTimer struct {
	timeout int64
	action  int
	data    []any
}

type UserTick struct {
	globalTick      *time.Ticker
	globalTickCount uint64
	timerIdCounter  uint64
	timerMap        map[uint64]*UserTimer
}

type TickManager struct {
	globalTick      *time.Ticker
	globalTickCount uint64
	userTickMap     map[uint32]*UserTick
	tm              time.Time
}

func NewTickManager() (r *TickManager) {
	r = new(TickManager)
	r.globalTick = time.NewTicker(time.Millisecond * ServerTickTime)
	r.globalTickCount = 0
	r.userTickMap = make(map[uint32]*UserTick)
	r.tm = time.Now()
	logger.Info("game server tick start at: %v", time.Now().UnixMilli())
	return r
}

func (t *TickManager) GetGlobalTick() *time.Ticker {
	return t.globalTick
}

// 每个玩家自己的tick

// CreateUserGlobalTick 创建玩家tick对象
func (t *TickManager) CreateUserGlobalTick(userId uint32) {
	t.userTickMap[userId] = &UserTick{
		globalTick:      time.NewTicker(time.Millisecond * UserTickTime),
		globalTickCount: 0,
		timerIdCounter:  0,
		timerMap:        make(map[uint64]*UserTimer),
	}
}

// DestroyUserGlobalTick 销毁玩家tick对象
func (t *TickManager) DestroyUserGlobalTick(userId uint32) {
	delete(t.userTickMap, userId)
}

// CreateUserTimer 创建玩家定时任务
func (t *TickManager) CreateUserTimer(userId uint32, action int, delay uint32, data ...any) {
	userTick, exist := t.userTickMap[userId]
	if !exist {
		logger.Error("user not exist, uid: %v", userId)
		return
	}
	userTick.timerIdCounter++
	timeout := time.Now().UnixMilli() + int64(delay)*1000
	userTick.timerMap[userTick.timerIdCounter] = &UserTimer{
		timeout: timeout,
		action:  action,
		data:    data,
	}
	logger.Debug("create user timer, uid: %v, action: %v, time: %v",
		userId, action, time.Now().Add(time.Second*time.Duration(delay)).Format("2006-01-02 15:04:05"))
}

func (t *TickManager) onUserTickSecond(userId uint32, now int64) {
}

func (t *TickManager) onUserTickMinute(userId uint32, now int64) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	if userId < PlayerBaseUid {
		return
	}
	if uint32(now/1000)-player.LastKeepaliveTime > 60 {
		logger.Error("remove keepalive timeout user, uid: %v", userId)
		GAME.OnOffline(userId, &ChangeGsInfo{
			IsChangeGs: false,
		})
	}
}

// 玩家定时任务常量

const (
	UserTimerActionTest = iota
	UserTimerActionLuaCreateMonster
	UserTimerActionLuaGroupTimerEvent
	UserTimerActionPlugin
)

func (t *TickManager) userTimerHandle(userId uint32, action int, data []any) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		return
	}
	switch action {
	case UserTimerActionTest:
		logger.Debug("UserTimerActionTest, data: %v, uid: %v", data[0], userId)
	case UserTimerActionLuaCreateMonster:
		logger.Debug("UserTimerActionLuaCreateMonster, groupId: %v, configId: %v, uid: %v", data[0], data[1], userId)
		groupId := data[0].(uint32)
		configId := data[1].(uint32)
		GAME.SceneGroupCreateEntity(player, groupId, configId, constant.ENTITY_TYPE_MONSTER)
	case UserTimerActionLuaGroupTimerEvent:
		logger.Debug("UserTimerActionLuaGroupTimerEvent, groupId: %v, source: %v, uid: %v", data[0], data[1], userId)
		groupId := data[0].(uint32)
		source := data[1].(string)
		world := WORLD_MANAGER.GetWorldById(player.WorldId)
		if world == nil {
			logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, userId)
			return
		}
		scene := world.GetSceneById(player.GetSceneId())
		group := scene.GetGroupById(groupId)
		if group == nil {
			logger.Error("get group is nil, groupId: %v, uid: %v", groupId, userId)
			return
		}
		GAME.TimerEventTriggerCheck(player, group, source)
	case UserTimerActionPlugin:
		logger.Debug("UserTimerActionPlugin, data: %v", data)
		PLUGIN_MANAGER.HandleUserTimer(player, data)
	}
}

// 服务器全局tick

func (t *TickManager) OnGameServerTick() {
	t.globalTickCount++
	tm := time.Now()
	now := tm.UnixMilli()
	if t.globalTickCount%(50/ServerTickTime) == 0 {
		t.onTick50MilliSecond(now)
	}
	if t.globalTickCount%(100/ServerTickTime) == 0 {
		t.onTick100MilliSecond(now)
	}
	if t.globalTickCount%(200/ServerTickTime) == 0 {
		t.onTick200MilliSecond(now)
	}
	if t.globalTickCount%(1000/ServerTickTime) == 0 {
		t.onTickSecond(now)
		PLUGIN_MANAGER.HandleGlobalTick(PluginGlobalTickSecond)
	}
	if t.globalTickCount%(5000/ServerTickTime) == 0 {
		t.onTick5Second(now)
	}
	if t.globalTickCount%(10000/ServerTickTime) == 0 {
		t.onTick10Second(now)
	}
	if t.globalTickCount%(60000/ServerTickTime) == 0 {
		t.onTickMinute(now)
	}
	if t.globalTickCount%(60000*60/ServerTickTime) == 0 {
		t.onTickHour(now)
	}
	for userId, userTick := range t.userTickMap {
		if len(userTick.globalTick.C) == 0 {
			// 跳过还没到时间的定时器
			continue
		}
		<-userTick.globalTick.C
		userTick.globalTickCount++
		if userTick.globalTickCount%(1000/UserTickTime) == 0 {
			t.onUserTickSecond(userId, now)
		}
		if userTick.globalTickCount%(60000/UserTickTime) == 0 {
			t.onUserTickMinute(userId, now)
		}
		for timerId, timer := range userTick.timerMap {
			if now < timer.timeout {
				// 跳过还没到时间的定时器
				continue
			}
			delete(userTick.timerMap, timerId)
			t.userTimerHandle(userId, timer.action, timer.data)
		}
	}
	if tm.Minute() != t.tm.Minute() {
		t.onMinuteChange(now)
		PLUGIN_MANAGER.HandleGlobalTick(PluginGlobalTickMinuteChange)
	}
	if tm.Hour() != t.tm.Hour() {
		t.onHourChange(now)
	}
	if tm.Day() != t.tm.Day() {
		t.onDayChange(now)
	}
	if tm.Month() != t.tm.Month() {
		t.onMonthChange(now)
	}
	t.tm = tm
}

func (t *TickManager) onMonthChange(now int64) {
	logger.Info("on month change, time: %v", now)
}

func (t *TickManager) onDayChange(now int64) {
	logger.Info("on day change, time: %v", now)
}

func (t *TickManager) onHourChange(now int64) {
	logger.Info("on hour change, time: %v", now)
}

func (t *TickManager) onMinuteChange(now int64) {
}

func (t *TickManager) onTickHour(now int64) {
	logger.Info("on tick hour, time: %v", now)
}

func (t *TickManager) onTickMinute(now int64) {
	gdconf.LuaStateLruRemove()
	for _, world := range WORLD_MANAGER.GetAllWorld() {
		if world.GetOwner().SceneLoadState == model.SceneEnterDone {
			GAME.PlayerGameTimeNotify(world)
		}
	}
}

func (t *TickManager) onTick10Second(now int64) {
	for _, world := range WORLD_MANAGER.GetAllWorld() {
		if world.GetOwner().SceneLoadState == model.SceneEnterDone {
			GAME.SceneTimeNotify(world)
			GAME.PlayerTimeNotify(world)
		}
	}
}

func (t *TickManager) onTick5Second(now int64) {
	for _, world := range WORLD_MANAGER.GetAllWorld() {
		if world.GetOwner().SceneLoadState == model.SceneEnterDone {
			// 多人世界其他玩家的坐标位置广播
			GAME.WorldPlayerLocationNotify(world)
			GAME.ScenePlayerLocationNotify(world)
		}
	}
	iPlugin, err := PLUGIN_MANAGER.GetPlugin(&PluginPubg{})
	if err != nil {
		logger.Error("get plugin pubg error: %v", err)
		return
	}
	pluginPubg := iPlugin.(*PluginPubg)
	agree := true
	if pluginPubg.IsStartPubg() {
		agree = false
	}
	aiWorld := WORLD_MANAGER.GetAiWorld()
	if aiWorld.GetWorldPlayerNum() >= 100 {
		agree = false
	}
	for applyUid := range aiWorld.GetOwner().CoopApplyMap {
		GAME.PlayerDealEnterWorld(aiWorld.GetOwner(), applyUid, agree)
	}
}

func (t *TickManager) onTickSecond(now int64) {
	for _, world := range WORLD_MANAGER.GetAllWorld() {
		if world.GetOwner().SceneLoadState == model.SceneEnterDone {
			// 世界里所有玩家的网络延迟广播
			GAME.WorldPlayerRTTNotify(world)
		}
		// 每个场景时间+1
		for _, scene := range world.GetAllScene() {
			if world.GetOwner().Pause {
				continue
			}
			GAME.ChangeGameTime(scene, scene.GetGameTime()+1)
		}
	}
	// GCG游戏Tick
	for _, game := range GCG_MANAGER.gameMap {
		game.onTick()
	}
}

func (t *TickManager) onTick200MilliSecond(now int64) {
	for _, world := range WORLD_MANAGER.GetAllWorld() {
		for _, player := range world.GetAllPlayer() {
			if player.SceneLoadState == model.SceneEnterDone {
				// 耐力消耗
				GAME.SustainStaminaHandler(player)
				GAME.VehicleRestoreStaminaHandler(player)
			}
		}
	}
}

func (t *TickManager) onTick100MilliSecond(now int64) {
	for _, world := range WORLD_MANAGER.GetAllWorld() {
		for _, player := range world.GetAllPlayer() {
			if player.SceneLoadState == model.SceneEnterDone {
				// 耐力回复计数器
				GAME.RestoreCountStaminaHandler(player)
			}
		}
	}

	iPlugin, err := PLUGIN_MANAGER.GetPlugin(&PluginPubg{})
	if err != nil {
		logger.Error("get plugin pubg error: %v", err)
		return
	}
	pluginPubg := iPlugin.(*PluginPubg)
	if !pluginPubg.IsStartPubg() {
		return
	}
	world := WORLD_MANAGER.GetAiWorld()
	bulletPhysicsEngine := world.GetBulletPhysicsEngine()
	hitList := bulletPhysicsEngine.Update(now)
	for _, rigidBody := range hitList {
		scene := world.GetSceneById(rigidBody.sceneId)
		pluginPubg.PubgHit(scene, rigidBody.hitAvatarEntityId, rigidBody.avatarEntityId, true)
	}
}

func (t *TickManager) onTick50MilliSecond(now int64) {
	// 音乐播放器
	for i := 0; i < len(AUDIO_CHAN); i++ {
		world := WORLD_MANAGER.GetAiWorld()
		GAME.SendToWorldA(world, cmd.SceneAudioNotify, 0, &proto.SceneAudioNotify{
			Type:      5,
			SourceUid: world.GetOwner().PlayerId,
			Param1:    []uint32{1, <-AUDIO_CHAN},
			Param2:    nil,
			Param3:    nil,
		}, 0)
	}
}
