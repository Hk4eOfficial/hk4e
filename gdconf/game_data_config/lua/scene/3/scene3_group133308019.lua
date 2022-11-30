-- 基础信息
local base_info = {
	group_id = 133308019
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 19001, monster_id = 28020108, pos = { x = -2028.624, y = 156.130, z = 5581.058 }, rot = { x = 0.000, y = 279.844, z = 0.000 }, level = 32, drop_tag = "走兽", disableWander = true, area_id = 27 },
	{ config_id = 19002, monster_id = 28020108, pos = { x = -2029.089, y = 156.164, z = 5582.308 }, rot = { x = 0.000, y = 218.982, z = 0.000 }, level = 32, drop_tag = "走兽", area_id = 27 }
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
		monsters = { 19001, 19002 },
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