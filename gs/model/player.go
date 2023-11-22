package model

import (
	"hk4e/pkg/logger"
	"hk4e/protocol/proto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	DbNone = iota
	DbInsert
	DbDelete
	DbNormal
)

const (
	SceneNone = iota
	SceneInitFinish
	SceneEnterDone
)

type GameObject interface {
}

type Player struct {
	// 离线数据 请尽量不要定义接口等复杂数据结构
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	PlayerId        uint32             `bson:"player_id"` // 玩家uid
	NickName        string             // 昵称
	HeadImage       uint32             // 头像
	Signature       string             // 签名
	IsBorn          bool               // 是否完成开场动画
	OnlineTime      uint32             // 上线时间点
	OfflineTime     uint32             // 离线时间点
	TotalOnlineTime uint32             // 累计在线时长
	PropMap         map[uint32]uint32  // 玩家属性表
	OpenStateMap    map[uint32]uint32  // 功能开放状态
	SceneId         uint32             // 存档场景
	Pos             *Vector            // 存档坐标 非实时
	Rot             *Vector            // 存档朝向 非实时
	CmdPerm         uint8              // 玩家命令权限等级
	DbSocial        *DbSocial          // 社交
	DbItem          *DbItem            // 道具
	DbAvatar        *DbAvatar          // 角色
	DbTeam          *DbTeam            // 队伍
	DbWeapon        *DbWeapon          // 武器
	DbReliquary     *DbReliquary       // 圣遗物
	DbGacha         *DbGacha           // 卡池
	DbQuest         *DbQuest           // 任务
	DbWorld         *DbWorld           // 大世界
	// 在线数据 请随意 记得加忽略字段的tag
	LastSaveTime          uint32                                   `bson:"-" msgpack:"-"` // 上一次存档保存时间
	DbState               int                                      `bson:"-" msgpack:"-"` // 数据库存档状态
	WorldId               uint64                                   `bson:"-" msgpack:"-"` // 所在的世界id
	GameObjectGuidCounter uint64                                   `bson:"-" msgpack:"-"` // 游戏对象guid计数器
	LastKeepaliveTime     uint32                                   `bson:"-" msgpack:"-"` // 上一次保持活跃时间
	ClientTime            uint32                                   `bson:"-" msgpack:"-"` // 客户端本地时钟
	ClientRTT             uint32                                   `bson:"-" msgpack:"-"` // 客户端网络往返时延
	GameObjectGuidMap     map[uint64]GameObject                    `bson:"-" msgpack:"-"` // 游戏对象guid映射表
	Online                bool                                     `bson:"-" msgpack:"-"` // 在线状态
	Pause                 bool                                     `bson:"-" msgpack:"-"` // 暂停状态
	SceneJump             bool                                     `bson:"-" msgpack:"-"` // 是否场景切换
	SceneLoadState        int                                      `bson:"-" msgpack:"-"` // 场景加载状态
	SceneEnterReason      uint32                                   `bson:"-" msgpack:"-"` // 场景进入原因
	CoopApplyMap          map[uint32]int64                         `bson:"-" msgpack:"-"` // 敲门申请的玩家uid及时间
	StaminaInfo           *StaminaInfo                             `bson:"-" msgpack:"-"` // 耐力在线数据
	VehicleInfo           *VehicleInfo                             `bson:"-" msgpack:"-"` // 载具在线数据
	ClientSeq             uint32                                   `bson:"-" msgpack:"-"` // 客户端发包请求的序号
	CombatInvokeHandler   *InvokeHandler[proto.CombatInvokeEntry]  `bson:"-" msgpack:"-"` // combat转发器
	AbilityInvokeHandler  *InvokeHandler[proto.AbilityInvokeEntry] `bson:"-" msgpack:"-"` // ability转发器
	GateAppId             string                                   `bson:"-" msgpack:"-"` // 网关服务器的appid
	MultiServerAppId      string                                   `bson:"-" msgpack:"-"` // 多功能服务器的appid
	GCGCurGameGuid        uint32                                   `bson:"-" msgpack:"-"` // GCG玩家所在的游戏guid
	GCGInfo               *GCGInfo                                 `bson:"-" msgpack:"-"` // 七圣召唤信息
	XLuaDebug             bool                                     `bson:"-" msgpack:"-"` // 是否开启客户端XLUA调试
	NetFreeze             bool                                     `bson:"-" msgpack:"-"` // 客户端网络上下行冻结状态
	CommandAssignUid      uint32                                   `bson:"-" msgpack:"-"` // 命令指定uid
	WeatherInfo           *WeatherInfo                             `bson:"-" msgpack:"-"` // 天气信息
	ClientVersion         int                                      `bson:"-" msgpack:"-"` // 玩家在线的客户端版本
	OfflineClear          bool                                     `bson:"-" msgpack:"-"` // 是否离线时清除账号数据
	NotSave               bool                                     `bson:"-" msgpack:"-"` // 是否离线回档
	Speed                 *Vector                                  `bson:"-" msgpack:"-"` // 速度
	WuDi                  bool                                     `bson:"-" msgpack:"-"` // 是否开启玩家角色无敌
	EnergyInf             bool                                     `bson:"-" msgpack:"-"` // 是否开启玩家角色无限能量
	StaminaInf            bool                                     `bson:"-" msgpack:"-"` // 是否开启玩家无限耐力
	IsInMp                bool                                     `bson:"-" msgpack:"-"` // 是否位于多人世界
	MpSceneId             uint32                                   `bson:"-" msgpack:"-"` // 多人世界场景
	MpPos                 *Vector                                  `bson:"-" msgpack:"-"` // 多人世界坐标
	MpRot                 *Vector                                  `bson:"-" msgpack:"-"` // 多人世界朝向
	// 特殊数据
	ChatMsgMap           map[uint32][]*ChatMsg `bson:"-" msgpack:"-"` // 聊天信息 数据量偏大 只从db读写 不保存到redis
	RemoteWorldPlayerNum uint32                `bson:"-"`             // 远程展示世界内人数 在线同步到redis 不保存到db
}

// 存档场景

func (p *Player) GetSceneId() uint32 {
	if p.IsInMp {
		return p.MpSceneId
	} else {
		return p.SceneId
	}
}

func (p *Player) SetSceneId(sceneId uint32) {
	if p.IsInMp {
		p.MpSceneId = sceneId
	} else {
		p.SceneId = sceneId
	}
}

// 存档坐标

func (p *Player) GetPos() *Vector {
	if p.IsInMp {
		return &Vector{X: p.MpPos.X, Y: p.MpPos.Y, Z: p.MpPos.Z}
	} else {
		return &Vector{X: p.Pos.X, Y: p.Pos.Y, Z: p.Pos.Z}
	}
}

func (p *Player) SetPos(pos *Vector) {
	if p.IsInMp {
		p.MpPos.X = pos.X
		p.MpPos.Y = pos.Y
		p.MpPos.Z = pos.Z
	} else {
		p.Pos.X = pos.X
		p.Pos.Y = pos.Y
		p.Pos.Z = pos.Z
	}
}

// 存档朝向

func (p *Player) GetRot() *Vector {
	if p.IsInMp {
		return &Vector{X: p.MpRot.X, Y: p.MpRot.Y, Z: p.MpRot.Z}
	} else {
		return &Vector{X: p.Rot.X, Y: p.Rot.Y, Z: p.Rot.Z}
	}
}

func (p *Player) SetRot(rot *Vector) {
	if p.IsInMp {
		p.MpRot.X = rot.X
		p.MpRot.Y = rot.Y
		p.MpRot.Z = rot.Z
	} else {
		p.Rot.X = rot.X
		p.Rot.Y = rot.Y
		p.Rot.Z = rot.Z
	}
}

func (p *Player) GetNextGameObjectGuid() uint64 {
	p.GameObjectGuidCounter++
	return uint64(p.PlayerId)<<32 + p.GameObjectGuidCounter
}

func (p *Player) InitOnlineData() {
	// 在线数据初始化
	p.GameObjectGuidMap = make(map[uint64]GameObject)
	p.CoopApplyMap = make(map[uint32]int64)
	p.StaminaInfo = NewStaminaInfo()
	p.VehicleInfo = NewVehicleInfo()
	p.CombatInvokeHandler = NewInvokeHandler[proto.CombatInvokeEntry]()
	p.AbilityInvokeHandler = NewInvokeHandler[proto.AbilityInvokeEntry]()
	p.GCGInfo = NewGCGInfo() // 临时测试用数据
	p.WeatherInfo = NewWeatherInfo()

	dbAvatar := p.GetDbAvatar()
	dbAvatar.InitDbAvatar(p)
	dbReliquary := p.GetDbReliquary()
	dbReliquary.InitDbReliquary(p)
	dbWeapon := p.GetDbWeapon()
	dbWeapon.InitDbWeapon(p)
	dbItem := p.GetDbItem()
	dbItem.InitDbItem(p)
}

type Vector struct {
	X float64
	Y float64
	Z float64
}

// 多人世界网络同步包转发器

type InvokeEntryType interface {
	proto.CombatInvokeEntry | proto.AbilityInvokeEntry
}

type InvokeHandler[T InvokeEntryType] struct {
	EntryListForwardAll          []*T
	EntryListForwardAllExceptCur []*T
	EntryListForwardHost         []*T
	EntryListForwardServer       []*T
}

func NewInvokeHandler[T InvokeEntryType]() (r *InvokeHandler[T]) {
	r = new(InvokeHandler[T])
	r.InitInvokeHandler()
	return r
}

func (i *InvokeHandler[T]) InitInvokeHandler() {
	i.EntryListForwardAll = make([]*T, 0)
	i.EntryListForwardAllExceptCur = make([]*T, 0)
	i.EntryListForwardHost = make([]*T, 0)
	i.EntryListForwardServer = make([]*T, 0)
}

func (i *InvokeHandler[T]) AddEntry(forward proto.ForwardType, entry *T) {
	switch forward {
	case proto.ForwardType_FORWARD_TO_ALL:
		i.EntryListForwardAll = append(i.EntryListForwardAll, entry)
	case proto.ForwardType_FORWARD_TO_ALL_EXCEPT_CUR:
		fallthrough
	case proto.ForwardType_FORWARD_TO_ALL_EXIST_EXCEPT_CUR:
		i.EntryListForwardAllExceptCur = append(i.EntryListForwardAllExceptCur, entry)
	case proto.ForwardType_FORWARD_TO_HOST:
		i.EntryListForwardHost = append(i.EntryListForwardHost, entry)
	case proto.ForwardType_FORWARD_TO_PEER:
		i.EntryListForwardAllExceptCur = append(i.EntryListForwardAllExceptCur, entry)
	case proto.ForwardType_FORWARD_ONLY_SERVER:
		i.EntryListForwardServer = append(i.EntryListForwardServer, entry)
	default:
		logger.Error("forward type: %v, entry: %v", forward, entry)
	}
}

func (i *InvokeHandler[T]) AllLen() int {
	return len(i.EntryListForwardAll)
}

func (i *InvokeHandler[T]) AllExceptCurLen() int {
	return len(i.EntryListForwardAllExceptCur)
}

func (i *InvokeHandler[T]) HostLen() int {
	return len(i.EntryListForwardHost)
}

func (i *InvokeHandler[T]) ServerLen() int {
	return len(i.EntryListForwardServer)
}

func (i *InvokeHandler[T]) Clear() {
	i.EntryListForwardAll = make([]*T, 0)
	i.EntryListForwardAllExceptCur = make([]*T, 0)
	i.EntryListForwardHost = make([]*T, 0)
	i.EntryListForwardServer = make([]*T, 0)
}
