package handle

import (
	"math"
	"time"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/pkg/logger"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

const (
	MoveVectorCacheNum        = 10
	MaxMoveSpeed              = 100.0
	JumpDistance              = 500.0
	PointDistance             = 10.0
	AttackCountLimitEntitySec = 10
	KickCheatPlayer           = false
)

type MoveVector struct {
	pos  *proto.Vector
	time int64
}

type AttackEntity struct {
	attackStartTime uint64
	attackCount     uint32
}

type AnticheatContext struct {
	sceneId         uint32
	moveVectorList  []*MoveVector
	attackEntityMap map[uint32]*AttackEntity
}

func (a *AnticheatContext) Move(pos *proto.Vector) bool {
	now := time.Now().UnixMilli()
	if len(a.moveVectorList) > 0 {
		lastMoveVector := a.moveVectorList[len(a.moveVectorList)-1]
		if now-lastMoveVector.time < 1000 {
			return true
		}
		distance := GetDistance(pos, lastMoveVector.pos)
		if distance > JumpDistance {
			// 瞬时变化太大 判断是否为传送
			scenePointMap := gdconf.GetScenePointMapBySceneId(int32(a.sceneId))
			if scenePointMap == nil {
				return true
			}
			isJump := true
			for _, pointData := range scenePointMap {
				if pointData.TranPos == nil {
					continue
				}
				d := GetDistance(pos, &proto.Vector{
					X: float32(pointData.TranPos.X),
					Y: float32(pointData.TranPos.Y),
					Z: float32(pointData.TranPos.Z),
				})
				if d < PointDistance {
					isJump = false
					break
				}
			}
			if isJump {
				return false
			} else {
				a.moveVectorList = make([]*MoveVector, 0)
			}
		}
	}
	a.moveVectorList = append(a.moveVectorList, &MoveVector{
		pos:  pos,
		time: now,
	})
	if len(a.moveVectorList) > MoveVectorCacheNum {
		a.moveVectorList = a.moveVectorList[len(a.moveVectorList)-MoveVectorCacheNum:]
	}
	return true
}

func (a *AnticheatContext) GetMoveSpeed() float32 {
	avgMoveSpeed := float32(0.0)
	if len(a.moveVectorList) < MoveVectorCacheNum {
		return avgMoveSpeed
	}
	for index := range a.moveVectorList {
		if index+1 >= len(a.moveVectorList) {
			break
		}
		nextMoveVector := a.moveVectorList[index+1]
		beforeMoveVector := a.moveVectorList[index]
		dx := GetDistance(nextMoveVector.pos, beforeMoveVector.pos)
		dt := float32(nextMoveVector.time-beforeMoveVector.time) / 1000.0
		avgMoveSpeed += dx / dt
	}
	avgMoveSpeed /= float32(len(a.moveVectorList))
	return avgMoveSpeed
}

func (a *AnticheatContext) Attack(defEntityId uint32) bool {
	now := uint64(time.Now().UnixMilli())
	attackEntity, exist := a.attackEntityMap[defEntityId]
	if !exist {
		attackEntity = &AttackEntity{
			attackStartTime: now,
			attackCount:     0,
		}
		a.attackEntityMap[defEntityId] = attackEntity
	}
	attackEntity.attackCount++
	if attackEntity.attackCount > AttackCountLimitEntitySec {
		if now-attackEntity.attackStartTime < 1000 {
			return false
		} else {
			attackEntity.attackStartTime = now
			attackEntity.attackCount = 0
		}
	}
	return true
}

func NewAnticheatContext() *AnticheatContext {
	r := &AnticheatContext{
		sceneId:         0,
		moveVectorList:  make([]*MoveVector, 0),
		attackEntityMap: make(map[uint32]*AttackEntity),
	}
	return r
}

func (h *Handle) AddPlayerAcCtx(userId uint32) {
	h.playerAcCtxMap[userId] = NewAnticheatContext()
}

func (h *Handle) DelPlayerAcCtx(userId uint32) {
	delete(h.playerAcCtxMap, userId)
}

func (h *Handle) GetPlayerAcCtx(userId uint32) *AnticheatContext {
	return h.playerAcCtxMap[userId]
}

func (h *Handle) CombatInvocationsNotify(userId uint32, gateAppId string, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.CombatInvocationsNotify)
	ctx := h.GetPlayerAcCtx(userId)
	if ctx == nil {
		logger.Error("get player anticheat context is nil, uid: %v", userId)
		return
	}
	for _, entry := range req.InvokeList {
		switch entry.ArgumentType {
		case proto.CombatTypeArgument_ENTITY_MOVE:
			entityMoveInfo := new(proto.EntityMoveInfo)
			err := pb.Unmarshal(entry.CombatData, entityMoveInfo)
			if err != nil {
				logger.Error("parse EntityMoveInfo error: %v, uid: %v", err, userId)
				continue
			}
			if GetEntityType(entityMoveInfo.EntityId) != constant.ENTITY_TYPE_AVATAR {
				continue
			}
			if entityMoveInfo.MotionInfo == nil {
				continue
			}
			motionInfo := entityMoveInfo.MotionInfo
			if motionInfo.Pos == nil {
				continue
			}
			// 玩家超速移动检测
			if ctx.sceneId != 3 {
				continue
			}
			ok := ctx.Move(motionInfo.Pos)
			if !ok {
				logger.Warn("player move jump, pos: %v, uid: %v", motionInfo.Pos, userId)
				h.KickPlayer(userId, gateAppId)
				continue
			}
			moveSpeed := ctx.GetMoveSpeed()
			if moveSpeed > MaxMoveSpeed {
				logger.Warn("player move overspeed, speed: %v, uid: %v", moveSpeed, userId)
				h.KickPlayer(userId, gateAppId)
				continue
			}
		case proto.CombatTypeArgument_COMBAT_EVT_BEING_HIT:
			evtBeingHitInfo := new(proto.EvtBeingHitInfo)
			err := pb.Unmarshal(entry.CombatData, evtBeingHitInfo)
			if err != nil {
				logger.Error("parse EvtBeingHitInfo error: %v, uid: %v", err, userId)
				continue
			}
			attackResult := evtBeingHitInfo.AttackResult
			if attackResult == nil {
				continue
			}
			if GetEntityType(attackResult.DefenseId) != constant.ENTITY_TYPE_MONSTER {
				continue
			}
			ok := ctx.Attack(attackResult.DefenseId)
			if !ok {
				logger.Warn("player attack monster feq too high, uid: %v", userId)
				h.KickPlayer(userId, gateAppId)
				continue
			}
		}
	}
}

func (h *Handle) ToTheMoonEnterSceneReq(userId uint32, gateAppId string, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.ToTheMoonEnterSceneReq)
	ctx := h.GetPlayerAcCtx(userId)
	if ctx == nil {
		logger.Error("get player anticheat context is nil, uid: %v", userId)
		return
	}
	ctx.sceneId = req.SceneId
	logger.Info("player enter scene: %v, uid: %v", req.SceneId, userId)
}

func GetEntityType(entityId uint32) int {
	return int(entityId >> 24)
}

func GetDistance(v1 *proto.Vector, v2 *proto.Vector) float32 {
	return float32(math.Sqrt(
		float64((v1.X-v2.X)*(v1.X-v2.X)) +
			float64((v1.Y-v2.Y)*(v1.Y-v2.Y)) +
			float64((v1.Z-v2.Z)*(v1.Z-v2.Z)),
	))
}
