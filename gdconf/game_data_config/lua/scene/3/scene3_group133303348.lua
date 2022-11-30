-- 基础信息
local base_info = {
	group_id = 133303348
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
	{ config_id = 348001, gadget_id = 70330197, pos = { x = -1785.498, y = 100.380, z = 3419.387 }, rot = { x = 0.000, y = 353.761, z = 0.000 }, level = 30, area_id = 23 },
	{ config_id = 348002, gadget_id = 70330197, pos = { x = -1834.181, y = 90.962, z = 3461.101 }, rot = { x = 0.000, y = 201.834, z = 0.000 }, level = 30, area_id = 23 },
	{ config_id = 348003, gadget_id = 70330197, pos = { x = -1819.400, y = 138.876, z = 3543.155 }, rot = { x = 0.000, y = 42.551, z = 0.000 }, level = 30, area_id = 23 },
	{ config_id = 348004, gadget_id = 70217020, pos = { x = -1759.189, y = 90.700, z = 3439.914 }, rot = { x = 0.000, y = 0.000, z = 0.000 }, level = 26, drop_tag = "摩拉石箱须弥", isOneoff = true, persistent = true, explore = { name = "chest", exp = 4 }, area_id = 23 }
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
		gadgets = { 348001, 348002, 348003, 348004 },
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