-- 基础信息
local base_info = {
	group_id = 133209084
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
}

-- NPC
npcs = {
	{ config_id = 84001, npc_id = 30211, pos = { x = -2594.811, y = 218.083, z = -3848.253 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, area_id = 11 }
}

-- 装置
gadgets = {
	{ config_id = 84003, gadget_id = 70710136, pos = { x = -2594.800, y = 218.086, z = -3848.356 }, rot = { x = 51.256, y = 351.000, z = 359.908 }, level = 1, area_id = 11 },
	{ config_id = 84004, gadget_id = 70710718, pos = { x = -2594.884, y = 218.088, z = -3848.257 }, rot = { x = 353.489, y = 86.790, z = 44.253 }, level = 1, area_id = 11 }
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
		monsters = { },
		gadgets = { 84003, 84004 },
		regions = { },
		triggers = { },
		npcs = { 84001 },
		rand_weight = 100
	},
	{
		-- suite_id = 2,
		-- description = ,
		monsters = { },
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