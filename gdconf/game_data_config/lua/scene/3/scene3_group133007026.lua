-- 基础信息
local base_info = {
	group_id = 133007026
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 55, monster_id = 21010501, pos = { x = 2740.844, y = 239.475, z = 101.545 }, rot = { x = 0.000, y = 199.318, z = 0.000 }, level = 24, drop_tag = "远程丘丘人", disableWander = true, pose_id = 32, area_id = 4 }
}

-- NPC
npcs = {
}

-- 装置
gadgets = {
	{ config_id = 251, gadget_id = 70220014, pos = { x = 2730.369, y = 239.229, z = 87.958 }, rot = { x = 0.000, y = 271.035, z = 0.000 }, level = 23, area_id = 4 },
	{ config_id = 252, gadget_id = 70220014, pos = { x = 2731.928, y = 239.324, z = 88.224 }, rot = { x = 0.000, y = 252.225, z = 0.000 }, level = 23, area_id = 4 },
	{ config_id = 253, gadget_id = 70220005, pos = { x = 2740.448, y = 239.438, z = 102.672 }, rot = { x = 0.000, y = 48.566, z = 0.000 }, level = 23, area_id = 4 },
	{ config_id = 254, gadget_id = 70220005, pos = { x = 2736.191, y = 239.294, z = 103.648 }, rot = { x = 0.000, y = 146.317, z = 0.000 }, level = 23, area_id = 4 },
	{ config_id = 255, gadget_id = 70220020, pos = { x = 2733.666, y = 239.317, z = 96.864 }, rot = { x = 0.000, y = 219.941, z = 0.000 }, level = 23, area_id = 4 },
	{ config_id = 256, gadget_id = 70220020, pos = { x = 2737.749, y = 239.430, z = 90.428 }, rot = { x = 0.000, y = 154.117, z = 0.000 }, level = 23, area_id = 4 },
	{ config_id = 592, gadget_id = 70211101, pos = { x = 2734.920, y = 219.001, z = 95.254 }, rot = { x = 0.000, y = 118.547, z = 0.000 }, level = 21, drop_tag = "解谜低级蒙德", isOneoff = true, persistent = true, explore = { name = "chest", exp = 1 }, area_id = 4 }
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
	rand_suite = true
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
		monsters = { 55 },
		gadgets = { 251, 252, 253, 254, 255, 256, 592 },
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