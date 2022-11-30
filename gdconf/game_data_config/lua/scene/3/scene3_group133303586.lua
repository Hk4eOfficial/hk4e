-- 基础信息
local base_info = {
	group_id = 133303586
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 586001, monster_id = 28060301, pos = { x = -1724.654, y = 294.238, z = 4052.307 }, rot = { x = 0.000, y = 255.264, z = 0.000 }, level = 32, drop_tag = "鸟类", pose_id = 101, area_id = 26 },
	{ config_id = 586002, monster_id = 28060301, pos = { x = -1709.924, y = 289.385, z = 4053.810 }, rot = { x = 0.000, y = 90.817, z = 0.000 }, level = 32, drop_tag = "鸟类", pose_id = 101, area_id = 26 }
}

-- NPC
npcs = {
}

-- 装置
gadgets = {
}

-- 区域
regions = {
}

-- 触发器
triggers = {
}

-- 变量
variables = {
}

--================================================================
-- 
-- 初始化配置
-- 
--================================================================

-- 初始化时创建
init_config = {
	suite = 1,
	end_suite = 0,
	rand_suite = false
}

--================================================================
-- 
-- 小组配置
-- 
--================================================================

suites = {
	{
		-- suite_id = 1,
		-- description = ,
		monsters = { 586001, 586002 },
		gadgets = { },
		regions = { },
		triggers = { },
		rand_weight = 100
	}
}

--================================================================
-- 
-- 触发器
-- 
--================================================================