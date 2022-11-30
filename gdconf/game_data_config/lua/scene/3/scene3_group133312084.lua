-- 基础信息
local base_info = {
	group_id = 133312084
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 84001, monster_id = 20011401, pos = { x = -3229.774, y = 255.371, z = 4296.358 }, rot = { x = 0.000, y = 143.064, z = 0.000 }, level = 32, drop_tag = "史莱姆", area_id = 28 },
	{ config_id = 84002, monster_id = 20011401, pos = { x = -3233.262, y = 256.190, z = 4292.107 }, rot = { x = 0.000, y = 68.508, z = 0.000 }, level = 32, drop_tag = "史莱姆", area_id = 28 },
	{ config_id = 84003, monster_id = 20011401, pos = { x = -3235.555, y = 258.146, z = 4296.286 }, rot = { x = 0.000, y = 124.907, z = 0.000 }, level = 32, drop_tag = "史莱姆", area_id = 28 }
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
		monsters = { 84001, 84002, 84003 },
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