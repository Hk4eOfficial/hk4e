-- 基础信息
local base_info = {
	group_id = 133309176
}

--================================================================
-- 
-- 配置
-- 
--================================================================

-- 怪物
monsters = {
	{ config_id = 176002, monster_id = 26090201, pos = { x = -2665.719, y = -21.405, z = 5845.318 }, rot = { x = 0.000, y = 253.133, z = 0.000 }, level = 32, drop_tag = "蕈兽", pose_id = 104, area_id = 27 },
	{ config_id = 176003, monster_id = 26090101, pos = { x = -2674.652, y = -21.897, z = 5831.763 }, rot = { x = 0.000, y = 263.766, z = 0.000 }, level = 32, drop_tag = "蕈兽", pose_id = 104, area_id = 27 }
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
		monsters = { 176002, 176003 },
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