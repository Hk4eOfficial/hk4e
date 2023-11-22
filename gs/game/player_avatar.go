package game

import (
	"strconv"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/pkg/object"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

/************************************************** 接口请求 **************************************************/

// AvatarUpgradeReq 角色升级请求
func (g *Game) AvatarUpgradeReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AvatarUpgradeReq)
	// 是否拥有角色
	avatar, ok := player.GameObjectGuidMap[req.AvatarGuid].(*model.Avatar)
	if !ok {
		logger.Error("avatar error, avatarGuid: %v", req.AvatarGuid)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{}, proto.Retcode_RET_CAN_NOT_FIND_AVATAR)
		return
	}
	// 获取经验书物品配置表
	itemDataConfig := gdconf.GetItemDataById(int32(req.ItemId))
	if itemDataConfig == nil {
		logger.Error("item data config error, itemId: %v", constant.ITEM_ID_SCOIN)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{}, proto.Retcode_RET_ITEM_NOT_EXIST)
		return
	}
	// 经验书将给予的经验数
	itemParam, err := strconv.Atoi(itemDataConfig.Use1Param1)
	if err != nil {
		logger.Error("parse item param error: %v", err)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{})
		return
	}
	// 角色获得的经验
	expCount := uint32(itemParam) * req.Count
	// 摩拉数量是否足够
	if g.GetPlayerItemCount(player.PlayerId, constant.ITEM_ID_SCOIN) < expCount/5 {
		logger.Error("item count not enough, itemId: %v", constant.ITEM_ID_SCOIN)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{}, proto.Retcode_RET_SCOIN_NOT_ENOUGH)
		return
	}
	// 获取角色配置表
	avatarDataConfig := gdconf.GetAvatarDataById(int32(avatar.AvatarId))
	if avatarDataConfig == nil {
		logger.Error("avatar config error, avatarId: %v", avatar.AvatarId)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{})
		return
	}
	// 获取角色突破配置表
	avatarPromoteConfig := gdconf.GetAvatarPromoteDataByIdAndLevel(avatarDataConfig.PromoteId, int32(avatar.Promote))
	if avatarPromoteConfig == nil {
		logger.Error("avatar promote config error, promoteLevel: %v", avatar.Promote)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{})
		return
	}
	// 角色等级是否达到限制
	if avatar.Level >= uint8(avatarPromoteConfig.LevelLimit) {
		logger.Error("avatar level ge level limit, level: %v", avatar.Level)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{}, proto.Retcode_RET_AVATAR_BREAK_LEVEL_LESS_THAN)
		return
	}
	// 消耗升级材料以及摩拉
	ok = g.CostPlayerItem(player.PlayerId, []*ChangeItem{
		{ItemId: req.ItemId, ChangeCount: req.Count},
		{ItemId: constant.ITEM_ID_SCOIN, ChangeCount: expCount / 5},
	})
	if !ok {
		logger.Error("item count not enough, uid: %v", player.PlayerId)
		g.SendError(cmd.AvatarUpgradeRsp, player, &proto.AvatarUpgradeRsp{}, proto.Retcode_RET_ITEM_COUNT_NOT_ENOUGH)
		return
	}
	// 角色升级前的信息
	oldLevel := avatar.Level
	oldFightPropMap := make(map[uint32]float32, len(avatar.FightPropMap))
	for propType, propValue := range avatar.FightPropMap {
		oldFightPropMap[propType] = propValue
	}

	// 角色添加经验
	g.UpgradePlayerAvatar(player, avatar, expCount)

	avatarUpgradeRsp := &proto.AvatarUpgradeRsp{
		CurLevel:        uint32(avatar.Level),
		OldLevel:        uint32(oldLevel),
		OldFightPropMap: oldFightPropMap,
		CurFightPropMap: avatar.FightPropMap,
		AvatarGuid:      req.AvatarGuid,
	}
	g.SendMsg(cmd.AvatarUpgradeRsp, player.PlayerId, player.ClientSeq, avatarUpgradeRsp)
}

// AvatarPromoteReq 角色突破请求
func (g *Game) AvatarPromoteReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AvatarPromoteReq)
	// 是否拥有角色
	avatar, ok := player.GameObjectGuidMap[req.Guid].(*model.Avatar)
	if !ok {
		logger.Error("avatar error, avatarGuid: %v", req.Guid)
		g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{}, proto.Retcode_RET_CAN_NOT_FIND_AVATAR)
		return
	}
	// 获取角色配置表
	avatarDataConfig := gdconf.GetAvatarDataById(int32(avatar.AvatarId))
	if avatarDataConfig == nil {
		logger.Error("avatar config error, avatarId: %v", avatar.AvatarId)
		g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{})
		return
	}
	// 获取角色突破配置表
	avatarPromoteConfig := gdconf.GetAvatarPromoteDataByIdAndLevel(avatarDataConfig.PromoteId, int32(avatar.Promote))
	if avatarPromoteConfig == nil {
		logger.Error("avatar promote config error, promoteLevel: %v", avatar.Promote)
		g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{})
		return
	}
	// 角色等级是否达到限制
	if avatar.Level < uint8(avatarPromoteConfig.LevelLimit) {
		logger.Error("avatar level le level limit, level: %v", avatar.Level)
		g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{}, proto.Retcode_RET_AVATAR_LEVEL_LESS_THAN)
		return
	}
	// 获取角色突破下一级的配置表
	avatarPromoteConfig = gdconf.GetAvatarPromoteDataByIdAndLevel(avatarDataConfig.PromoteId, int32(avatar.Promote+1))
	if avatarPromoteConfig == nil {
		logger.Error("avatar promote config error, next promoteLevel: %v", avatar.Promote+1)
		g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{}, proto.Retcode_RET_AVATAR_ON_MAX_BREAK_LEVEL)
		return
	}
	// 将被消耗的物品列表
	costItemList := make([]*ChangeItem, 0, len(avatarPromoteConfig.CostItemMap)+1)
	// 突破材料是否足够并添加到消耗物品列表
	for itemId, count := range avatarPromoteConfig.CostItemMap {
		costItemList = append(costItemList, &ChangeItem{
			ItemId:      itemId,
			ChangeCount: count,
		})
	}
	// 消耗列表添加摩拉的消耗
	costItemList = append(costItemList, &ChangeItem{
		ItemId:      constant.ITEM_ID_SCOIN,
		ChangeCount: uint32(avatarPromoteConfig.CostCoin),
	})
	// 突破材料以及摩拉是否足够
	for _, item := range costItemList {
		if g.GetPlayerItemCount(player.PlayerId, item.ItemId) < item.ChangeCount {
			logger.Error("item count not enough, itemId: %v", item.ItemId)
			// 摩拉的错误提示与材料不同
			if item.ItemId == constant.ITEM_ID_SCOIN {
				g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{}, proto.Retcode_RET_SCOIN_NOT_ENOUGH)
			}
			g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{}, proto.Retcode_RET_ITEM_COUNT_NOT_ENOUGH)
			return
		}
	}
	// 冒险等级是否符合要求
	if player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL] < uint32(avatarPromoteConfig.MinPlayerLevel) {
		logger.Error("player level not enough, level: %v", player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL])
		g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{}, proto.Retcode_RET_PLAYER_LEVEL_LESS_THAN)
		return
	}
	// 消耗突破材料和摩拉
	ok = g.CostPlayerItem(player.PlayerId, costItemList)
	if !ok {
		logger.Error("item count not enough, uid: %v", player.PlayerId)
		g.SendError(cmd.AvatarPromoteRsp, player, &proto.AvatarPromoteRsp{}, proto.Retcode_RET_ITEM_COUNT_NOT_ENOUGH)
		return
	}

	// 角色突破等级+1
	avatar.Promote++
	// 角色更新面板
	g.UpdatePlayerAvatarFightProp(player.PlayerId, avatar.AvatarId)
	// 角色属性表更新通知
	g.SendMsg(cmd.AvatarPropNotify, player.PlayerId, player.ClientSeq, g.PacketAvatarPropNotify(avatar))

	avatarPromoteRsp := &proto.AvatarPromoteRsp{
		Guid: req.Guid,
	}
	g.SendMsg(cmd.AvatarPromoteRsp, player.PlayerId, player.ClientSeq, avatarPromoteRsp)
}

// AvatarPromoteGetRewardReq 角色突破获取奖励请求
func (g *Game) AvatarPromoteGetRewardReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AvatarPromoteGetRewardReq)
	// 是否拥有角色
	avatar, ok := player.GameObjectGuidMap[req.AvatarGuid].(*model.Avatar)
	if !ok {
		logger.Error("avatar error, avatarGuid: %v", req.AvatarGuid)
		g.SendError(cmd.AvatarPromoteGetRewardRsp, player, &proto.AvatarPromoteGetRewardRsp{}, proto.Retcode_RET_CAN_NOT_FIND_AVATAR)
		return
	}
	// 获取角色配置表
	avatarDataConfig := gdconf.GetAvatarDataById(int32(avatar.AvatarId))
	if avatarDataConfig == nil {
		logger.Error("avatar config error, avatarId: %v", avatar.AvatarId)
		g.SendError(cmd.AvatarPromoteGetRewardRsp, player, &proto.AvatarPromoteGetRewardRsp{})
		return
	}
	// 角色是否获取过该突破等级的奖励
	if avatar.PromoteRewardMap[req.PromoteLevel] {
		logger.Error("avatar config error, avatarId: %v", avatar.AvatarId)
		g.SendError(cmd.AvatarPromoteGetRewardRsp, player, &proto.AvatarPromoteGetRewardRsp{}, proto.Retcode_RET_REWARD_HAS_TAKEN)
		return
	}
	// 获取奖励配置表
	rewardConfig := gdconf.GetRewardDataById(int32(avatarDataConfig.PromoteRewardMap[req.PromoteLevel]))
	if rewardConfig == nil {
		logger.Error("reward config error, rewardId: %v", avatarDataConfig.PromoteRewardMap[req.PromoteLevel])
		g.SendError(cmd.AvatarPromoteGetRewardRsp, player, &proto.AvatarPromoteGetRewardRsp{})
		return
	}
	// 设置该奖励为已被获取状态
	avatar.PromoteRewardMap[req.PromoteLevel] = true
	// 给予突破奖励
	rewardItemList := make([]*ChangeItem, 0, len(rewardConfig.RewardItemMap))
	for itemId, count := range rewardConfig.RewardItemMap {
		rewardItemList = append(rewardItemList, &ChangeItem{
			ItemId:      itemId,
			ChangeCount: count,
		})
	}
	g.AddPlayerItem(player.PlayerId, rewardItemList, proto.ActionReasonType_ACTION_REASON_AVATAR_PROMOTE)

	avatarPromoteGetRewardRsp := &proto.AvatarPromoteGetRewardRsp{
		RewardId:     uint32(rewardConfig.RewardId),
		AvatarGuid:   req.AvatarGuid,
		PromoteLevel: req.PromoteLevel,
	}
	g.SendMsg(cmd.AvatarPromoteGetRewardRsp, player.PlayerId, player.ClientSeq, avatarPromoteGetRewardRsp)
}

// AvatarWearFlycloakReq 角色装备风之翼请求
func (g *Game) AvatarWearFlycloakReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AvatarWearFlycloakReq)

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		g.SendError(cmd.AvatarWearFlycloakRsp, player, &proto.AvatarWearFlycloakRsp{})
		return
	}
	scene := world.GetSceneById(player.GetSceneId())

	// 确保角色存在
	avatar, ok := player.GameObjectGuidMap[req.AvatarGuid].(*model.Avatar)
	if !ok {
		logger.Error("avatar error, avatarGuid: %v", req.AvatarGuid)
		g.SendError(cmd.AvatarWearFlycloakRsp, player, &proto.AvatarWearFlycloakRsp{}, proto.Retcode_RET_CAN_NOT_FIND_AVATAR)
		return
	}

	// 确保要更换的风之翼已获得
	exist := false
	dbAvatar := player.GetDbAvatar()
	for _, v := range dbAvatar.FlyCloakList {
		if v == req.FlycloakId {
			exist = true
		}
	}
	if !exist {
		logger.Error("flycloak not exist, flycloakId: %v", req.FlycloakId)
		g.SendError(cmd.AvatarWearFlycloakRsp, player, &proto.AvatarWearFlycloakRsp{}, proto.Retcode_RET_NOT_HAS_FLYCLOAK)
		return
	}

	// 设置角色风之翼
	avatar.FlyCloak = req.FlycloakId

	ntf := &proto.AvatarFlycloakChangeNotify{
		AvatarGuid: req.AvatarGuid,
		FlycloakId: req.FlycloakId,
	}
	g.SendToSceneA(scene, cmd.AvatarFlycloakChangeNotify, player.ClientSeq, ntf, 0)

	rsp := &proto.AvatarWearFlycloakRsp{
		AvatarGuid: req.AvatarGuid,
		FlycloakId: req.FlycloakId,
	}
	g.SendMsg(cmd.AvatarWearFlycloakRsp, player.PlayerId, player.ClientSeq, rsp)
}

// AvatarChangeCostumeReq 角色更换时装请求
func (g *Game) AvatarChangeCostumeReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AvatarChangeCostumeReq)

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		g.SendError(cmd.AvatarChangeCostumeRsp, player, &proto.AvatarChangeCostumeRsp{})
		return
	}
	scene := world.GetSceneById(player.GetSceneId())

	// 确保角色存在
	avatar, ok := player.GameObjectGuidMap[req.AvatarGuid].(*model.Avatar)
	if !ok {
		logger.Error("avatar error, avatarGuid: %v", req.AvatarGuid)
		g.SendError(cmd.AvatarChangeCostumeRsp, player, &proto.AvatarChangeCostumeRsp{}, proto.Retcode_RET_COSTUME_AVATAR_ERROR)
		return
	}

	// 确保要更换的时装已获得
	exist := false
	dbAvatar := player.GetDbAvatar()
	for _, v := range dbAvatar.CostumeList {
		if v == req.CostumeId {
			exist = true
		}
	}
	if req.CostumeId == 0 {
		exist = true
	}
	if !exist {
		logger.Error("costume not exist, costumeId: %v", req.CostumeId)
		g.SendError(cmd.AvatarChangeCostumeRsp, player, &proto.AvatarChangeCostumeRsp{}, proto.Retcode_RET_NOT_HAS_COSTUME)
		return
	}

	// 设置角色时装
	avatar.Costume = req.CostumeId

	// 角色更换时装通知
	ntf := new(proto.AvatarChangeCostumeNotify)
	// 要更换时装的角色实体不存在代表更换的是仓库内的角色
	if world.GetPlayerWorldAvatar(player, avatar.AvatarId) == nil {
		ntf.EntityInfo = &proto.SceneEntityInfo{
			Entity: &proto.SceneEntityInfo_Avatar{
				Avatar: g.PacketSceneAvatarInfo(scene, player, avatar.AvatarId),
			},
		}
	} else {
		ntf.EntityInfo = g.PacketSceneEntityInfoAvatar(scene, player, avatar.AvatarId)
	}
	g.SendToSceneA(scene, cmd.AvatarChangeCostumeNotify, player.ClientSeq, ntf, 0)

	rsp := &proto.AvatarChangeCostumeRsp{
		AvatarGuid: req.AvatarGuid,
		CostumeId:  req.CostumeId,
	}
	g.SendMsg(cmd.AvatarChangeCostumeRsp, player.PlayerId, player.ClientSeq, rsp)
}

// AvatarSkillUpgradeReq 角色技能升级请求
func (g *Game) AvatarSkillUpgradeReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AvatarSkillUpgradeReq)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		g.SendError(cmd.AvatarSkillUpgradeRsp, player, &proto.AvatarSkillUpgradeRsp{})
		return
	}
	avatar, ok := player.GameObjectGuidMap[req.AvatarGuid].(*model.Avatar)
	if !ok {
		g.SendError(cmd.AvatarSkillUpgradeRsp, player, &proto.AvatarSkillUpgradeRsp{})
		return
	}
	skillLevel, exist := avatar.SkillLevelMap[req.AvatarSkillId]
	if !exist {
		g.SendError(cmd.AvatarSkillUpgradeRsp, player, &proto.AvatarSkillUpgradeRsp{})
		return
	}
	avatarSkillDataConfig := gdconf.GetAvatarSkillDataById(int32(req.AvatarSkillId))
	if avatarSkillDataConfig == nil {
		g.SendError(cmd.AvatarSkillUpgradeRsp, player, &proto.AvatarSkillUpgradeRsp{})
		return
	}
	proudSkillDataConfig := gdconf.GetProudSkillDataByGroupIdAndLevel(avatarSkillDataConfig.UpgradeSkillGroupId, int32(skillLevel))
	if proudSkillDataConfig == nil {
		g.SendError(cmd.AvatarSkillUpgradeRsp, player, &proto.AvatarSkillUpgradeRsp{})
		return
	}

	// 消耗物品列表
	costItemList := make([]*ChangeItem, 0)
	for _, costItem := range proudSkillDataConfig.CostItemList {
		costItemList = append(costItemList, &ChangeItem{
			ItemId:      uint32(costItem.ItemId),
			ChangeCount: uint32(costItem.ItemCount),
		})
	}
	costItemList = append(costItemList, &ChangeItem{
		ItemId:      constant.ITEM_ID_SCOIN,
		ChangeCount: uint32(proudSkillDataConfig.CostSCoin),
	})
	ok = g.CostPlayerItem(player.PlayerId, costItemList)
	if !ok {
		g.SendError(cmd.AvatarSkillUpgradeRsp, player, &proto.AvatarSkillUpgradeRsp{})
		return
	}

	skillLevel++
	avatar.SkillLevelMap[req.AvatarSkillId] = skillLevel

	entityId := world.GetPlayerWorldAvatarEntityId(player, avatar.AvatarId)
	ntf := &proto.AvatarSkillChangeNotify{
		CurLevel:      skillLevel,
		AvatarGuid:    req.AvatarGuid,
		EntityId:      entityId,
		SkillDepotId:  avatar.SkillDepotId,
		OldLevel:      req.OldLevel,
		AvatarSkillId: req.AvatarSkillId,
	}
	g.SendMsg(cmd.AvatarSkillChangeNotify, player.PlayerId, player.ClientSeq, ntf)

	rsp := &proto.AvatarSkillUpgradeRsp{
		AvatarGuid:    req.AvatarGuid,
		CurLevel:      skillLevel,
		AvatarSkillId: req.AvatarSkillId,
		OldLevel:      req.OldLevel,
	}
	g.SendMsg(cmd.AvatarSkillUpgradeRsp, player.PlayerId, player.ClientSeq, rsp)
}

// UnlockAvatarTalentReq 角色命座解锁请求
func (g *Game) UnlockAvatarTalentReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.UnlockAvatarTalentReq)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
		g.SendError(cmd.UnlockAvatarTalentRsp, player, &proto.UnlockAvatarTalentRsp{})
		return
	}
	avatar, ok := player.GameObjectGuidMap[req.AvatarGuid].(*model.Avatar)
	if !ok {
		g.SendError(cmd.UnlockAvatarTalentRsp, player, &proto.UnlockAvatarTalentRsp{})
		return
	}

	ok = g.CostPlayerItem(player.PlayerId, []*ChangeItem{{ItemId: avatar.AvatarId - 10000000 + 1100, ChangeCount: 1}})
	if !ok {
		g.SendError(cmd.UnlockAvatarTalentRsp, player, &proto.UnlockAvatarTalentRsp{})
		return
	}

	avatar.TalentIdList = append(avatar.TalentIdList, req.TalentId)

	entityId := world.GetPlayerWorldAvatarEntityId(player, avatar.AvatarId)
	ntf := &proto.AvatarUnlockTalentNotify{
		EntityId:     entityId,
		AvatarGuid:   req.AvatarGuid,
		TalentId:     req.TalentId,
		SkillDepotId: avatar.SkillDepotId,
	}
	g.SendMsg(cmd.AvatarUnlockTalentNotify, player.PlayerId, player.ClientSeq, ntf)

	rsp := &proto.UnlockAvatarTalentRsp{
		TalentId:   req.TalentId,
		AvatarGuid: req.AvatarGuid,
	}
	g.SendMsg(cmd.UnlockAvatarTalentRsp, player.PlayerId, player.ClientSeq, rsp)
}

/************************************************** 游戏功能 **************************************************/

// GetAllAvatarDataConfig 获取所有角色数据配置表
func (g *Game) GetAllAvatarDataConfig() map[int32]*gdconf.AvatarData {
	allAvatarDataConfig := make(map[int32]*gdconf.AvatarData)
	for avatarId, avatarData := range gdconf.GetAvatarDataMap() {
		if avatarId <= 10000001 || avatarId >= 11000000 {
			// 跳过无效角色
			continue
		}
		if avatarId == 10000005 || avatarId == 10000007 {
			// 跳过主角
			continue
		}
		allAvatarDataConfig[avatarId] = avatarData
	}
	return allAvatarDataConfig
}

// AddPlayerAvatar 给予玩家角色
func (g *Game) AddPlayerAvatar(userId uint32, avatarId uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	// 判断玩家是否已有该角色
	dbAvatar := player.GetDbAvatar()
	avatar := dbAvatar.GetAvatarById(avatarId)
	if avatar != nil {
		return
	}
	dbAvatar.AddAvatar(player, avatarId)

	// 添加初始武器
	avatarDataConfig := gdconf.GetAvatarDataById(int32(avatarId))
	if avatarDataConfig == nil {
		logger.Error("config is nil, itemId: %v", avatarId)
		return
	}
	weaponId := g.AddPlayerWeapon(player.PlayerId, uint32(avatarDataConfig.InitialWeapon))

	// 角色装上初始武器
	g.WearPlayerAvatarWeapon(player.PlayerId, avatarId, weaponId)

	g.UpdatePlayerAvatarFightProp(player.PlayerId, avatarId)

	avatarAddNotify := &proto.AvatarAddNotify{
		Avatar:   g.PacketAvatarInfo(dbAvatar.GetAvatarById(avatarId)),
		IsInTeam: false,
	}
	g.SendMsg(cmd.AvatarAddNotify, userId, player.ClientSeq, avatarAddNotify)

	dbTeam := player.GetDbTeam()
	if len(dbTeam.GetActiveTeam().GetAvatarIdList()) >= 4 {
		return
	}
	activeTeam := dbTeam.GetActiveTeam()
	g.ChangeTeam(player, uint32(dbTeam.GetActiveTeamId()), append(activeTeam.GetAvatarIdList(), avatarId), dbTeam.GetActiveAvatarId())
}

// AddPlayerFlycloak 给予玩家风之翼
func (g *Game) AddPlayerFlycloak(userId uint32, flyCloakId uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	// 验证玩家是否已拥有该风之翼
	dbAvatar := player.GetDbAvatar()
	for _, flycloak := range dbAvatar.FlyCloakList {
		if flycloak == flyCloakId {
			return
		}
	}
	dbAvatar.FlyCloakList = append(dbAvatar.FlyCloakList, flyCloakId)

	avatarGainFlycloakNotify := &proto.AvatarGainFlycloakNotify{
		FlycloakId: flyCloakId,
	}
	g.SendMsg(cmd.AvatarGainFlycloakNotify, userId, player.ClientSeq, avatarGainFlycloakNotify)
}

// AddPlayerCostume 给予玩家时装
func (g *Game) AddPlayerCostume(userId uint32, costumeId uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	// 验证玩家是否已拥有该时装
	dbAvatar := player.GetDbAvatar()
	for _, costume := range dbAvatar.CostumeList {
		if costume == costumeId {
			return
		}
	}
	dbAvatar.CostumeList = append(dbAvatar.CostumeList, costumeId)

	avatarGainCostumeNotify := &proto.AvatarGainCostumeNotify{
		CostumeId: costumeId,
	}
	g.SendMsg(cmd.AvatarGainCostumeNotify, userId, player.ClientSeq, avatarGainCostumeNotify)
}

// UpgradePlayerAvatar 玩家角色升级
func (g *Game) UpgradePlayerAvatar(player *model.Player, avatar *model.Avatar, expCount uint32) {
	// 获取角色配置表
	avatarDataConfig := gdconf.GetAvatarDataById(int32(avatar.AvatarId))
	if avatarDataConfig == nil {
		logger.Error("avatar config error, avatarId: %v", avatar.AvatarId)
		return
	}
	// 获取角色突破配置表
	avatarPromoteConfig := gdconf.GetAvatarPromoteDataByIdAndLevel(avatarDataConfig.PromoteId, int32(avatar.Promote))
	if avatarPromoteConfig == nil {
		logger.Error("avatar promote config error, promoteLevel: %v", avatar.Promote)
		return
	}
	// 角色增加经验
	avatar.Exp += expCount
	// 角色升级
	for {
		// 获取角色等级配置表
		avatarLevelConfig := gdconf.GetAvatarLevelDataByLevel(int32(avatar.Level))
		if avatarLevelConfig == nil {
			// 获取不到代表已经到达最大等级
			break
		}
		// 角色当前等级未突破则跳出循环
		if avatar.Level >= uint8(avatarPromoteConfig.LevelLimit) {
			// 角色未突破溢出的经验处理
			avatar.Exp = 0
			break
		}
		// 角色经验小于升级所需的经验则跳出循环
		if avatar.Exp < uint32(avatarLevelConfig.Exp) {
			break
		}
		// 角色等级提升
		avatar.Exp -= uint32(avatarLevelConfig.Exp)
		avatar.Level++
	}
	// 角色更新面板
	g.UpdatePlayerAvatarFightProp(player.PlayerId, avatar.AvatarId)
	// 角色属性表更新通知
	g.SendMsg(cmd.AvatarPropNotify, player.PlayerId, player.ClientSeq, g.PacketAvatarPropNotify(avatar))

	g.AddPlayerAvatarHp(player.PlayerId, avatar.AvatarId, 0.0, true, proto.ChangHpReason_CHANGE_HP_ADD_UPGRADE)
}

// UpdatePlayerAvatarFightProp 更新玩家角色战斗属性
func (g *Game) UpdatePlayerAvatarFightProp(userId uint32, avatarId uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil || WORLD_MANAGER.IsAiWorld(world) {
		return
	}

	dbAvatar := player.GetDbAvatar()
	avatar := dbAvatar.GetAvatarById(avatarId)
	if avatar == nil {
		logger.Error("get avatar is nil, avatarId: %v", avatarId)
		return
	}

	// 更新角色面板
	dbAvatar.UpdateAvatarFightProp(avatar)

	avatarFightPropNotify := &proto.AvatarFightPropNotify{
		AvatarGuid:   avatar.Guid,
		FightPropMap: avatar.FightPropMap,
	}
	g.SendMsg(cmd.AvatarFightPropNotify, userId, player.ClientSeq, avatarFightPropNotify)
}

// ChangePlayerAvatarSkillDepot 改变角色技能库
func (g *Game) ChangePlayerAvatarSkillDepot(userId uint32, avatarId uint32, changeSkillDepotId uint32, elementType int) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	if changeSkillDepotId == 0 {
		avatarDataConfig := gdconf.GetAvatarDataById(int32(avatarId))
		if avatarDataConfig == nil {
			return
		}
		for _, skillDepotId := range avatarDataConfig.SkillDepotIdList {
			skillDepotDataConfig := gdconf.GetAvatarSkillDepotDataById(skillDepotId)
			if skillDepotDataConfig == nil {
				continue
			}
			avatarSkillDataConfig := gdconf.GetAvatarSkillDataById(skillDepotDataConfig.EnergySkill)
			if avatarSkillDataConfig == nil {
				continue
			}
			if avatarSkillDataConfig.CostElemType != int32(elementType) {
				continue
			}
			changeSkillDepotId = uint32(skillDepotId)
			break
		}
	}
	dbAvatar := player.GetDbAvatar()
	avatar := dbAvatar.GetAvatarById(avatarId)
	if avatar == nil {
		logger.Error("get avatar is nil, avatarId: %v", avatarId)
		return
	}

	dbAvatar.ChangeSkillDepot(avatarId, changeSkillDepotId)
	entityId := world.GetPlayerWorldAvatarEntityId(player, avatarId)

	g.SendMsg(cmd.AvatarSkillDepotChangeNotify, player.PlayerId, player.ClientSeq, &proto.AvatarSkillDepotChangeNotify{
		EntityId:      entityId,
		AvatarGuid:    avatar.Guid,
		SkillDepotId:  changeSkillDepotId,
		SkillLevelMap: avatar.SkillLevelMap,
	})

	g.SendMsg(cmd.AbilityChangeNotify, player.PlayerId, player.ClientSeq, &proto.AbilityChangeNotify{
		EntityId:            entityId,
		AbilityControlBlock: g.PacketAvatarAbilityControlBlock(avatar.AvatarId, changeSkillDepotId),
	})

	avatarSkillDataConfig := gdconf.GetAvatarEnergySkillConfig(avatar.SkillDepotId)
	if avatarSkillDataConfig == nil {
		return
	}
	fightPropEnergy := constant.ELEMENT_TYPE_FIGHT_PROP_ENERGY_MAP[int(avatarSkillDataConfig.CostElemType)]
	avatar.FightPropMap[uint32(fightPropEnergy.MaxEnergy)] = float32(avatarSkillDataConfig.CostElemVal)
	avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] = float32(avatar.CurrEnergy)
	g.UpdatePlayerAvatarFightProp(player.PlayerId, avatarId)
}

// AddPlayerAvatarHp 角色加血
func (g *Game) AddPlayerAvatarHp(userId uint32, avatarId uint32, value float32, max bool, reason proto.ChangHpReason) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	entityId := world.GetPlayerWorldAvatarEntityId(player, avatarId)
	if entityId == 0 {
		return
	}
	entity := scene.GetEntity(entityId)
	fightProp := entity.GetFightProp()
	currHp := fightProp[constant.FIGHT_PROP_CUR_HP]
	maxHp := fightProp[constant.FIGHT_PROP_MAX_HP]
	deltaHp := float32(0.0)
	if max {
		deltaHp = maxHp - currHp
		fightProp[constant.FIGHT_PROP_CUR_HP] = maxHp
	} else {
		currHp += value
		deltaHp = value
		if currHp > maxHp {
			deltaHp = value - (currHp - maxHp)
			currHp = maxHp
		}
		fightProp[constant.FIGHT_PROP_CUR_HP] = currHp
	}
	g.EntityFightPropUpdateNotifyBroadcast(scene, entity)
	g.SendMsg(cmd.EntityFightPropChangeReasonNotify, player.PlayerId, player.ClientSeq, &proto.EntityFightPropChangeReasonNotify{
		PropDelta:      deltaHp,
		ChangeHpReason: reason,
		EntityId:       entity.GetId(),
		PropType:       constant.FIGHT_PROP_CUR_HP,
	})
}

// SubPlayerAvatarHp 角色扣血
func (g *Game) SubPlayerAvatarHp(userId uint32, avatarId uint32, value float32, max bool, reason proto.ChangHpReason) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	if player.WuDi {
		return
	}
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		return
	}
	scene := world.GetSceneById(player.GetSceneId())
	entityId := world.GetPlayerWorldAvatarEntityId(player, avatarId)
	if entityId == 0 {
		return
	}
	entity := scene.GetEntity(entityId)
	fightProp := entity.GetFightProp()
	currHp := fightProp[constant.FIGHT_PROP_CUR_HP]
	deltaHp := float32(0.0)
	if max {
		deltaHp = currHp
		fightProp[constant.FIGHT_PROP_CUR_HP] = 0.0
	} else {
		currHp -= value
		deltaHp = -value
		if currHp < 0.0 {
			deltaHp = value - currHp
			currHp = 0.0
		}
		fightProp[constant.FIGHT_PROP_CUR_HP] = currHp
	}
	g.EntityFightPropUpdateNotifyBroadcast(scene, entity)
	g.SendMsg(cmd.EntityFightPropChangeReasonNotify, player.PlayerId, player.ClientSeq, &proto.EntityFightPropChangeReasonNotify{
		PropDelta:      deltaHp,
		ChangeHpReason: reason,
		EntityId:       entity.GetId(),
		PropType:       constant.FIGHT_PROP_CUR_HP,
	})
	if currHp == 0.0 {
		var dieType proto.PlayerDieType
		switch reason {
		case proto.ChangHpReason_CHANGE_HP_SUB_MONSTER:
			dieType = proto.PlayerDieType_PLAYER_DIE_KILL_BY_MONSTER
		case proto.ChangHpReason_CHANGE_HP_SUB_GEAR:
			dieType = proto.PlayerDieType_PLAYER_DIE_KILL_BY_GEAR
		case proto.ChangHpReason_CHANGE_HP_SUB_FALL:
			dieType = proto.PlayerDieType_PLAYER_DIE_FALL
		case proto.ChangHpReason_CHANGE_HP_SUB_DRAWN:
			dieType = proto.PlayerDieType_PLAYER_DIE_DRAWN
		case proto.ChangHpReason_CHANGE_HP_SUB_ABYSS:
			dieType = proto.PlayerDieType_PLAYER_DIE_ABYSS
		case proto.ChangHpReason_CHANGE_HP_SUB_GM:
			dieType = proto.PlayerDieType_PLAYER_DIE_GM
		case proto.ChangHpReason_CHANGE_HP_SUB_CLIMATE_COLD:
			dieType = proto.PlayerDieType_PLAYER_DIE_CLIMATE_COLD
		case proto.ChangHpReason_CHANGE_HP_SUB_STORM_LIGHTNING:
			dieType = proto.PlayerDieType_PLAYER_DIE_STORM_LIGHTING
		default:
			dieType = proto.PlayerDieType_PLAYER_DIE_GM
		}
		g.KillPlayerAvatar(player, entity.GetAvatarEntity().GetAvatarId(), dieType)
	}
}

// AddPlayerAvatarEnergy 角色恢复元素能量
func (g *Game) AddPlayerAvatarEnergy(userId uint32, avatarId uint32, value float32, max bool) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	dbAvatar := player.GetDbAvatar()
	avatar := dbAvatar.GetAvatarById(avatarId)
	if avatar == nil {
		logger.Error("get avatar is nil, avatarId: %v", avatarId)
		return
	}
	avatarSkillDataConfig := gdconf.GetAvatarEnergySkillConfig(avatar.SkillDepotId)
	if avatarSkillDataConfig == nil {
		logger.Error("get avatar energy skill is nil, skillDepotId: %v", avatar.SkillDepotId)
		return
	}
	fightPropEnergy := constant.ELEMENT_TYPE_FIGHT_PROP_ENERGY_MAP[int(avatarSkillDataConfig.CostElemType)]
	if max {
		avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] = float32(avatarSkillDataConfig.CostElemVal)
	} else {
		avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] += value
		if avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] > float32(avatarSkillDataConfig.CostElemVal) {
			avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] = float32(avatarSkillDataConfig.CostElemVal)
		}
	}
	g.UpdatePlayerAvatarFightProp(player.PlayerId, avatarId)
}

// CostPlayerAvatarEnergy 角色消耗元素能量
func (g *Game) CostPlayerAvatarEnergy(userId uint32, avatarId uint32, value float32, max bool) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	if player.EnergyInf {
		return
	}
	dbAvatar := player.GetDbAvatar()
	avatar := dbAvatar.GetAvatarById(avatarId)
	if avatar == nil {
		logger.Error("get avatar is nil, avatarId: %v", avatarId)
		return
	}
	avatarSkillDataConfig := gdconf.GetAvatarEnergySkillConfig(avatar.SkillDepotId)
	if avatarSkillDataConfig == nil {
		logger.Error("get avatar energy skill is nil, skillDepotId: %v", avatar.SkillDepotId)
		return
	}
	fightPropEnergy := constant.ELEMENT_TYPE_FIGHT_PROP_ENERGY_MAP[int(avatarSkillDataConfig.CostElemType)]
	if max {
		avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] = 0.0
	} else {
		avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] -= value
		if avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] < 0.0 {
			avatar.FightPropMap[uint32(fightPropEnergy.CurEnergy)] = 0.0
		}
	}
	g.UpdatePlayerAvatarFightProp(player.PlayerId, avatarId)
}

/************************************************** 打包封装 **************************************************/

// PacketAvatarInfo 打包角色信息
func (g *Game) PacketAvatarInfo(avatar *model.Avatar) *proto.AvatarInfo {
	pbAvatar := &proto.AvatarInfo{
		IsFocus:  false,
		AvatarId: avatar.AvatarId,
		Guid:     avatar.Guid,
		PropMap: map[uint32]*proto.PropValue{
			uint32(constant.PLAYER_PROP_LEVEL): {
				Type:  uint32(constant.PLAYER_PROP_LEVEL),
				Val:   int64(avatar.Level),
				Value: &proto.PropValue_Ival{Ival: int64(avatar.Level)},
			},
			uint32(constant.PLAYER_PROP_EXP): {
				Type:  uint32(constant.PLAYER_PROP_EXP),
				Val:   int64(avatar.Exp),
				Value: &proto.PropValue_Ival{Ival: int64(avatar.Exp)},
			},
			uint32(constant.PLAYER_PROP_BREAK_LEVEL): {
				Type:  uint32(constant.PLAYER_PROP_BREAK_LEVEL),
				Val:   int64(avatar.Promote),
				Value: &proto.PropValue_Ival{Ival: int64(avatar.Promote)},
			},
			uint32(constant.PLAYER_PROP_SATIATION_VAL): {
				Type:  uint32(constant.PLAYER_PROP_SATIATION_VAL),
				Val:   int64(avatar.Satiation),
				Value: &proto.PropValue_Ival{Ival: int64(avatar.Satiation)},
			},
			uint32(constant.PLAYER_PROP_SATIATION_PENALTY_TIME): {
				Type:  uint32(constant.PLAYER_PROP_SATIATION_PENALTY_TIME),
				Val:   int64(avatar.SatiationPenalty),
				Value: &proto.PropValue_Ival{Ival: int64(avatar.SatiationPenalty)},
			},
		},
		LifeState:     uint32(avatar.LifeState),
		EquipGuidList: object.ConvMapToList(avatar.EquipGuidMap),
		FightPropMap:  avatar.FightPropMap,
		SkillDepotId:  avatar.SkillDepotId,
		FetterInfo: &proto.AvatarFetterInfo{
			ExpLevel:                uint32(avatar.FetterLevel),
			ExpNumber:               avatar.FetterExp,
			FetterList:              nil,
			RewardedFetterLevelList: []uint32{10},
		},
		SkillLevelMap:            avatar.SkillLevelMap,
		TalentIdList:             avatar.TalentIdList,
		InherentProudSkillList:   gdconf.GetAvatarInherentProudSkillList(avatar.SkillDepotId, avatar.Promote),
		AvatarType:               1,
		WearingFlycloakId:        avatar.FlyCloak,
		CostumeId:                avatar.Costume,
		BornTime:                 uint32(avatar.BornTime),
		PendingPromoteRewardList: make([]uint32, 0, len(avatar.PromoteRewardMap)),
	}
	for _, v := range avatar.FetterList {
		pbAvatar.FetterInfo.FetterList = append(pbAvatar.FetterInfo.FetterList, &proto.FetterData{
			FetterId:    v,
			FetterState: constant.FETTER_STATE_FINISH,
		})
	}
	// 解锁全部资料
	for _, v := range gdconf.GetFetterIdListByAvatarId(int32(avatar.AvatarId)) {
		pbAvatar.FetterInfo.FetterList = append(pbAvatar.FetterInfo.FetterList, &proto.FetterData{
			FetterId:    uint32(v),
			FetterState: constant.FETTER_STATE_FINISH,
		})
	}
	// 突破等级奖励
	for promoteLevel, isTaken := range avatar.PromoteRewardMap {
		if !isTaken {
			pbAvatar.PendingPromoteRewardList = append(pbAvatar.PendingPromoteRewardList, promoteLevel)
		}
	}
	return pbAvatar
}

// PacketAvatarPropNotify 角色属性表更新通知
func (g *Game) PacketAvatarPropNotify(avatar *model.Avatar) *proto.AvatarPropNotify {
	avatarPropNotify := &proto.AvatarPropNotify{
		PropMap:    make(map[uint32]int64, 5),
		AvatarGuid: avatar.Guid,
	}
	// 角色等级
	avatarPropNotify.PropMap[uint32(constant.PLAYER_PROP_LEVEL)] = int64(avatar.Level)
	// 角色经验
	avatarPropNotify.PropMap[uint32(constant.PLAYER_PROP_EXP)] = int64(avatar.Exp)
	// 角色突破等级
	avatarPropNotify.PropMap[uint32(constant.PLAYER_PROP_BREAK_LEVEL)] = int64(avatar.Promote)
	// 角色饱食度
	avatarPropNotify.PropMap[uint32(constant.PLAYER_PROP_SATIATION_VAL)] = int64(avatar.Satiation)
	// 角色饱食度溢出
	avatarPropNotify.PropMap[uint32(constant.PLAYER_PROP_SATIATION_PENALTY_TIME)] = int64(avatar.SatiationPenalty)

	return avatarPropNotify
}

// PacketAvatarDataNotify 角色数据通知
func (g *Game) PacketAvatarDataNotify(player *model.Player) *proto.AvatarDataNotify {
	dbAvatar := player.GetDbAvatar()
	dbTeam := player.GetDbTeam()
	avatarDataNotify := &proto.AvatarDataNotify{
		CurAvatarTeamId:   uint32(dbTeam.GetActiveTeamId()),
		ChooseAvatarGuid:  dbAvatar.GetAvatarById(dbAvatar.MainCharAvatarId).Guid,
		OwnedFlycloakList: dbAvatar.FlyCloakList,
		// 角色衣装
		OwnedCostumeList: dbAvatar.CostumeList,
		AvatarList:       make([]*proto.AvatarInfo, 0),
		AvatarTeamMap:    make(map[uint32]*proto.AvatarTeam),
	}
	for _, avatar := range dbAvatar.GetAvatarMap() {
		pbAvatar := g.PacketAvatarInfo(avatar)
		avatarDataNotify.AvatarList = append(avatarDataNotify.AvatarList, pbAvatar)
	}
	for teamIndex, team := range dbTeam.TeamList {
		var teamAvatarGuidList []uint64 = nil
		for _, avatarId := range team.GetAvatarIdList() {
			teamAvatarGuidList = append(teamAvatarGuidList, dbAvatar.GetAvatarById(avatarId).Guid)
		}
		avatarDataNotify.AvatarTeamMap[uint32(teamIndex)+1] = &proto.AvatarTeam{
			AvatarGuidList: teamAvatarGuidList,
			TeamName:       team.Name,
		}
	}
	return avatarDataNotify
}
