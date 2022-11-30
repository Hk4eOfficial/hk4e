-- 基础信息
local base_info = {
	group_id = 250015021
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 21004, monster_id = 28020403, pos = { x = -36.641, y = 0.500, z = -58.276 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 1, pose_id = 1 }
}

-- NPC
npcs = {
}

-- 装置
gadgets = {
	{ config_id = 21001, gadget_id = 70210104, pos = { x = -71.154, y = 0.500, z = -91.468 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 1, chest_drop_id = 1000100, drop_count = 1 },
	{ config_id = 21002, gadget_id = 70210104, pos = { x = -39.173, y = 0.500, z = -92.497 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 1, chest_drop_id = 1000100, drop_count = 1 },
	{ config_id = 21003, gadget_id = 70210104, pos = { x = -54.240, y = 0.500, z = -58.531 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 1, chest_drop_id = 1000100, drop_count = 1 },
	{ config_id = 21005, gadget_id = 70210104, pos = { x = -57.044, y = 0.500, z = -76.627 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 1, chest_drop_id = 1000100, drop_count = 1 }
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
		monsters = { 21004 },
		gadgets = { 21001, 21002, 21003, 21005 },
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