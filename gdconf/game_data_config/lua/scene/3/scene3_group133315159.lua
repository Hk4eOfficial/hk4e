-- 基础信息
local base_info = {
	group_id = 133315159
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
	{ config_id = 159001, gadget_id = 70500000, pos = { x = 151.288, y = 253.677, z = 2850.141 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 27, point_type = 1001, area_id = 20 },
	{ config_id = 159002, gadget_id = 70500000, pos = { x = 159.704, y = 259.748, z = 2847.992 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 27, point_type = 1001, area_id = 20 },
	{ config_id = 159003, gadget_id = 70500000, pos = { x = 160.062, y = 254.449, z = 2857.129 }, rot = { x = 27.547, y = 0.000, z = 0.000 }, level = 27, point_type = 1001, area_id = 20 },
	{ config_id = 159004, gadget_id = 70500000, pos = { x = 152.330, y = 253.153, z = 2853.044 }, rot = { x = 10.434, y = 302.191, z = 355.099 }, level = 27, point_type = 1002, area_id = 20 },
	{ config_id = 159005, gadget_id = 70500000, pos = { x = 157.260, y = 258.377, z = 2844.325 }, rot = { x = 0.000, y = 311.677, z = 0.000 }, level = 27, point_type = 1002, area_id = 20 }
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
		gadgets = { 159001, 159002, 159003, 159004, 159005 },
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