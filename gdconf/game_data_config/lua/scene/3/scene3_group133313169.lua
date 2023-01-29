-- 基础信息
local base_info = {
	group_id = 133313169
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 169002, monster_id = 26090901, pos = { x = -698.709, y = -229.438, z = 5794.790 }, rot = { x = 0.000, y = 151.512, z = 0.000 }, level = 32, drop_tag = "蕈兽", pose_id = 101, area_id = 32 },
	{ config_id = 169003, monster_id = 26090901, pos = { x = -699.997, y = -229.438, z = 5793.355 }, rot = { x = 0.000, y = 122.470, z = 0.000 }, level = 32, drop_tag = "蕈兽", pose_id = 101, area_id = 32 }
}

-- NPC
npcs = {
}

-- 装置
gadgets = {
	{ config_id = 169001, gadget_id = 70330342, pos = { x = -697.990, y = -229.438, z = 5792.888 }, rot = { x = 0.000, y = 336.223, z = 0.000 }, level = 26, drop_tag = "摩拉石箱须弥", isOneoff = true, persistent = true, explore = { name = "chest", exp = 4 }, area_id = 32 }
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
		monsters = { 169002, 169003 },
		gadgets = { 169001 },
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