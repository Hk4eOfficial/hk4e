-- 基础信息
local base_info = {
	group_id = 133307220
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 220001, monster_id = 25410101, pos = { x = -1400.253, y = 1.971, z = 5362.782 }, rot = { x = 0.000, y = 334.272, z = 0.000 }, level = 32, drop_tag = "高级镀金旅团", disableWander = true, area_id = 32 },
	{ config_id = 220002, monster_id = 25210303, pos = { x = -1367.098, y = 6.547, z = 5347.341 }, rot = { x = 0.000, y = 50.708, z = 0.000 }, level = 32, drop_tag = "镀金旅团", area_id = 32 },
	{ config_id = 220003, monster_id = 25210302, pos = { x = -1338.400, y = -4.703, z = 5382.910 }, rot = { x = 0.000, y = 50.708, z = 0.000 }, level = 32, drop_tag = "镀金旅团", area_id = 32 }
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
		monsters = { 220001, 220002, 220003 },
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