package game

import (
	"math"
	"time"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/alg"
	"hk4e/pkg/logger"
	"hk4e/pkg/random"
	"hk4e/protocol/proto"
)

// 世界管理器

const (
	ENTITY_NUM_UNLIMIT        = false // 是否不限制场景内实体数量
	ENTITY_MAX_SEND_NUM       = 10000 // 场景内最大实体数量
	MAX_MULTIPLAYER_WORLD_NUM = 10    // 本服务器最大多人世界数量
)

type WorldManager struct {
	worldMap            map[uint64]*World
	snowflake           *alg.SnowflakeWorker
	aiWorld             *World                     // 本服的Ai玩家世界
	sceneBlockAoiMap    map[uint32]*alg.AoiManager // 全局各场景地图的aoi管理器
	multiplayerWorldNum uint32                     // 本服务器的多人世界数量
}

func NewWorldManager(snowflake *alg.SnowflakeWorker) (r *WorldManager) {
	r = new(WorldManager)
	r.worldMap = make(map[uint64]*World)
	r.snowflake = snowflake
	r.LoadSceneBlockAoiMap()
	r.multiplayerWorldNum = 0
	return r
}

func (w *WorldManager) GetWorldById(worldId uint64) *World {
	return w.worldMap[worldId]
}

func (w *WorldManager) GetAllWorld() map[uint64]*World {
	return w.worldMap
}

func (w *WorldManager) CreateWorld(owner *model.Player) *World {
	worldId := uint64(w.snowflake.GenId())
	world := &World{
		id:                   worldId,
		owner:                owner,
		playerMap:            make(map[uint32]*model.Player),
		sceneMap:             make(map[uint32]*Scene),
		enterSceneToken:      uint32(random.GetRandomInt32(5000, 50000)),
		enterSceneContextMap: make(map[uint32]*EnterSceneContext),
		entityIdCounter:      0,
		worldLevel:           0,
		multiplayer:          false,
		mpLevelEntityId:      0,
		chatMsgList:          make([]*proto.ChatInfo, 0),
		playerFirstEnterMap:  make(map[uint32]int64),
		waitEnterPlayerMap:   make(map[uint32]int64),
		multiplayerTeam:      CreateMultiplayerTeam(),
		peerList:             make([]*model.Player, 0),
		aiWorldAoi:           nil,
		bulletPhysicsEngine:  nil,
	}
	world.mpLevelEntityId = world.GetNextWorldEntityId(constant.ENTITY_TYPE_MP_LEVEL)
	w.worldMap[worldId] = world

	if w.IsAiWorld(world) {
		aoiManager := alg.NewAoiManager()
		aoiManager.SetAoiRange(-8000, 4000, -200, 1000, -5500, 6500)
		aoiManager.Init3DRectAoiManager(120, 12, 120, true)
		world.aiWorldAoi = aoiManager
		logger.Info("ai world aoi init finish")
		world.NewPhysicsEngine()
	}

	return world
}

func (w *WorldManager) DestroyWorld(worldId uint64) {
	world := w.GetWorldById(worldId)
	for _, player := range world.playerMap {
		world.RemovePlayer(player)
		player.WorldId = 0
	}
	delete(w.worldMap, worldId)
	if world.multiplayer {
		w.multiplayerWorldNum--
	}
}

// GetAiWorld 获取本服务器的Ai世界
func (w *WorldManager) GetAiWorld() *World {
	return w.aiWorld
}

// InitAiWorld 初始化Ai世界
func (w *WorldManager) InitAiWorld(owner *model.Player) {
	w.aiWorld = w.GetWorldById(owner.WorldId)
}

func (w *WorldManager) IsAiWorld(world *World) bool {
	return world.owner.PlayerId < PlayerBaseUid
}

func (w *WorldManager) GetSceneBlockAoiMap() map[uint32]*alg.AoiManager {
	return w.sceneBlockAoiMap
}

func (w *WorldManager) LoadSceneBlockAoiMap() {
	w.sceneBlockAoiMap = make(map[uint32]*alg.AoiManager)
	for _, sceneLuaConfig := range gdconf.GetSceneLuaConfigMap() {
		// 检查各block大小是否相同 并提取出block大小
		minX := int32(math.MaxInt32)
		maxX := int32(math.MinInt32)
		minZ := int32(math.MaxInt32)
		maxZ := int32(math.MinInt32)
		blockXLen := uint32(0)
		blockZLen := uint32(0)
		ok := true
		for _, blockConfig := range sceneLuaConfig.BlockMap {
			if int32(blockConfig.BlockRange.Min.X) < minX {
				minX = int32(blockConfig.BlockRange.Min.X)
			}
			if int32(blockConfig.BlockRange.Max.X) > maxX {
				maxX = int32(blockConfig.BlockRange.Max.X)
			}
			if int32(blockConfig.BlockRange.Min.Z) < minZ {
				minZ = int32(blockConfig.BlockRange.Min.Z)
			}
			if int32(blockConfig.BlockRange.Max.Z) > maxZ {
				maxZ = int32(blockConfig.BlockRange.Max.Z)
			}
			xLen := uint32(int32(blockConfig.BlockRange.Max.X) - int32(blockConfig.BlockRange.Min.X))
			zLen := uint32(int32(blockConfig.BlockRange.Max.Z) - int32(blockConfig.BlockRange.Min.Z))
			if blockXLen == 0 {
				blockXLen = xLen
			} else {
				if blockXLen != xLen {
					logger.Error("scene block x len not same, scene id: %v", sceneLuaConfig.Id)
					ok = false
					break
				}
			}
			if blockZLen == 0 {
				blockZLen = zLen
			} else {
				if blockZLen != zLen {
					logger.Error("scene block z len not same, scene id: %v", sceneLuaConfig.Id)
					ok = false
					break
				}
			}
		}
		if !ok {
			continue
		}
		numX := uint32(0)
		if blockXLen == 0 {
			logger.Debug("scene block x len is zero, scene id: %v", sceneLuaConfig.Id)
			numX = 1
		} else {
			numX = uint32(maxX-minX) / blockXLen
		}
		numZ := uint32(0)
		if blockZLen == 0 {
			logger.Debug("scene block z len is zero, scene id: %v", sceneLuaConfig.Id)
			numZ = 1
		} else {
			numZ = uint32(maxZ-minZ) / blockZLen
		}
		// 将每个block作为aoi格子 并在格子中放入block拥有的所有group
		aoiManager := alg.NewAoiManager()
		aoiManager.SetAoiRange(minX, maxX, -1000, 1000, minZ, maxZ)
		aoiManager.Init3DRectAoiManager(numX, 1, numZ, true)
		for _, block := range sceneLuaConfig.BlockMap {
			for _, group := range block.GroupMap {
				aoiManager.AddObjectToGridByPos(int64(group.Id), group,
					group.Pos.X,
					0.0,
					group.Pos.Z)
			}
		}
		w.sceneBlockAoiMap[uint32(sceneLuaConfig.Id)] = aoiManager
	}
}

func (w *World) IsValidSceneBlockPos(sceneId uint32, x, y, z float32) bool {
	aoiManager, exist := WORLD_MANAGER.sceneBlockAoiMap[sceneId]
	if !exist {
		return false
	}
	return aoiManager.IsValidAoiPos(x, y, z)
}

func (w *World) IsValidAiWorldPos(sceneId uint32, x, y, z float32) bool {
	return w.aiWorldAoi.IsValidAoiPos(x, y, z)
}

func (w *WorldManager) GetMultiplayerWorldNum() uint32 {
	return w.multiplayerWorldNum
}

// EnterSceneContext 场景切换上下文数据结构
type EnterSceneContext struct {
	OldSceneId        uint32
	OldPos            *model.Vector
	NewSceneId        uint32
	NewPos            *model.Vector
	NewRot            *model.Vector
	OldDungeonPointId uint32
	Uid               uint32
}

// World 世界数据结构
type World struct {
	id                   uint64
	owner                *model.Player
	playerMap            map[uint32]*model.Player
	sceneMap             map[uint32]*Scene
	enterSceneToken      uint32
	enterSceneContextMap map[uint32]*EnterSceneContext // 场景切换上下文 key:EnterSceneToken value:EnterSceneContext
	entityIdCounter      uint32                        // 世界的实体id生成计数器
	worldLevel           uint8                         // 世界等级
	multiplayer          bool                          // 是否多人世界
	mpLevelEntityId      uint32                        // 多人世界等级实体id
	chatMsgList          []*proto.ChatInfo             // 世界聊天消息列表
	playerFirstEnterMap  map[uint32]int64              // 玩家第一次进入世界的时间 key:uid value:进入时间
	waitEnterPlayerMap   map[uint32]int64              // 进入世界的玩家等待列表 key:uid value:开始时间
	multiplayerTeam      *MultiplayerTeam              // 多人队伍
	peerList             []*model.Player               // 玩家编号列表
	aiWorldAoi           *alg.AoiManager               // ai世界的aoi管理器
	bulletPhysicsEngine  *PhysicsEngine                // 蓄力箭子弹物理引擎
}

func (w *World) GetBulletPhysicsEngine() *PhysicsEngine {
	return w.bulletPhysicsEngine
}

func (w *World) GetId() uint64 {
	return w.id
}

func (w *World) GetOwner() *model.Player {
	return w.owner
}

func (w *World) GetAllPlayer() map[uint32]*model.Player {
	return w.playerMap
}

func (w *World) GetAllScene() map[uint32]*Scene {
	return w.sceneMap
}

func (w *World) GetEnterSceneToken() uint32 {
	return w.enterSceneToken
}

func (w *World) GetEnterSceneContextByToken(token uint32) *EnterSceneContext {
	return w.enterSceneContextMap[token]
}

func (w *World) AddEnterSceneContext(ctx *EnterSceneContext) uint32 {
	w.enterSceneToken += 100
	w.enterSceneContextMap[w.enterSceneToken] = ctx
	return w.enterSceneToken
}

func (w *World) GetLastEnterSceneContextByUid(uid uint32) *EnterSceneContext {
	for token := w.enterSceneToken; token >= 5000; token -= 100 {
		ctx, exist := w.enterSceneContextMap[token]
		if !exist {
			continue
		}
		if ctx.Uid != uid {
			continue
		}
		return ctx
	}
	return nil
}

func (w *World) RemoveAllEnterSceneContextByUid(uid uint32) {
	for token := w.enterSceneToken; token >= 5000; token -= 100 {
		ctx, exist := w.enterSceneContextMap[token]
		if !exist {
			continue
		}
		if ctx.Uid != uid {
			continue
		}
		delete(w.enterSceneContextMap, token)
	}
}

func (w *World) GetWorldLevel() uint8 {
	return w.worldLevel
}

func (w *World) IsMultiplayerWorld() bool {
	return w.multiplayer
}

func (w *World) GetMpLevelEntityId() uint32 {
	return w.mpLevelEntityId
}

func (w *World) GetNextWorldEntityId(entityType uint8) uint32 {
	for {
		w.entityIdCounter++
		ret := (uint32(entityType) << 24) + w.entityIdCounter
		reTry := false
		for _, scene := range w.sceneMap {
			_, exist := scene.entityMap[ret]
			if exist {
				reTry = true
				break
			}
		}
		if reTry {
			continue
		} else {
			return ret
		}
	}
}

// GetPlayerPeerId 获取当前玩家世界内编号
func (w *World) GetPlayerPeerId(player *model.Player) uint32 {
	peerId := uint32(0)
	for peerIdIndex, worldPlayer := range w.peerList {
		if worldPlayer.PlayerId == player.PlayerId {
			peerId = uint32(peerIdIndex) + 1
		}
	}
	return peerId
}

// GetPlayerByPeerId 通过世界内编号获取玩家
func (w *World) GetPlayerByPeerId(peerId uint32) *model.Player {
	peerIdIndex := int(peerId) - 1
	if peerIdIndex >= len(w.peerList) {
		return nil
	}
	return w.peerList[peerIdIndex]
}

// GetWorldPlayerNum 获取世界中玩家的数量
func (w *World) GetWorldPlayerNum() int {
	return len(w.playerMap)
}

func (w *World) GetAiWorldAoi() *alg.AoiManager {
	return w.aiWorldAoi
}

func (w *World) AddPlayer(player *model.Player) {
	w.peerList = append(w.peerList, player)
	w.playerMap[player.PlayerId] = player

	// 将玩家自身当前的队伍角色信息复制到世界的玩家本地队伍
	dbTeam := player.GetDbTeam()
	team := dbTeam.GetActiveTeam()
	if WORLD_MANAGER.IsAiWorld(w) {
		w.SetPlayerLocalTeam(player, []uint32{dbTeam.GetActiveAvatarId()})
	} else {
		w.SetPlayerLocalTeam(player, team.GetAvatarIdList())
	}
	w.SetPlayerActiveAvatarId(player, dbTeam.GetActiveAvatarId())
	if WORLD_MANAGER.IsAiWorld(w) {
		w.AddMultiplayerTeam(player)
	} else {
		w.UpdateMultiplayerTeam()
	}

	scene := w.GetSceneById(player.GetSceneId())
	scene.AddPlayer(player)
	w.InitPlayerTeamEntityId(player)
}

func (w *World) RemovePlayer(player *model.Player) {
	peerId := w.GetPlayerPeerId(player)
	w.peerList = append(w.peerList[:peerId-1], w.peerList[peerId:]...)
	scene := w.sceneMap[player.GetSceneId()]
	scene.RemovePlayer(player)
	w.RemoveAllEnterSceneContextByUid(player.PlayerId)
	delete(w.playerMap, player.PlayerId)
	delete(w.playerFirstEnterMap, player.PlayerId)
	delete(w.multiplayerTeam.localTeamMap, player.PlayerId)
	delete(w.multiplayerTeam.localTeamEntityMap, player.PlayerId)
	delete(w.multiplayerTeam.localActiveAvatarMap, player.PlayerId)
	if WORLD_MANAGER.IsAiWorld(w) {
		w.RemoveMultiplayerTeam(player)
	} else {
		if player.PlayerId != w.owner.PlayerId {
			w.UpdateMultiplayerTeam()
		}
	}
}

// WorldAvatar 世界角色
type WorldAvatar struct {
	uid            uint32
	avatarId       uint32
	avatarEntityId uint32
	weaponEntityId uint32
	abilityMap     map[uint32]*proto.AbilityAppliedAbility
	modifierMap    map[uint32]*proto.AbilityAppliedModifier
}

func (w *WorldAvatar) GetUid() uint32 {
	return w.uid
}

func (w *WorldAvatar) GetAvatarId() uint32 {
	return w.avatarId
}

func (w *WorldAvatar) GetAvatarEntityId() uint32 {
	return w.avatarEntityId
}

func (w *WorldAvatar) GetWeaponEntityId() uint32 {
	return w.weaponEntityId
}

func (w *WorldAvatar) SetWeaponEntityId(weaponEntityId uint32) {
	w.weaponEntityId = weaponEntityId
}

func (w *WorldAvatar) GetAbilityList() []*proto.AbilityAppliedAbility {
	abilityList := make([]*proto.AbilityAppliedAbility, 0)
	for _, ability := range w.abilityMap {
		abilityList = append(abilityList, ability)
	}
	return abilityList
}

func (w *WorldAvatar) GetAbilityByInstanceId(instanceId uint32) *proto.AbilityAppliedAbility {
	return w.abilityMap[instanceId]
}

func (w *WorldAvatar) AddAbility(ability *proto.AbilityAppliedAbility) {
	w.abilityMap[ability.InstancedAbilityId] = ability
}

func (w *WorldAvatar) GetModifierList() []*proto.AbilityAppliedModifier {
	modifierList := make([]*proto.AbilityAppliedModifier, 0)
	for _, modifier := range w.modifierMap {
		modifierList = append(modifierList, modifier)
	}
	return modifierList
}

func (w *WorldAvatar) AddModifier(modifier *proto.AbilityAppliedModifier) {
	w.modifierMap[modifier.InstancedModifierId] = modifier
}

// GetWorldAvatarList 获取世界队伍的全部角色列表
func (w *World) GetWorldAvatarList() []*WorldAvatar {
	worldAvatarList := make([]*WorldAvatar, 0)
	for _, worldAvatar := range w.multiplayerTeam.worldTeam {
		if worldAvatar.uid == 0 {
			continue
		}
		worldAvatarList = append(worldAvatarList, worldAvatar)
	}
	return worldAvatarList
}

// GetPlayerWorldAvatar 获取某玩家在世界队伍中的某角色
func (w *World) GetPlayerWorldAvatar(player *model.Player, avatarId uint32) *WorldAvatar {
	for _, worldAvatar := range w.GetWorldAvatarList() {
		if worldAvatar.uid == player.PlayerId && worldAvatar.avatarId == avatarId {
			return worldAvatar
		}
	}
	return nil
}

// GetPlayerWorldAvatarList 获取某玩家在世界队伍中的所有角色列表
func (w *World) GetPlayerWorldAvatarList(player *model.Player) []*WorldAvatar {
	worldAvatarList := make([]*WorldAvatar, 0)
	for _, worldAvatar := range w.GetWorldAvatarList() {
		if worldAvatar.uid == player.PlayerId {
			worldAvatarList = append(worldAvatarList, worldAvatar)
		}
	}
	return worldAvatarList
}

// GetWorldAvatarByEntityId 通过场景实体id获取世界队伍中的角色
func (w *World) GetWorldAvatarByEntityId(avatarEntityId uint32) *WorldAvatar {
	for _, worldAvatar := range w.GetWorldAvatarList() {
		if worldAvatar.avatarEntityId == avatarEntityId {
			return worldAvatar
		}
	}
	return nil
}

// UpdatePlayerWorldAvatar 更新某玩家在世界队伍中的所有角色
func (w *World) UpdatePlayerWorldAvatar(player *model.Player) {
	scene := w.GetSceneById(player.GetSceneId())
	for _, worldAvatar := range w.GetPlayerWorldAvatarList(player) {
		if worldAvatar.avatarEntityId != 0 {
			continue
		}
		worldAvatar.avatarEntityId = scene.CreateEntityAvatar(player, worldAvatar.avatarId)
		worldAvatar.weaponEntityId = scene.CreateEntityWeapon(player.GetPos(), player.GetRot())
	}
}

// GetPlayerTeamEntityId 获取某玩家的本地队伍实体id
func (w *World) GetPlayerTeamEntityId(player *model.Player) uint32 {
	return w.multiplayerTeam.localTeamEntityMap[player.PlayerId]
}

// InitPlayerTeamEntityId 初始化某玩家的本地队伍实体id
func (w *World) InitPlayerTeamEntityId(player *model.Player) {
	w.multiplayerTeam.localTeamEntityMap[player.PlayerId] = w.GetNextWorldEntityId(constant.ENTITY_TYPE_TEAM)
}

// GetPlayerWorldAvatarEntityId 获取某玩家在世界队伍中的某角色的实体id
func (w *World) GetPlayerWorldAvatarEntityId(player *model.Player, avatarId uint32) uint32 {
	worldAvatar := w.GetPlayerWorldAvatar(player, avatarId)
	if worldAvatar == nil {
		return 0
	}
	return worldAvatar.avatarEntityId
}

// GetPlayerWorldAvatarWeaponEntityId 获取某玩家在世界队伍中的某角色的武器的实体id
func (w *World) GetPlayerWorldAvatarWeaponEntityId(player *model.Player, avatarId uint32) uint32 {
	worldAvatar := w.GetPlayerWorldAvatar(player, avatarId)
	if worldAvatar == nil {
		return 0
	}
	return worldAvatar.weaponEntityId
}

// GetPlayerActiveAvatarId 获取玩家当前活跃角色id
func (w *World) GetPlayerActiveAvatarId(player *model.Player) uint32 {
	return w.multiplayerTeam.localActiveAvatarMap[player.PlayerId]
}

// SetPlayerActiveAvatarId 设置玩家当前活跃角色id
func (w *World) SetPlayerActiveAvatarId(player *model.Player, avatarId uint32) {
	localTeam := w.GetPlayerLocalTeam(player)
	exist := false
	for _, worldAvatar := range localTeam {
		if worldAvatar.avatarId == avatarId {
			exist = true
			break
		}
	}
	if !exist {
		return
	}
	w.multiplayerTeam.localActiveAvatarMap[player.PlayerId] = avatarId
}

// GetPlayerAvatarIndexByAvatarId 获取玩家某角色的索引
func (w *World) GetPlayerAvatarIndexByAvatarId(player *model.Player, avatarId uint32) int {
	localTeam := w.GetPlayerLocalTeam(player)
	for index, worldAvatar := range localTeam {
		if worldAvatar.avatarId == avatarId {
			return index
		}
	}
	return -1
}

// GetPlayerActiveAvatarEntity 获取玩家当前活跃角色场景实体
func (w *World) GetPlayerActiveAvatarEntity(player *model.Player) *Entity {
	activeAvatarId := w.GetPlayerActiveAvatarId(player)
	avatarEntityId := w.GetPlayerWorldAvatarEntityId(player, activeAvatarId)
	scene := w.GetSceneById(player.GetSceneId())
	entity := scene.GetEntity(avatarEntityId)
	return entity
}

// IsPlayerActiveAvatarEntity 是否为玩家当前活跃角色场景实体
func (w *World) IsPlayerActiveAvatarEntity(player *model.Player, entityId uint32) bool {
	entity := w.GetPlayerActiveAvatarEntity(player)
	if entity == nil {
		return false
	}
	return entity.GetId() == entityId
}

type MultiplayerTeam struct {
	// key:uid value:玩家的本地队伍
	localTeamMap map[uint32][]*WorldAvatar
	// key:uid value:玩家的本地队伍实体id
	localTeamEntityMap map[uint32]uint32
	// key:uid value:玩家当前活跃角色id
	localActiveAvatarMap map[uint32]uint32
	// 最终的世界队伍
	worldTeam []*WorldAvatar
}

func CreateMultiplayerTeam() (r *MultiplayerTeam) {
	r = new(MultiplayerTeam)
	r.localTeamMap = make(map[uint32][]*WorldAvatar)
	r.localTeamEntityMap = make(map[uint32]uint32)
	r.localActiveAvatarMap = make(map[uint32]uint32)
	r.worldTeam = make([]*WorldAvatar, 0)
	return r
}

func (w *World) GetPlayerLocalTeam(player *model.Player) []*WorldAvatar {
	return w.multiplayerTeam.localTeamMap[player.PlayerId]
}

func (w *World) SetPlayerLocalTeam(player *model.Player, avatarIdList []uint32) {
	oldLocalTeam := w.multiplayerTeam.localTeamMap[player.PlayerId]
	sameAvatarIdList := make([]uint32, 0)
	addAvatarIdList := make([]uint32, 0)
	for _, avatarId := range avatarIdList {
		exist := false
		for _, worldAvatar := range oldLocalTeam {
			if worldAvatar.avatarId == avatarId {
				exist = true
			}
		}
		if exist {
			sameAvatarIdList = append(sameAvatarIdList, avatarId)
		} else {
			addAvatarIdList = append(addAvatarIdList, avatarId)
		}
	}
	newLocalTeam := make([]*WorldAvatar, len(avatarIdList))
	for _, avatarId := range sameAvatarIdList {
		for _, worldAvatar := range oldLocalTeam {
			if worldAvatar.avatarId == avatarId {
				index := 0
				for i, v := range avatarIdList {
					if avatarId == v {
						index = i
					}
				}
				newLocalTeam[index] = worldAvatar
			}
		}
	}
	for _, avatarId := range addAvatarIdList {
		index := 0
		for i, v := range avatarIdList {
			if avatarId == v {
				index = i
			}
		}
		newLocalTeam[index] = &WorldAvatar{
			uid:            player.PlayerId,
			avatarId:       avatarId,
			avatarEntityId: 0,
			weaponEntityId: 0,
			abilityMap:     make(map[uint32]*proto.AbilityAppliedAbility),
			modifierMap:    make(map[uint32]*proto.AbilityAppliedModifier),
		}
	}
	scene := w.GetSceneById(player.GetSceneId())
	for _, worldAvatar := range oldLocalTeam {
		exist := false
		for _, avatarId := range avatarIdList {
			if worldAvatar.avatarId == avatarId {
				exist = true
			}
		}
		if !exist {
			scene.DestroyEntity(worldAvatar.avatarEntityId)
			scene.DestroyEntity(worldAvatar.weaponEntityId)
		}
	}
	w.multiplayerTeam.localTeamMap[player.PlayerId] = newLocalTeam
}

// 为了实现大世界无限人数写的
// 现在看来把世界里所有人放进队伍里发给客户端超过8个客户端会崩溃
// 看来还是不能简单的走通用逻辑 需要对大世界场景队伍做特殊处理 欺骗客户端其他玩家仅仅以场景角色实体的形式出现

func (w *World) AddMultiplayerTeam(player *model.Player) {
	localTeam := w.GetPlayerLocalTeam(player)
	w.multiplayerTeam.worldTeam = append(w.multiplayerTeam.worldTeam, localTeam...)
}

func (w *World) RemoveMultiplayerTeam(player *model.Player) {
	worldTeam := make([]*WorldAvatar, 0)
	for _, worldAvatar := range w.multiplayerTeam.worldTeam {
		if worldAvatar.uid == player.PlayerId {
			continue
		}
		worldTeam = append(worldTeam, worldAvatar)
	}
	w.multiplayerTeam.worldTeam = worldTeam
}

// UpdateMultiplayerTeam 整合所有玩家的本地队伍计算出世界队伍
func (w *World) UpdateMultiplayerTeam() {
	playerNum := w.GetWorldPlayerNum()
	if playerNum > 4 {
		return
	}
	w.multiplayerTeam.worldTeam = make([]*WorldAvatar, 4)
	switch playerNum {
	case 1:
		// 1P*4
		p1 := w.GetPlayerByPeerId(1)
		p1LocalTeam := w.GetPlayerLocalTeam(p1)
		for index := 0; index <= 3; index++ {
			worldAvatar := &WorldAvatar{
				uid:            0,
				avatarId:       0,
				avatarEntityId: 0,
				weaponEntityId: 0,
				abilityMap:     nil,
				modifierMap:    nil,
			}
			if index < len(p1LocalTeam) {
				worldAvatar = p1LocalTeam[index]
			}
			w.multiplayerTeam.worldTeam[index] = worldAvatar
		}
	case 2:
		// 1P*2 + 2P*2
		for index := 0; index <= 3; index++ {
			switch index {
			case 0:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(1, true)
			case 1:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(1, false)
			case 2:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(2, true)
			case 3:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(2, false)
			}
		}
	case 3:
		// 1P*2 + 2P*1 + 3P*1
		for index := 0; index <= 3; index++ {
			switch index {
			case 0:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(1, true)
			case 1:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(1, false)
			case 2:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(2, true)
			case 3:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(3, true)
			}
		}
	case 4:
		// 1P*1 + 2P*1 + 3P*1 + 4P*1
		for index := 0; index <= 3; index++ {
			switch index {
			case 0:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(1, true)
			case 1:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(2, true)
			case 2:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(3, true)
			case 3:
				w.multiplayerTeam.worldTeam[index] = w.SelectPlayerWorldAvatar(4, true)
			}
		}
	}
}

func (w *World) SelectPlayerWorldAvatar(peerId uint32, active bool) *WorldAvatar {
	worldAvatar := &WorldAvatar{
		uid:            0,
		avatarId:       0,
		avatarEntityId: 0,
		weaponEntityId: 0,
		abilityMap:     nil,
		modifierMap:    nil,
	}
	player := w.GetPlayerByPeerId(peerId)
	localTeam := w.GetPlayerLocalTeam(player)
	activeAvatarId := w.GetPlayerActiveAvatarId(player)
	for _, wa := range localTeam {
		if active {
			if wa.avatarId == activeAvatarId {
				worldAvatar = wa
				break
			}
		} else {
			if wa.avatarId != activeAvatarId {
				worldAvatar = wa
				break
			}
		}
	}
	return worldAvatar
}

// 世界聊天

func (w *World) AddChat(chatInfo *proto.ChatInfo) {
	if len(w.chatMsgList) > 100 {
		w.chatMsgList = w.chatMsgList[1:]
	}
	w.chatMsgList = append(w.chatMsgList, chatInfo)
	chatMsg := GAME.ConvChatInfoToChatMsg(chatInfo)
	chatMsg.IsDelete = true
	go USER_MANAGER.SaveUserChatMsgToDbSync(chatMsg)
}

func (w *World) GetChatList() []*proto.ChatInfo {
	return w.chatMsgList
}

// ChangeToMultiplayer 转换为多人世界
func (w *World) ChangeToMultiplayer() {
	WORLD_MANAGER.multiplayerWorldNum++
	w.multiplayer = true
	w.owner.IsInMp = true
}

// IsPlayerFirstEnter 获取玩家是否首次加入本世界
func (w *World) IsPlayerFirstEnter(player *model.Player) bool {
	_, exist := w.playerFirstEnterMap[player.PlayerId]
	if !exist {
		return true
	} else {
		return false
	}
}

func (w *World) PlayerEnter(uid uint32) {
	w.playerFirstEnterMap[uid] = time.Now().UnixMilli()
}

func (w *World) AddWaitPlayer(uid uint32) {
	w.waitEnterPlayerMap[uid] = time.Now().UnixMilli()
}

func (w *World) GetAllWaitPlayer() []uint32 {
	uidList := make([]uint32, 0)
	for uid := range w.waitEnterPlayerMap {
		uidList = append(uidList, uid)
	}
	return uidList
}

func (w *World) RemoveWaitPlayer(uid uint32) {
	delete(w.waitEnterPlayerMap, uid)
}

func (w *World) CreateScene(sceneId uint32) *Scene {
	scene := &Scene{
		id:         sceneId,
		world:      w,
		playerMap:  make(map[uint32]*model.Player),
		entityMap:  make(map[uint32]*Entity),
		groupMap:   make(map[uint32]*Group),
		gameTime:   0,
		createTime: time.Now().UnixMilli(),
		meeoIndex:  0,
	}
	w.sceneMap[sceneId] = scene
	return scene
}

func (w *World) GetSceneById(sceneId uint32) *Scene {
	// 场景是取时创建 可以简化代码不判空
	scene, exist := w.sceneMap[sceneId]
	if !exist {
		scene = w.CreateScene(sceneId)
	}
	return scene
}
