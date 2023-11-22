package game

import (
	"math"
	"time"

	"hk4e/gdconf"

	"hk4e/common/constant"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/proto"
)

// Scene 场景数据结构
type Scene struct {
	id          uint32
	world       *World
	playerMap   map[uint32]*model.Player
	entityMap   map[uint32]*Entity // 场景中全部的实体
	groupMap    map[uint32]*Group  // 场景中按group->suite分类的实体
	gameTime    uint32             // 游戏内提瓦特大陆的时间
	createTime  int64              // 场景创建时间
	meeoIndex   uint32             // 客户端风元素染色同步协议的计数器
	monsterWudi bool               // 是否开启场景内怪物无敌
}

func (s *Scene) GetId() uint32 {
	return s.id
}

func (s *Scene) GetWorld() *World {
	return s.world
}

func (s *Scene) GetAllPlayer() map[uint32]*model.Player {
	return s.playerMap
}

func (s *Scene) GetAllEntity() map[uint32]*Entity {
	return s.entityMap
}

func (s *Scene) GetGroupById(groupId uint32) *Group {
	return s.groupMap[groupId]
}

func (s *Scene) GetAllGroup() map[uint32]*Group {
	return s.groupMap
}

func (s *Scene) GetGameTime() uint32 {
	return s.gameTime
}

func (s *Scene) GetMeeoIndex() uint32 {
	return s.meeoIndex
}

func (s *Scene) SetMeeoIndex(meeoIndex uint32) {
	s.meeoIndex = meeoIndex
}

func (s *Scene) GetMonsterWudi() bool {
	return s.monsterWudi
}

func (s *Scene) SetMonsterWudi(monsterWudi bool) {
	s.monsterWudi = monsterWudi
}

func (s *Scene) ChangeGameTime(time uint32) {
	s.gameTime = time % 1440
}

func (s *Scene) GetSceneCreateTime() int64 {
	return s.createTime
}

func (s *Scene) GetSceneTime() int64 {
	now := time.Now().UnixMilli()
	return now - s.createTime
}

func (s *Scene) AddPlayer(player *model.Player) {
	s.playerMap[player.PlayerId] = player
	for _, worldAvatar := range s.world.GetPlayerWorldAvatarList(player) {
		worldAvatar.avatarEntityId = s.CreateEntityAvatar(player, worldAvatar.avatarId)
		worldAvatar.weaponEntityId = s.CreateEntityWeapon(player.GetPos(), player.GetRot())
	}
}

func (s *Scene) RemovePlayer(player *model.Player) {
	delete(s.playerMap, player.PlayerId)
	worldAvatarList := s.world.GetPlayerWorldAvatarList(player)
	for _, worldAvatar := range worldAvatarList {
		s.DestroyEntity(worldAvatar.avatarEntityId)
		s.DestroyEntity(worldAvatar.weaponEntityId)
	}
}

func (s *Scene) CreateEntityAvatar(player *model.Player, avatarId uint32) uint32 {
	entityId := s.world.GetNextWorldEntityId(constant.ENTITY_TYPE_AVATAR)
	dbAvatar := player.GetDbAvatar()
	avatar := dbAvatar.GetAvatarById(avatarId)
	if avatar == nil {
		logger.Error("get avatar is nil, avatarId: %v", avatar)
		return 0
	}
	entity := &Entity{
		id:                  entityId,
		scene:               s,
		lifeState:           avatar.LifeState,
		pos:                 player.GetPos(),
		rot:                 player.GetRot(),
		moveState:           uint16(proto.MotionState_MOTION_NONE),
		lastMoveSceneTimeMs: 0,
		lastMoveReliableSeq: 0,
		fightProp:           avatar.FightPropMap, // 使用角色结构的数据
		entityType:          constant.ENTITY_TYPE_AVATAR,
		avatarEntity: &AvatarEntity{
			uid:      player.PlayerId,
			avatarId: avatarId,
		},
	}
	return s.CreateEntity(entity)
}

func (s *Scene) CreateEntityWeapon(pos, rot *model.Vector) uint32 {
	entityId := s.world.GetNextWorldEntityId(constant.ENTITY_TYPE_WEAPON)
	entity := &Entity{
		id:                  entityId,
		scene:               s,
		lifeState:           constant.LIFE_STATE_ALIVE,
		pos:                 &model.Vector{X: pos.X, Y: pos.Y, Z: pos.Z},
		rot:                 &model.Vector{X: rot.X, Y: rot.Y, Z: rot.Z},
		moveState:           uint16(proto.MotionState_MOTION_NONE),
		lastMoveSceneTimeMs: 0,
		lastMoveReliableSeq: 0,
		fightProp: map[uint32]float32{
			constant.FIGHT_PROP_CUR_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_MAX_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_BASE_HP: float32(1),
		},
		entityType: constant.ENTITY_TYPE_WEAPON,
	}
	return s.CreateEntity(entity)
}

func (s *Scene) CreateEntityMonster(pos, rot *model.Vector, monsterId uint32, level uint8, configId, groupId uint32) uint32 {
	fpm := map[uint32]float32{
		constant.FIGHT_PROP_BASE_ATTACK:       float32(50.0),
		constant.FIGHT_PROP_CUR_ATTACK:        float32(50.0),
		constant.FIGHT_PROP_BASE_DEFENSE:      float32(500.0),
		constant.FIGHT_PROP_CUR_DEFENSE:       float32(500.0),
		constant.FIGHT_PROP_BASE_HP:           float32(50.0),
		constant.FIGHT_PROP_CUR_HP:            float32(50.0),
		constant.FIGHT_PROP_MAX_HP:            float32(50.0),
		constant.FIGHT_PROP_PHYSICAL_SUB_HURT: float32(0.1),
		constant.FIGHT_PROP_ICE_SUB_HURT:      float32(0.1),
		constant.FIGHT_PROP_FIRE_SUB_HURT:     float32(0.1),
		constant.FIGHT_PROP_ELEC_SUB_HURT:     float32(0.1),
		constant.FIGHT_PROP_WIND_SUB_HURT:     float32(0.1),
		constant.FIGHT_PROP_ROCK_SUB_HURT:     float32(0.1),
		constant.FIGHT_PROP_GRASS_SUB_HURT:    float32(0.1),
		constant.FIGHT_PROP_WATER_SUB_HURT:    float32(0.1),
	}
	monsterDataConfig := gdconf.GetMonsterDataById(int32(monsterId))
	if monsterDataConfig == nil {
		logger.Error("get monster data config is nil, monsterId: %v", monsterId)
		return 0
	}
	fpm[constant.FIGHT_PROP_BASE_ATTACK] = monsterDataConfig.GetBaseAttackByLevel(level)
	fpm[constant.FIGHT_PROP_CUR_ATTACK] = monsterDataConfig.GetBaseAttackByLevel(level)
	fpm[constant.FIGHT_PROP_BASE_DEFENSE] = monsterDataConfig.GetBaseDefenseByLevel(level)
	fpm[constant.FIGHT_PROP_CUR_DEFENSE] = monsterDataConfig.GetBaseDefenseByLevel(level)
	fpm[constant.FIGHT_PROP_BASE_HP] = monsterDataConfig.GetBaseHpByLevel(level)
	fpm[constant.FIGHT_PROP_CUR_HP] = monsterDataConfig.GetBaseHpByLevel(level)
	fpm[constant.FIGHT_PROP_MAX_HP] = monsterDataConfig.GetBaseHpByLevel(level)
	entityId := s.world.GetNextWorldEntityId(constant.ENTITY_TYPE_MONSTER)
	entity := &Entity{
		id:                  entityId,
		scene:               s,
		lifeState:           constant.LIFE_STATE_ALIVE,
		pos:                 &model.Vector{X: pos.X, Y: pos.Y, Z: pos.Z},
		rot:                 &model.Vector{X: rot.X, Y: rot.Y, Z: rot.Z},
		moveState:           uint16(proto.MotionState_MOTION_NONE),
		lastMoveSceneTimeMs: 0,
		lastMoveReliableSeq: 0,
		fightProp:           fpm,
		entityType:          constant.ENTITY_TYPE_MONSTER,
		level:               level,
		monsterEntity: &MonsterEntity{
			monsterId: monsterId,
		},
		configId: configId,
		groupId:  groupId,
	}
	return s.CreateEntity(entity)
}

func (s *Scene) CreateEntityNpc(pos, rot *model.Vector, npcId, roomId, parentQuestId, blockId, configId, groupId uint32) uint32 {
	entityId := s.world.GetNextWorldEntityId(constant.ENTITY_TYPE_NPC)
	entity := &Entity{
		id:                  entityId,
		scene:               s,
		lifeState:           constant.LIFE_STATE_ALIVE,
		pos:                 &model.Vector{X: pos.X, Y: pos.Y, Z: pos.Z},
		rot:                 &model.Vector{X: rot.X, Y: rot.Y, Z: rot.Z},
		moveState:           uint16(proto.MotionState_MOTION_NONE),
		lastMoveSceneTimeMs: 0,
		lastMoveReliableSeq: 0,
		fightProp: map[uint32]float32{
			constant.FIGHT_PROP_CUR_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_MAX_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_BASE_HP: float32(1),
		},
		entityType: constant.ENTITY_TYPE_NPC,
		npcEntity: &NpcEntity{
			npcId:         npcId,
			roomId:        roomId,
			parentQuestId: parentQuestId,
			blockId:       blockId,
		},
		configId: configId,
		groupId:  groupId,
	}
	return s.CreateEntity(entity)
}

func (s *Scene) CreateEntityGadgetNormal(pos, rot *model.Vector, gadgetId, gadgetState uint32, gadgetNormalEntity *GadgetNormalEntity, configId, groupId uint32) uint32 {
	entityId := s.world.GetNextWorldEntityId(constant.ENTITY_TYPE_GADGET)
	entity := &Entity{
		id:                  entityId,
		scene:               s,
		lifeState:           constant.LIFE_STATE_ALIVE,
		pos:                 &model.Vector{X: pos.X, Y: pos.Y, Z: pos.Z},
		rot:                 &model.Vector{X: rot.X, Y: rot.Y, Z: rot.Z},
		moveState:           uint16(proto.MotionState_MOTION_NONE),
		lastMoveSceneTimeMs: 0,
		lastMoveReliableSeq: 0,
		fightProp: map[uint32]float32{
			constant.FIGHT_PROP_CUR_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_MAX_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_BASE_HP: float32(1),
		},
		entityType: constant.ENTITY_TYPE_GADGET,
		gadgetEntity: &GadgetEntity{
			gadgetId:           gadgetId,
			gadgetState:        gadgetState,
			gadgetType:         GADGET_TYPE_NORMAL,
			gadgetNormalEntity: gadgetNormalEntity,
		},
		configId: configId,
		groupId:  groupId,
	}
	return s.CreateEntity(entity)
}

func (s *Scene) CreateEntityGadgetClient(pos, rot *model.Vector, entityId, configId, campId, campType, ownerEntityId, targetEntityId, propOwnerEntityId uint32) bool {
	entity := &Entity{
		id:                  entityId,
		scene:               s,
		lifeState:           constant.LIFE_STATE_ALIVE,
		pos:                 &model.Vector{X: pos.X, Y: pos.Y, Z: pos.Z},
		rot:                 &model.Vector{X: rot.X, Y: rot.Y, Z: rot.Z},
		moveState:           uint16(proto.MotionState_MOTION_NONE),
		lastMoveSceneTimeMs: 0,
		lastMoveReliableSeq: 0,
		fightProp: map[uint32]float32{
			constant.FIGHT_PROP_CUR_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_MAX_HP:  math.MaxFloat32,
			constant.FIGHT_PROP_BASE_HP: float32(1),
		},
		entityType: constant.ENTITY_TYPE_GADGET,
		gadgetEntity: &GadgetEntity{
			gadgetType: GADGET_TYPE_CLIENT,
			gadgetClientEntity: &GadgetClientEntity{
				configId:          configId,
				campId:            campId,
				campType:          campType,
				ownerEntityId:     ownerEntityId,
				targetEntityId:    targetEntityId,
				propOwnerEntityId: propOwnerEntityId,
			},
		},
	}
	if s.CreateEntity(entity) == 0 {
		return false
	}
	return true
}

func (s *Scene) CreateEntityGadgetVehicle(ownerUid uint32, pos, rot *model.Vector, vehicleId uint32) uint32 {
	// 获取载具配置表
	vehicleDataConfig := gdconf.GetVehicleDataById(int32(vehicleId))
	if vehicleDataConfig == nil {
		logger.Error("vehicle config error, vehicleId: %v", vehicleId)
		return 0
	}
	entityId := s.world.GetNextWorldEntityId(constant.ENTITY_TYPE_GADGET)
	entity := &Entity{
		id:                  entityId,
		scene:               s,
		lifeState:           constant.LIFE_STATE_ALIVE,
		pos:                 &model.Vector{X: pos.X, Y: pos.Y, Z: pos.Z},
		rot:                 &model.Vector{X: rot.X, Y: rot.Y, Z: rot.Z},
		moveState:           uint16(proto.MotionState_MOTION_NONE),
		lastMoveSceneTimeMs: 0,
		lastMoveReliableSeq: 0,
		fightProp: map[uint32]float32{
			constant.FIGHT_PROP_BASE_DEFENSE: vehicleDataConfig.ConfigGadgetVehicle.Combat.Property.DefenseBase,
			constant.FIGHT_PROP_CUR_HP:       vehicleDataConfig.ConfigGadgetVehicle.Combat.Property.HP,
			constant.FIGHT_PROP_MAX_HP:       vehicleDataConfig.ConfigGadgetVehicle.Combat.Property.HP,
			constant.FIGHT_PROP_CUR_ATTACK:   vehicleDataConfig.ConfigGadgetVehicle.Combat.Property.Attack,
		},
		entityType: constant.ENTITY_TYPE_GADGET,
		gadgetEntity: &GadgetEntity{
			gadgetType: GADGET_TYPE_VEHICLE,
			gadgetVehicleEntity: &GadgetVehicleEntity{
				vehicleId:    vehicleId,
				worldId:      s.world.id,
				ownerUid:     ownerUid,
				maxStamina:   vehicleDataConfig.ConfigGadgetVehicle.Vehicle.Stamina.StaminaUpperLimit,
				curStamina:   vehicleDataConfig.ConfigGadgetVehicle.Vehicle.Stamina.StaminaUpperLimit,
				restoreDelay: 0,
				memberMap:    make(map[uint32]*model.Player),
			},
		},
	}
	return s.CreateEntity(entity)
}

func (s *Scene) CreateEntity(entity *Entity) uint32 {
	if len(s.entityMap) >= ENTITY_MAX_SEND_NUM && !ENTITY_NUM_UNLIMIT {
		logger.Error("above max scene entity num limit: %v, id: %v, pos: %v", ENTITY_MAX_SEND_NUM, entity.id, entity.pos)
		return 0
	}
	s.entityMap[entity.id] = entity
	return entity.id
}

func (s *Scene) DestroyEntity(entityId uint32) {
	entity := s.GetEntity(entityId)
	if entity == nil {
		return
	}
	delete(s.entityMap, entity.id)
}

func (s *Scene) GetEntity(entityId uint32) *Entity {
	return s.entityMap[entityId]
}

func (s *Scene) AddGroupSuite(groupId uint32, suiteId uint8, entityMap map[uint32]*Entity) {
	group, exist := s.groupMap[groupId]
	if !exist {
		group = &Group{
			id:       groupId,
			suiteMap: make(map[uint8]*Suite),
		}
		s.groupMap[groupId] = group
	}
	suite, exist := group.suiteMap[suiteId]
	if !exist {
		suite = &Suite{
			id:        suiteId,
			entityMap: make(map[uint32]*Entity),
		}
		group.suiteMap[suiteId] = suite
	}
	for k, v := range entityMap {
		suite.entityMap[k] = v
	}
}

func (s *Scene) RemoveGroupSuite(groupId uint32, suiteId uint8) {
	group := s.groupMap[groupId]
	if group == nil {
		logger.Error("group not exist, groupId: %v", groupId)
		return
	}
	suite := group.suiteMap[suiteId]
	if suite == nil {
		logger.Error("suite not exist, suiteId: %v", suiteId)
		return
	}
	for _, entity := range suite.entityMap {
		s.DestroyEntity(entity.id)
	}
	delete(group.suiteMap, suiteId)
	if len(group.suiteMap) == 0 {
		delete(s.groupMap, groupId)
	}
}

type Group struct {
	id       uint32
	suiteMap map[uint8]*Suite
}

type Suite struct {
	id        uint8
	entityMap map[uint32]*Entity
}

func (g *Group) GetId() uint32 {
	return g.id
}

func (g *Group) GetSuiteById(suiteId uint8) *Suite {
	return g.suiteMap[suiteId]
}

func (g *Group) GetAllSuite() map[uint8]*Suite {
	return g.suiteMap
}

func (g *Group) GetAllEntity() map[uint32]*Entity {
	entityMap := make(map[uint32]*Entity)
	for _, suite := range g.suiteMap {
		for _, entity := range suite.entityMap {
			entityMap[entity.id] = entity
		}
	}
	return entityMap
}

func (g *Group) GetEntityByConfigId(configId uint32) *Entity {
	for _, suite := range g.suiteMap {
		for _, entity := range suite.entityMap {
			if entity.configId == configId {
				return entity
			}
		}
	}
	return nil
}

func (g *Group) DestroyEntity(entityId uint32) {
	for _, suite := range g.suiteMap {
		for _, entity := range suite.entityMap {
			if entity.id == entityId {
				delete(suite.entityMap, entity.id)
				return
			}
		}
	}
}

func (s *Suite) GetId() uint8 {
	return s.id
}

func (s *Suite) GetEntityById(entityId uint32) *Entity {
	return s.entityMap[entityId]
}

func (s *Suite) GetAllEntity() map[uint32]*Entity {
	return s.entityMap
}

// Entity 场景实体数据结构
type Entity struct {
	id                  uint32 // 实体id
	scene               *Scene // 实体归属上级场景的访问指针
	lifeState           uint16 // 存活状态
	lastDieType         int32
	pos                 *model.Vector // 位置
	rot                 *model.Vector // 朝向
	moveState           uint16        // 运动状态
	lastMoveSceneTimeMs uint32
	lastMoveReliableSeq uint32
	fightProp           map[uint32]float32 // 战斗属性
	level               uint8              // 等级
	entityType          uint8              // 实体类型
	// TODO 这堆东西变得有点答辩了 我觉得之后需要重构 做面向对象继承多态接口的设计
	avatarEntity  *AvatarEntity
	monsterEntity *MonsterEntity
	npcEntity     *NpcEntity
	gadgetEntity  *GadgetEntity
	configId      uint32 // LUA配置相关
	groupId       uint32
}

func (e *Entity) GetId() uint32 {
	return e.id
}

func (e *Entity) GetScene() *Scene {
	return e.scene
}

func (e *Entity) GetLifeState() uint16 {
	return e.lifeState
}

func (e *Entity) SetLifeState(lifeState uint16) {
	e.lifeState = lifeState
}

func (e *Entity) GetLastDieType() int32 {
	return e.lastDieType
}

func (e *Entity) SetLastDieType(lastDieType int32) {
	e.lastDieType = lastDieType
}

func (e *Entity) GetPos() *model.Vector {
	return &model.Vector{X: e.pos.X, Y: e.pos.Y, Z: e.pos.Z}
}

func (e *Entity) SetPos(pos *model.Vector) {
	e.pos.X, e.pos.Y, e.pos.Z = pos.X, pos.Y, pos.Z
}

func (e *Entity) GetRot() *model.Vector {
	return &model.Vector{X: e.rot.X, Y: e.rot.Y, Z: e.rot.Z}
}

func (e *Entity) SetRot(rot *model.Vector) {
	e.rot.X, e.rot.Y, e.rot.Z = rot.X, rot.Y, rot.Z
}

func (e *Entity) GetMoveState() uint16 {
	return e.moveState
}

func (e *Entity) SetMoveState(moveState uint16) {
	e.moveState = moveState
}

func (e *Entity) GetLastMoveSceneTimeMs() uint32 {
	return e.lastMoveSceneTimeMs
}

func (e *Entity) SetLastMoveSceneTimeMs(lastMoveSceneTimeMs uint32) {
	e.lastMoveSceneTimeMs = lastMoveSceneTimeMs
}

func (e *Entity) GetLastMoveReliableSeq() uint32 {
	return e.lastMoveReliableSeq
}

func (e *Entity) SetLastMoveReliableSeq(lastMoveReliableSeq uint32) {
	e.lastMoveReliableSeq = lastMoveReliableSeq
}

func (e *Entity) GetFightProp() map[uint32]float32 {
	return e.fightProp
}

func (e *Entity) GetLevel() uint8 {
	return e.level
}

func (e *Entity) GetEntityType() uint8 {
	return e.entityType
}

func (e *Entity) GetAvatarEntity() *AvatarEntity {
	return e.avatarEntity
}

func (e *Entity) GetMonsterEntity() *MonsterEntity {
	return e.monsterEntity
}

func (e *Entity) GetNpcEntity() *NpcEntity {
	return e.npcEntity
}

func (e *Entity) GetGadgetEntity() *GadgetEntity {
	return e.gadgetEntity
}

func (e *Entity) GetConfigId() uint32 {
	return e.configId
}

func (e *Entity) GetGroupId() uint32 {
	return e.groupId
}

type AvatarEntity struct {
	uid      uint32
	avatarId uint32
}

func (a *AvatarEntity) GetUid() uint32 {
	return a.uid
}

func (a *AvatarEntity) GetAvatarId() uint32 {
	return a.avatarId
}

type MonsterEntity struct {
	monsterId uint32
}

func (m *MonsterEntity) GetMonsterId() uint32 {
	return m.monsterId
}

type NpcEntity struct {
	npcId         uint32
	roomId        uint32
	parentQuestId uint32
	blockId       uint32
}

func (n *NpcEntity) GetNpcId() uint32 {
	return n.npcId
}

func (n *NpcEntity) GetRoomId() uint32 {
	return n.roomId
}

func (n *NpcEntity) GetParentQuestId() uint32 {
	return n.parentQuestId
}

func (n *NpcEntity) GetBlockId() uint32 {
	return n.blockId
}

const (
	GADGET_TYPE_NORMAL = iota
	GADGET_TYPE_CLIENT
	GADGET_TYPE_VEHICLE // 载具
)

type GadgetEntity struct {
	gadgetType          int
	gadgetId            uint32
	gadgetState         uint32
	gadgetNormalEntity  *GadgetNormalEntity
	gadgetClientEntity  *GadgetClientEntity
	gadgetVehicleEntity *GadgetVehicleEntity
}

func (g *GadgetEntity) GetGadgetType() int {
	return g.gadgetType
}

func (g *GadgetEntity) GetGadgetId() uint32 {
	return g.gadgetId
}

func (g *GadgetEntity) GetGadgetState() uint32 {
	return g.gadgetState
}

func (g *GadgetEntity) SetGadgetState(state uint32) {
	g.gadgetState = state
}

func (g *GadgetEntity) GetGadgetNormalEntity() *GadgetNormalEntity {
	return g.gadgetNormalEntity
}

func (g *GadgetEntity) GetGadgetClientEntity() *GadgetClientEntity {
	return g.gadgetClientEntity
}

func (g *GadgetEntity) GetGadgetVehicleEntity() *GadgetVehicleEntity {
	return g.gadgetVehicleEntity
}

type GadgetNormalEntity struct {
	isDrop bool
	itemId uint32
	count  uint32
}

func (g *GadgetNormalEntity) GetIsDrop() bool {
	return g.isDrop
}

func (g *GadgetNormalEntity) GetItemId() uint32 {
	return g.itemId
}

func (g *GadgetNormalEntity) GetCount() uint32 {
	return g.count
}

type GadgetClientEntity struct {
	configId          uint32
	campId            uint32
	campType          uint32
	ownerEntityId     uint32
	targetEntityId    uint32
	propOwnerEntityId uint32
}

func (g *GadgetClientEntity) GetConfigId() uint32 {
	return g.configId
}

func (g *GadgetClientEntity) GetCampId() uint32 {
	return g.campId
}

func (g *GadgetClientEntity) GetCampType() uint32 {
	return g.campType
}

func (g *GadgetClientEntity) GetOwnerEntityId() uint32 {
	return g.ownerEntityId
}

func (g *GadgetClientEntity) GetTargetEntityId() uint32 {
	return g.targetEntityId
}

func (g *GadgetClientEntity) GetPropOwnerEntityId() uint32 {
	return g.propOwnerEntityId
}

type GadgetVehicleEntity struct {
	vehicleId    uint32
	worldId      uint64
	ownerUid     uint32
	maxStamina   float32
	curStamina   float32
	restoreDelay uint8                    // 载具耐力回复延时
	memberMap    map[uint32]*model.Player // uint32 = pos
}

func (g *GadgetVehicleEntity) GetVehicleId() uint32 {
	return g.vehicleId
}

func (g *GadgetVehicleEntity) GetWorldId() uint64 {
	return g.worldId
}

func (g *GadgetVehicleEntity) GetOwnerUid() uint32 {
	return g.ownerUid
}

func (g *GadgetVehicleEntity) GetMaxStamina() float32 {
	return g.maxStamina
}

func (g *GadgetVehicleEntity) GetCurStamina() float32 {
	return g.curStamina
}

func (g *GadgetVehicleEntity) SetCurStamina(curStamina float32) {
	g.curStamina = curStamina
}

func (g *GadgetVehicleEntity) GetMemberMap() map[uint32]*model.Player {
	return g.memberMap
}

func (g *GadgetVehicleEntity) GetRestoreDelay() uint8 {
	return g.restoreDelay
}

func (g *GadgetVehicleEntity) SetRestoreDelay(restoreDelay uint8) {
	g.restoreDelay = restoreDelay
}
