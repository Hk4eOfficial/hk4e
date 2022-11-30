-- 基础信息
local base_info = {
	group_id = 133310523
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
}

-- 装置
gadgets = {
	{ config_id = 523001, gadget_id = 70310007, pos = { x = -3297.828, y = 230.489, z = 4590.774 }, rot = { x = 16.829, y = 150.243, z = 16.680 }, level = 32, area_id = 28 },
	{ config_id = 523002, gadget_id = 70310007, pos = { x = -3320.867, y = 225.209, z = 4617.601 }, rot = { x = 22.079, y = 84.722, z = 352.562 }, level = 32, area_id = 28 },
	{ config_id = 523003, gadget_id = 70310007, pos = { x = -3305.795, y = 229.238, z = 4630.983 }, rot = { x = 357.358, y = 200.344, z = 23.380 }, level = 32, area_id = 28 },
	{ config_id = 523004, gadget_id = 70310007, pos = { x = -3257.369, y = 221.065, z = 4612.684 }, rot = { x = 355.473, y = 169.153, z = 21.858 }, level = 32, area_id = 28 },
	{ config_id = 523005, gadget_id = 70310007, pos = { x = -3254.003, y = 220.104, z = 4610.572 }, rot = { x = 319.611, y = 324.429, z = 323.073 }, level = 32, area_id = 28 },
	{ config_id = 523006, gadget_id = 70310007, pos = { x = -3254.212, y = 219.196, z = 4602.787 }, rot = { x = 349.095, y = 312.073, z = 354.965 }, level = 32, area_id = 28 },
	{ config_id = 523007, gadget_id = 70310007, pos = { x = -3263.704, y = 222.416, z = 4600.369 }, rot = { x = 2.245, y = 27.729, z = 348.210 }, level = 32, area_id = 28 }
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
		gadgets = { 523001, 523002, 523003, 523004, 523005, 523006, 523007 },
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