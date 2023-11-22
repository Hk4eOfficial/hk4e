package constant

const (
	// 消耗耐力
	STAMINA_COST_CLIMBING_BASE   = -100  // 缓慢攀爬基数
	STAMINA_COST_CLIMB_START     = -500  // 攀爬开始
	STAMINA_COST_CLIMB_JUMP      = -2500 // 攀爬跳跃
	STAMINA_COST_DASH            = -360  // 快速跑步
	STAMINA_COST_FLY             = -60   // 滑翔
	STAMINA_COST_SPRINT          = -1800 // 冲刺
	STAMINA_COST_SWIM_DASH_START = -200  // 快速游泳开始
	STAMINA_COST_SWIM_DASH       = -204  // 快速游泳
	STAMINA_COST_SWIMMING        = -400  // 缓慢游泳
	// 恢复耐力
	STAMINA_COST_POWERED_FLY = 500 // 滑翔加速(风圈等)
	STAMINA_COST_RUN         = 500 // 正常跑步
	STAMINA_COST_STANDBY     = 500 // 站立
	STAMINA_COST_WALK        = 500 // 走路
	STAMINA_COST_FIGHT       = 500 // 战斗
	// 载具浪船
	STAMINA_COST_SKIFF_DASH    = -204 // 浪船加速
	STAMINA_COST_SKIFF_NORMAL  = 500  // 浪船正常移动 (回复耐力)
	STAMINA_COST_POWERED_SKIFF = 500  // 浪船加速(风圈等) (回复耐力)
	STAMINA_COST_IN_SKIFF      = 500  // 处于浪船中回复角色耐力 (回复耐力)
	STAMINA_COST_SKIFF_NOBODY  = 500  // 浪船无人时回复载具耐力 (回复耐力)
	// 耐力回复延迟
	STAMINA_PLAYER_RESTORE_DELAY = 15 // 玩家耐力回复延迟
)
