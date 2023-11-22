package game

import (
	"math"

	"hk4e/gs/model"
	"hk4e/pkg/alg"
	"hk4e/pkg/logger"
)

const (
	AVATAR_RADIUS      = 0.5
	AVATAR_HEIGHT      = 2.0
	AVATAR_Y_OFFSET    = 1.0
	ACC                = -5.0
	DRAG               = 0.01
	PITCH_ANGLE_OFFSET = 3.0
	INIT_SPEED         = 50.0
)

// RigidBody 刚体
type RigidBody struct {
	entityId          uint32       // 子弹实体id
	avatarEntityId    uint32       // 子弹发射者角色实体id
	hitAvatarEntityId uint32       // 子弹命中的角色实体id
	sceneId           uint32       // 子弹所在场景id
	position          *alg.Vector3 // 坐标
	velocity          *alg.Vector3 // 速度
}

// PhysicsEngine 物理引擎
type PhysicsEngine struct {
	rigidBodyMap     map[uint32]*RigidBody // 刚体集合
	pathTracing      bool                  // 子弹路径追踪调试
	acc              float32               // 重力加速度
	drag             float32               // 阻力参数
	pitchAngleOffset float32               // 子弹俯仰角偏移
	initSpeed        float32               // 子弹初始速度
	avatarYOffset    float32               // 角色中心点位置高度偏移
	lastUpdateTime   int64                 // 上一次更新时间
	world            *World                // 世界对象
}

func (w *World) NewPhysicsEngine() {
	w.bulletPhysicsEngine = &PhysicsEngine{
		rigidBodyMap:     make(map[uint32]*RigidBody),
		pathTracing:      false,
		acc:              ACC,
		drag:             DRAG,
		pitchAngleOffset: PITCH_ANGLE_OFFSET,
		initSpeed:        INIT_SPEED,
		avatarYOffset:    AVATAR_Y_OFFSET,
		lastUpdateTime:   0,
		world:            w,
	}
}

func (p *PhysicsEngine) SetPhysicsEngineParam(pathTracing bool) {
	p.pathTracing = pathTracing
}

func (p *PhysicsEngine) ShowAvatarCollider() {
	for _, scene := range p.world.GetAllScene() {
		for _, player := range scene.GetAllPlayer() {
			entity := p.world.GetPlayerActiveAvatarEntity(player)
			avatarPos := entity.GetPos()
			avatarPos.Y += float64(p.avatarYOffset)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{X: avatarPos.X, Y: avatarPos.Y, Z: avatarPos.Z}, GADGET_GREEN, nil)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{X: avatarPos.X, Y: avatarPos.Y + AVATAR_HEIGHT/2.0, Z: avatarPos.Z}, GADGET_GREEN, nil)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{X: avatarPos.X, Y: avatarPos.Y - AVATAR_HEIGHT/2.0, Z: avatarPos.Z}, GADGET_GREEN, nil)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{X: avatarPos.X + AVATAR_RADIUS, Y: avatarPos.Y, Z: avatarPos.Z}, GADGET_GREEN, nil)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{X: avatarPos.X - AVATAR_RADIUS, Y: avatarPos.Y, Z: avatarPos.Z}, GADGET_GREEN, nil)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{X: avatarPos.X, Y: avatarPos.Y, Z: avatarPos.Z + AVATAR_RADIUS}, GADGET_GREEN, nil)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{X: avatarPos.X, Y: avatarPos.Y, Z: avatarPos.Z - AVATAR_RADIUS}, GADGET_GREEN, nil)
		}
	}
}

func (p *PhysicsEngine) Update(now int64) []*RigidBody {
	hitList := make([]*RigidBody, 0)
	dt := float32(now-p.lastUpdateTime) / 1000.0
	for _, rigidBody := range p.rigidBodyMap {
		if !p.world.IsValidAiWorldPos(rigidBody.sceneId, rigidBody.position.X, rigidBody.position.Y, rigidBody.position.Z) {
			p.DestroyRigidBody(rigidBody.entityId)
			continue
		}
		// 阻力作用于速度
		dvx := p.drag * rigidBody.velocity.X * dt
		if math.Abs(float64(dvx)) >= math.Abs(float64(rigidBody.velocity.X)) {
			rigidBody.velocity.X = 0.0
		} else {
			rigidBody.velocity.X -= dvx
		}
		dvy := p.drag * rigidBody.velocity.Y * dt
		if math.Abs(float64(dvy)) >= math.Abs(float64(rigidBody.velocity.Y)) {
			rigidBody.velocity.Y = 0.0
		} else {
			rigidBody.velocity.Y -= dvy
		}
		dvz := p.drag * rigidBody.velocity.Z * dt
		if math.Abs(float64(dvz)) >= math.Abs(float64(rigidBody.velocity.Z)) {
			rigidBody.velocity.Z = 0.0
		} else {
			rigidBody.velocity.Z -= dvz
		}
		// 重力作用于速度
		rigidBody.velocity.Y += p.acc * dt
		// 速度作用于位移
		oldPos := &alg.Vector3{X: rigidBody.position.X, Y: rigidBody.position.Y, Z: rigidBody.position.Z}
		rigidBody.position.X += rigidBody.velocity.X * dt
		rigidBody.position.Y += rigidBody.velocity.Y * dt
		rigidBody.position.Z += rigidBody.velocity.Z * dt
		newPos := &alg.Vector3{X: rigidBody.position.X, Y: rigidBody.position.Y, Z: rigidBody.position.Z}
		// 碰撞检测
		hitAvatarEntityId := p.Collision(rigidBody.sceneId, rigidBody.avatarEntityId, oldPos, newPos)
		if hitAvatarEntityId != 0 {
			rigidBody.hitAvatarEntityId = hitAvatarEntityId
			hitList = append(hitList, rigidBody)
			p.DestroyRigidBody(rigidBody.entityId)
		}
		if p.pathTracing {
			logger.Debug("[PhysicsEngineUpdate] e: %v, s: %v, p: %v, v: %v", rigidBody.entityId, rigidBody.sceneId, rigidBody.position, rigidBody.velocity)
			GAME.CreateGadget(p.world.GetOwner(), &model.Vector{
				X: float64(rigidBody.position.X),
				Y: float64(rigidBody.position.Y),
				Z: float64(rigidBody.position.Z),
			}, GADGET_RED, nil)
		}
	}
	p.lastUpdateTime = now
	return hitList
}

func (p *PhysicsEngine) Collision(sceneId uint32, avatarEntityId uint32, oldPos *alg.Vector3, newPos *alg.Vector3) uint32 {
	scene := p.world.GetSceneById(sceneId)
	world := scene.GetWorld()
	for _, player := range scene.GetAllPlayer() {
		entity := world.GetPlayerActiveAvatarEntity(player)
		if entity.GetId() == avatarEntityId {
			continue
		}
		avatarPos := entity.GetPos()
		avatarPos.Y += float64(p.avatarYOffset)
		// x轴
		lineMinX := float32(0)
		lineMaxX := float32(0)
		if oldPos.X < newPos.X {
			lineMinX = oldPos.X
			lineMaxX = newPos.X
		} else {
			lineMinX = newPos.X
			lineMaxX = oldPos.X
		}
		shapeMinX := float32(avatarPos.X) - AVATAR_RADIUS
		shapeMaxX := float32(avatarPos.X) + AVATAR_RADIUS
		if lineMaxX < shapeMinX || lineMinX > shapeMaxX {
			continue
		}
		// z轴
		lineMinZ := float32(0)
		lineMaxZ := float32(0)
		if oldPos.Z < newPos.Z {
			lineMinZ = oldPos.Z
			lineMaxZ = newPos.Z
		} else {
			lineMinZ = newPos.Z
			lineMaxZ = oldPos.Z
		}
		shapeMinZ := float32(avatarPos.Z) - AVATAR_RADIUS
		shapeMaxZ := float32(avatarPos.Z) + AVATAR_RADIUS
		if lineMaxZ < shapeMinZ || lineMinZ > shapeMaxZ {
			continue
		}
		// y轴
		lineMinY := float32(0)
		lineMaxY := float32(0)
		if oldPos.Y < newPos.Y {
			lineMinY = oldPos.Y
			lineMaxY = newPos.Y
		} else {
			lineMinY = newPos.Y
			lineMaxY = oldPos.Y
		}
		shapeMinY := float32(avatarPos.Y) - AVATAR_HEIGHT/2.0
		shapeMaxY := float32(avatarPos.Y) + AVATAR_HEIGHT/2.0
		if lineMaxY < shapeMinY || lineMinY > shapeMaxY {
			continue
		}
		return entity.GetId()
	}
	return 0
}

func (p *PhysicsEngine) IsRigidBody(entityId uint32) bool {
	_, exist := p.rigidBodyMap[entityId]
	return exist
}

func (p *PhysicsEngine) CreateRigidBody(entityId, avatarEntityId, sceneId uint32, x, y, z float32, pitchAngle, yawAngle float32) {
	pitchAngle += p.pitchAngleOffset
	vy := math.Sin(float64(pitchAngle)/360.0*2*math.Pi) * float64(p.initSpeed)
	vxz := math.Cos(float64(pitchAngle)/360.0*2*math.Pi) * float64(p.initSpeed)
	vx := math.Sin(float64(yawAngle)/360.0*2*math.Pi) * vxz
	vz := math.Cos(float64(yawAngle)/360.0*2*math.Pi) * vxz
	rigidBody := &RigidBody{
		entityId:       entityId,
		avatarEntityId: avatarEntityId,
		sceneId:        sceneId,
		position:       &alg.Vector3{X: x, Y: y, Z: z},
		velocity:       &alg.Vector3{X: float32(vx), Y: float32(vy), Z: float32(vz)},
	}
	logger.Debug("[CreateRigidBody] e: %v, s: %v, p: %v, v: %v", rigidBody.entityId, rigidBody.sceneId, rigidBody.position, rigidBody.velocity)
	p.rigidBodyMap[entityId] = rigidBody
}

func (p *PhysicsEngine) DestroyRigidBody(entityId uint32) {
	if !p.IsRigidBody(entityId) {
		return
	}
	rigidBody := p.rigidBodyMap[entityId]
	logger.Debug("[DestroyRigidBody] e: %v, s: %v, p: %v, v: %v", rigidBody.entityId, rigidBody.sceneId, rigidBody.position, rigidBody.velocity)
	delete(p.rigidBodyMap, entityId)
}
