-- 基础信息
local base_info = {
	group_id = 133308426
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
	{ config_id = 426002, gadget_id = 70500000, pos = { x = -1869.935, y = 381.942, z = 4264.841 }, rot = { x = 3.529, y = 160.584, z = 17.084 }, level = 32, point_type = 2045, area_id = 26 },
	{ config_id = 426003, gadget_id = 70500000, pos = { x = -1856.744, y = 379.260, z = 4187.578 }, rot = { x = 354.629, y = 284.253, z = 329.382 }, level = 32, point_type = 2045, area_id = 26 },
	{ config_id = 426004, gadget_id = 70500000, pos = { x = -1825.102, y = 372.723, z = 4209.232 }, rot = { x = 7.179, y = 300.956, z = 10.395 }, level = 32, point_type = 2045, area_id = 26 },
	{ config_id = 426005, gadget_id = 70500000, pos = { x = -1844.715, y = 378.631, z = 4276.193 }, rot = { x = 347.205, y = 116.242, z = 329.737 }, level = 32, point_type = 2045, area_id = 26 },
	{ config_id = 426007, gadget_id = 70500000, pos = { x = -1871.086, y = 381.996, z = 4223.184 }, rot = { x = 5.655, y = 150.294, z = 346.763 }, level = 32, point_type = 2045, area_id = 26 }
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
		gadgets = { 426002, 426003, 426004, 426005, 426007 },
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