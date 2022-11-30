-- 基础信息
local base_info = {
	group_id = 133315276
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 276001, monster_id = 20010301, pos = { x = 338.591, y = 182.500, z = 2277.679 }, rot = { x = 0.000, y = 178.503, z = 0.000 }, level = 27, drop_tag = "史莱姆", disableWander = true, area_id = 20 },
	{ config_id = 276002, monster_id = 20010301, pos = { x = 334.981, y = 182.500, z = 2275.040 }, rot = { x = 0.000, y = 228.037, z = 0.000 }, level = 27, drop_tag = "史莱姆", disableWander = true, area_id = 20 },
	{ config_id = 276003, monster_id = 20010301, pos = { x = 339.003, y = 182.500, z = 2268.912 }, rot = { x = 0.000, y = 8.720, z = 0.000 }, level = 27, drop_tag = "史莱姆", disableWander = true, area_id = 20 }
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
		monsters = { 276001, 276002, 276003 },
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