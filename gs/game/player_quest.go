package game

import (
	"strconv"
	"strings"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

/************************************************** 接口请求 **************************************************/

// AddQuestContentProgressReq 添加任务内容进度请求
func (g *Game) AddQuestContentProgressReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.AddQuestContentProgressReq)

	g.TriggerQuest(player, int32(req.ContentType), "", int32(req.Param))

	rsp := &proto.AddQuestContentProgressRsp{
		ContentType: req.ContentType,
	}
	g.SendMsg(cmd.AddQuestContentProgressRsp, player.PlayerId, player.ClientSeq, rsp)

	g.AcceptQuest(player, true)
}

func (g *Game) QuestCreateEntityReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.QuestCreateEntityReq)
	itemDataConfig := gdconf.GetItemDataById(int32(req.Entity.GetItemId()))
	if itemDataConfig == nil {
		g.SendError(cmd.QuestCreateEntityRsp, player, &proto.QuestCreateEntityRsp{})
		return
	}
	pos := &model.Vector{X: float64(req.Entity.Pos.X), Y: float64(req.Entity.Pos.Y), Z: float64(req.Entity.Pos.Z)}
	entityId := g.CreateDropGadget(player, pos, uint32(itemDataConfig.GadgetId), req.Entity.GetItemId(), 1)
	if entityId == 0 {
		g.SendError(cmd.QuestCreateEntityRsp, player, &proto.QuestCreateEntityRsp{})
		return
	}
	rsp := &proto.QuestCreateEntityRsp{
		QuestId:       req.QuestId,
		EntityId:      entityId,
		Entity:        req.Entity,
		ParentQuestId: req.ParentQuestId,
		IsRewind:      req.IsRewind,
	}
	g.SendMsg(cmd.QuestCreateEntityRsp, player.PlayerId, player.ClientSeq, rsp)
}

func (g *Game) QuestDestroyEntityReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.QuestDestroyEntityReq)
	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world == nil {
		g.SendError(cmd.QuestDestroyEntityRsp, player, &proto.QuestDestroyEntityRsp{})
		return
	}
	scene := world.GetSceneById(req.SceneId)
	entity := scene.GetEntity(req.EntityId)
	if entity == nil {
		g.SendError(cmd.QuestDestroyEntityRsp, player, &proto.QuestDestroyEntityRsp{})
		return
	}
	scene.DestroyEntity(req.EntityId)
	g.RemoveSceneEntityNotifyBroadcast(scene, proto.VisionType_VISION_MISS, []uint32{req.EntityId}, 0)
	rsp := &proto.QuestDestroyEntityRsp{
		QuestId:  req.QuestId,
		SceneId:  req.SceneId,
		EntityId: req.EntityId,
	}
	g.SendMsg(cmd.QuestDestroyEntityRsp, player.PlayerId, player.ClientSeq, rsp)
}

func (g *Game) QuestDestroyNpcReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.QuestDestroyNpcReq)
	logger.Debug("quest destroy npc, npcId: %v, parentQuestId: %v, uid: %v", req.NpcId, req.ParentQuestId, player.PlayerId)
	rsp := &proto.QuestDestroyNpcRsp{
		NpcId:         req.NpcId,
		ParentQuestId: req.ParentQuestId,
	}
	g.SendMsg(cmd.QuestDestroyNpcRsp, player.PlayerId, player.ClientSeq, rsp)
}

/************************************************** 游戏功能 **************************************************/

const (
	QuestExecTypeFinish = iota
	QuestExecTypeFail
	QuestExecTypeStart
)

// 通用参数匹配
func matchParamEqual(param1 []int32, param2 []int32, num int) bool {
	if len(param1) != num || len(param2) != num {
		return false
	}
	for i := 0; i < num; i++ {
		if param1[i] != param2[i] {
			return false
		}
	}
	return true
}

// AcceptQuest 接取任务
func (g *Game) AcceptQuest(player *model.Player, notifyClient bool) {
	g.EndlessLoopCheck(EndlessLoopCheckTypeAcceptQuest)
	dbQuest := player.GetDbQuest()
	addQuestIdList := make([]uint32, 0)
	for _, questData := range gdconf.GetQuestDataMap() {
		if dbQuest.GetQuestById(uint32(questData.QuestId)) != nil {
			continue
		}
		acceptCondResultList := make([]bool, 0)
		for _, acceptCond := range questData.AcceptCondList {
			result := false
			switch acceptCond.Type {
			case constant.QUEST_ACCEPT_COND_TYPE_STATE_EQUAL:
				// 某个任务状态等于 参数1:任务id 参数2:任务状态
				if len(acceptCond.Param) != 2 {
					break
				}
				quest := dbQuest.GetQuestById(uint32(acceptCond.Param[0]))
				if quest == nil {
					break
				}
				if quest.State != uint8(acceptCond.Param[1]) {
					break
				}
				result = true
			case constant.QUEST_ACCEPT_COND_TYPE_STATE_NOT_EQUAL:
				// 某个任务状态不等于 参数1:任务id 参数2:任务状态
				if len(acceptCond.Param) != 2 {
					break
				}
				quest := dbQuest.GetQuestById(uint32(acceptCond.Param[0]))
				if quest == nil {
					break
				}
				if quest.State == uint8(acceptCond.Param[1]) {
					break
				}
				result = true
			default:
				break
			}
			acceptCondResultList = append(acceptCondResultList, result)
		}
		canAccept := false
		switch questData.AcceptCondCompose {
		case constant.QUEST_LOGIC_TYPE_NONE:
			fallthrough
		case constant.QUEST_LOGIC_TYPE_AND:
			canAccept = true
			for _, acceptCondResult := range acceptCondResultList {
				if !acceptCondResult {
					canAccept = false
					break
				}
			}
		case constant.QUEST_LOGIC_TYPE_OR:
			canAccept = false
			for _, acceptCondResult := range acceptCondResultList {
				if acceptCondResult {
					canAccept = true
					break
				}
			}
		}
		if canAccept {
			if questData.QuestId == 35304 {
				// TODO 任务异常的权柄释放元素爆发时没有能量
				world := WORLD_MANAGER.GetWorldById(player.WorldId)
				if world != nil {
					g.AddPlayerAvatarEnergy(player.PlayerId, world.GetPlayerActiveAvatarId(player), 0.0, true)
				}
			}
			if questData.QuestId == 35721 {
				// TODO 由于风龙任务进入秘境客户端会无限重连相关原因暂时屏蔽
				// 直接福瑞
				if player.OpenStateMap[constant.OPEN_STATE_LIMIT_REGION_FRESHMEAT] == 0 {
					COMMAND_MANAGER.gmCmd.GMFreeMode(player.PlayerId)
					for _, openStateData := range gdconf.GetOpenStateDataMap() {
						player.OpenStateMap[uint32(openStateData.OpenStateId)] = 1
					}
					GAME.SendMsg(cmd.OpenStateChangeNotify, player.PlayerId, player.ClientSeq, &proto.OpenStateChangeNotify{
						OpenStateMap: player.OpenStateMap,
					})
				}
				continue
			}
			if questData.QuestId == 36301 {
				// TODO 懒得搞
				g.SendMsg(cmd.ChapterStateNotify, player.PlayerId, player.ClientSeq, &proto.ChapterStateNotify{
					ChapterState: proto.ChapterState_CHAPTER_STATE_BEGIN,
					ChapterId:    1001,
				})
			}
			dbQuest.AddQuest(uint32(questData.QuestId))
			addQuestIdList = append(addQuestIdList, uint32(questData.QuestId))
		}
	}
	if notifyClient {
		ntf := &proto.QuestListUpdateNotify{
			QuestList: make([]*proto.Quest, 0),
		}
		for _, questId := range addQuestIdList {
			pbQuest := g.PacketQuest(player, questId)
			if pbQuest == nil {
				continue
			}
			ntf.QuestList = append(ntf.QuestList, pbQuest)
		}
		g.SendMsg(cmd.QuestListUpdateNotify, player.PlayerId, player.ClientSeq, ntf)
	}
	for _, questId := range addQuestIdList {
		g.StartQuest(player, questId, notifyClient)
	}
}

// StartQuest 开始任务
func (g *Game) StartQuest(player *model.Player, questId uint32, notifyClient bool) {
	g.EndlessLoopCheck(EndlessLoopCheckTypeStartQuest)
	dbQuest := player.GetDbQuest()
	dbQuest.StartQuest(questId)

	g.ExecQuest(player, questId, QuestExecTypeStart)
	g.QuestStartTriggerCheck(player, questId)

	if notifyClient {
		ntf := &proto.QuestListUpdateNotify{
			QuestList: make([]*proto.Quest, 0),
		}
		pbQuest := g.PacketQuest(player, questId)
		if pbQuest == nil {
			return
		}
		ntf.QuestList = append(ntf.QuestList, pbQuest)
		g.SendMsg(cmd.QuestListUpdateNotify, player.PlayerId, player.ClientSeq, ntf)
	}
}

// ExecQuest 执行任务
func (g *Game) ExecQuest(player *model.Player, questId uint32, questExecType int) {
	g.EndlessLoopCheck(EndlessLoopCheckTypeExecQuest)
	questDataConfig := gdconf.GetQuestDataById(int32(questId))
	if questDataConfig == nil {
		return
	}
	var questExecList []*gdconf.QuestExec = nil
	switch questExecType {
	case QuestExecTypeFinish:
		questExecList = questDataConfig.ExecList
	case QuestExecTypeFail:
		questExecList = questDataConfig.FailExecList
	case QuestExecTypeStart:
		questExecList = questDataConfig.StartExecList
	default:
		return
	}
	for _, questExec := range questExecList {
		switch questExec.Type {
		case constant.QUEST_EXEC_TYPE_NOTIFY_GROUP_LUA:
			// 通知LUA侧
		case constant.QUEST_EXEC_TYPE_REFRESH_GROUP_SUITE:
			// 刷新场景小组
			if len(questExec.Param) != 2 {
				continue
			}
			split := strings.Split(questExec.Param[1], ",")
			if len(split) != 2 {
				continue
			}
			groupId, err := strconv.Atoi(split[0])
			if err != nil {
				continue
			}
			suiteId, err := strconv.Atoi(split[1])
			if err != nil {
				continue
			}
			g.RefreshSceneGroupSuite(player, uint32(groupId), uint8(suiteId))
		case constant.QUEST_EXEC_TYPE_SET_OPEN_STATE:
			// 设置游戏功能开放状态
			if len(questExec.Param) != 2 {
				continue
			}
			key, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			value, err := strconv.Atoi(questExec.Param[1])
			if err != nil {
				continue
			}
			g.ChangePlayerOpenState(player.PlayerId, uint32(key), uint32(value))
		case constant.QUEST_EXEC_TYPE_UNLOCK_POINT:
			// 解锁传送点
			if len(questExec.Param) != 2 {
				continue
			}
			sceneId, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			pointId, err := strconv.Atoi(questExec.Param[1])
			if err != nil {
				continue
			}
			g.UnlockPlayerScenePoint(player, uint32(sceneId), uint32(pointId))
		case constant.QUEST_EXEC_TYPE_UNLOCK_AREA:
			// 解锁场景区域
			if len(questExec.Param) != 2 {
				continue
			}
			sceneId, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			areaId, err := strconv.Atoi(questExec.Param[1])
			if err != nil {
				continue
			}
			g.UnlockPlayerSceneArea(player, uint32(sceneId), uint32(areaId))
		case constant.QUEST_EXEC_TYPE_CHANGE_AVATAR_ELEMET:
			// 改变主角元素类型
			if len(questExec.Param) != 1 {
				continue
			}
			elementType, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			dbAvatar := player.GetDbAvatar()
			g.ChangePlayerAvatarSkillDepot(player.PlayerId, dbAvatar.MainCharAvatarId, 0, elementType)
		case constant.QUEST_EXEC_TYPE_SET_IS_FLYABLE:
			// 设置允许飞行状态
			if len(questExec.Param) != 1 {
				continue
			}
			value, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			player.PropMap[constant.PLAYER_PROP_IS_FLYABLE] = uint32(value)
			g.SendMsg(cmd.PlayerPropNotify, player.PlayerId, player.ClientSeq, g.PacketPlayerPropNotify(player, constant.PLAYER_PROP_IS_FLYABLE))
		case constant.QUEST_EXEC_TYPE_SET_IS_WEATHER_LOCKED:
			// 设置天气锁定状态
			if len(questExec.Param) != 1 {
				continue
			}
			value, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			player.PropMap[constant.PLAYER_PROP_IS_WEATHER_LOCKED] = uint32(value)
			g.SendMsg(cmd.PlayerPropNotify, player.PlayerId, player.ClientSeq, g.PacketPlayerPropNotify(player, constant.PLAYER_PROP_IS_WEATHER_LOCKED))
		case constant.QUEST_EXEC_TYPE_SET_IS_GAME_TIME_LOCKED:
			// 设置游戏时间锁定状态
			if len(questExec.Param) != 1 {
				continue
			}
			value, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			player.PropMap[constant.PLAYER_PROP_IS_GAME_TIME_LOCKED] = uint32(value)
			g.SendMsg(cmd.PlayerPropNotify, player.PlayerId, player.ClientSeq, g.PacketPlayerPropNotify(player, constant.PLAYER_PROP_IS_GAME_TIME_LOCKED))
		case constant.QUEST_EXEC_TYPE_SET_IS_TRANSFERABLE:
			// 设置允许传送状态
			if len(questExec.Param) != 1 {
				continue
			}
			value, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			player.PropMap[constant.PLAYER_PROP_IS_TRANSFERABLE] = uint32(value)
			g.SendMsg(cmd.PlayerPropNotify, player.PlayerId, player.ClientSeq, g.PacketPlayerPropNotify(player, constant.PLAYER_PROP_IS_TRANSFERABLE))
		case constant.QUEST_EXEC_TYPE_SET_GAME_TIME:
			// 设置游戏时间
			if len(questExec.Param) != 1 {
				continue
			}
			hour, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			world := WORLD_MANAGER.GetWorldById(player.WorldId)
			if world == nil {
				logger.Error("get world is nil, worldId: %v, uid: %v", player.WorldId, player.PlayerId)
				continue
			}
			scene := world.GetSceneById(player.GetSceneId())
			if scene == nil {
				logger.Error("scene is nil, sceneId: %v, uid: %v", player.GetSceneId(), player.PlayerId)
				continue
			}
			g.ChangeGameTime(scene, uint32(hour*60))
		case constant.QUEST_EXEC_TYPE_ROLLBACK_QUEST:
			// 回滚任务
			if len(questExec.Param) != 1 {
				continue
			}
			rollbackQuestId, err := strconv.Atoi(questExec.Param[0])
			if err != nil {
				continue
			}
			dbQuest := player.GetDbQuest()
			rollbackQuest := dbQuest.GetQuestById(uint32(rollbackQuestId))
			rollbackQuest.State = constant.QUEST_STATE_UNSTARTED
			g.StartQuest(player, rollbackQuest.QuestId, true)
		default:
			logger.Error("not support quest exec type: %v, uid: %v", questExec.Type, player.PlayerId)
		}
	}
}

// TriggerQuest 触发任务
func (g *Game) TriggerQuest(player *model.Player, cond int32, complexParam string, param ...int32) {
	g.EndlessLoopCheck(EndlessLoopCheckTypeTriggerQuest)
	dbQuest := player.GetDbQuest()
	updateQuestIdList := make([]uint32, 0)
	for _, quest := range dbQuest.GetQuestMap() {
		if quest.State != constant.QUEST_STATE_UNFINISHED {
			continue
		}
		questDataConfig := gdconf.GetQuestDataById(int32(quest.QuestId))
		if questDataConfig == nil {
			continue
		}
		for _, questCond := range questDataConfig.FailCondList {
			if questCond.Type != cond {
				continue
			}
			switch cond {
			case constant.QUEST_FINISH_COND_TYPE_LUA_NOTIFY:
				// LUA侧通知 复杂参数
				if questCond.ComplexParam != complexParam {
					continue
				}
				dbQuest.FailQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			}
		}
		for _, questCond := range questDataConfig.FinishCondList {
			if questCond.Type != cond {
				continue
			}
			switch cond {
			case constant.QUEST_FINISH_COND_TYPE_FINISH_PLOT:
				ok := matchParamEqual(questCond.Param, param, 1)
				if !ok {
					continue
				}
				dbQuest.ForceFinishQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			case constant.QUEST_FINISH_COND_TYPE_TRIGGER_FIRE:
				// 场景触发器跳了 参数1:触发器id
				ok := matchParamEqual(questCond.Param, param, 1)
				if !ok {
					continue
				}
				dbQuest.ForceFinishQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			case constant.QUEST_FINISH_COND_TYPE_UNLOCK_TRANS_POINT:
				// 解锁传送锚点 参数1:场景id 参数2:传送锚点id
				ok := matchParamEqual(questCond.Param, param, 2)
				if !ok {
					continue
				}
				dbQuest.ForceFinishQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			case constant.QUEST_FINISH_COND_TYPE_COMPLETE_TALK:
				// 与NPC对话 参数1:对话id
				ok := matchParamEqual(questCond.Param, param, 1)
				if !ok {
					continue
				}
				dbQuest.ForceFinishQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			case constant.QUEST_FINISH_COND_TYPE_LUA_NOTIFY:
				// LUA侧通知 复杂参数
				if questCond.ComplexParam != complexParam {
					continue
				}
				dbQuest.ForceFinishQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			case constant.QUEST_FINISH_COND_TYPE_SKILL:
				// TODO 实在不知道客户端要在怎样的情况下 才会发长按10006这个技能 这里先临时改表解决了
				// 是走ability体系计算出来的 操了
				if quest.QuestId == 35303 {
					questCond.Param[0] = 10067
				}
				// 使用技能 参数1:技能id
				ok := matchParamEqual(questCond.Param, param, 1)
				if !ok {
					continue
				}
				dbQuest.ForceFinishQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			case constant.QUEST_FINISH_COND_TYPE_OBTAIN_ITEM:
				// 获得道具 参数1:道具id
				ok := matchParamEqual(questCond.Param, param, 1)
				if !ok {
					continue
				}
				dbQuest.ForceFinishQuest(quest.QuestId)
				updateQuestIdList = append(updateQuestIdList, quest.QuestId)
			case constant.QUEST_FINISH_COND_TYPE_UNLOCK_AREA:
				// 解锁场景区域 参数1:场景id 参数2:场景区域id
			default:
				logger.Error("not support quest cond type: %v, uid: %v", cond, player.PlayerId)
			}
		}
	}
	if len(updateQuestIdList) > 0 {
		questList := make([]*proto.Quest, 0)
		for _, questId := range updateQuestIdList {
			pbQuest := g.PacketQuest(player, questId)
			if pbQuest == nil {
				continue
			}
			questList = append(questList, pbQuest)
		}
		g.SendMsg(cmd.QuestListUpdateNotify, player.PlayerId, player.ClientSeq, &proto.QuestListUpdateNotify{
			QuestList: questList,
		})

		parentQuestList := g.PacketParentQuestList(player, updateQuestIdList)
		if len(parentQuestList) > 0 {
			g.SendMsg(cmd.FinishedParentQuestUpdateNotify, player.PlayerId, player.ClientSeq, &proto.FinishedParentQuestUpdateNotify{
				ParentQuestList: parentQuestList,
			})
		}

		for _, questId := range updateQuestIdList {
			quest := dbQuest.GetQuestById(questId)
			questDataConfig := gdconf.GetQuestDataById(int32(quest.QuestId))
			if questDataConfig == nil {
				continue
			}
			if quest.State == constant.QUEST_STATE_FINISHED {
				g.ExecQuest(player, quest.QuestId, QuestExecTypeFinish)
				if len(questDataConfig.ItemIdList) != 0 {
					for index, itemId := range questDataConfig.ItemIdList {
						questItem := []*ChangeItem{{ItemId: uint32(itemId), ChangeCount: uint32(questDataConfig.ItemCountList[index])}}
						g.AddPlayerItem(player.PlayerId, questItem, proto.ActionReasonType_ACTION_REASON_QUEST_ITEM)
					}
				}
			}
			if quest.State == constant.QUEST_STATE_FAILED {
				g.ExecQuest(player, quest.QuestId, QuestExecTypeFail)
			}
		}
		g.AcceptQuest(player, true)
		g.TriggerOpenState(player.PlayerId)
	}
}

/************************************************** 打包封装 **************************************************/

// PacketQuest 打包一个任务
func (g *Game) PacketQuest(player *model.Player, questId uint32) *proto.Quest {
	dbQuest := player.GetDbQuest()
	questDataConfig := gdconf.GetQuestDataById(int32(questId))
	if questDataConfig == nil {
		logger.Error("get quest data config is nil, questId: %v", questId)
		return nil
	}
	quest := dbQuest.GetQuestById(questId)
	if quest == nil {
		logger.Error("get quest is nil, questId: %v", questId)
		return nil
	}
	pbQuest := &proto.Quest{
		QuestId:            quest.QuestId,
		State:              uint32(quest.State),
		StartTime:          quest.StartTime,
		ParentQuestId:      uint32(questDataConfig.ParentQuestId),
		StartGameTime:      0,
		AcceptTime:         quest.AcceptTime,
		FinishProgressList: quest.FinishProgressList,
	}
	return pbQuest
}

// PacketQuestListNotify 打包任务列表通知
func (g *Game) PacketQuestListNotify(player *model.Player) *proto.QuestListNotify {
	ntf := &proto.QuestListNotify{
		QuestList: make([]*proto.Quest, 0),
	}
	dbQuest := player.GetDbQuest()
	for _, quest := range dbQuest.GetQuestMap() {
		pbQuest := g.PacketQuest(player, quest.QuestId)
		if pbQuest == nil {
			continue
		}
		ntf.QuestList = append(ntf.QuestList, pbQuest)
	}
	return ntf
}

// PacketFinishedParentQuestNotify 打包已完成父任务列表通知
func (g *Game) PacketFinishedParentQuestNotify(player *model.Player) *proto.FinishedParentQuestNotify {
	dbQuest := player.GetDbQuest()
	questIdList := make([]uint32, len(dbQuest.QuestMap))
	for questId := range dbQuest.GetQuestMap() {
		questIdList = append(questIdList, questId)
	}
	ntf := &proto.FinishedParentQuestNotify{
		ParentQuestList: g.PacketParentQuestList(player, questIdList),
	}
	return ntf
}

func (g *Game) PacketParentQuestList(player *model.Player, questIdList []uint32) []*proto.ParentQuest {
	dbQuest := player.GetDbQuest()
	parentQuestIdMap := make(map[int32]bool)
	parentQuestList := make([]*proto.ParentQuest, 0)
	for questId := range questIdList {
		questDataConfig := gdconf.GetQuestDataById(int32(questId))
		if questDataConfig == nil {
			continue
		}
		_, exist := parentQuestIdMap[questDataConfig.ParentQuestId]
		if exist {
			continue
		}
		parentQuestIdMap[questDataConfig.ParentQuestId] = true
		finishedParentQuest := true
		subQuestDataMap := gdconf.GetQuestDataMapByParentQuestId(questDataConfig.ParentQuestId)
		for _, subQuestData := range subQuestDataMap {
			quest := dbQuest.GetQuestById(uint32(subQuestData.QuestId))
			if quest == nil {
				finishedParentQuest = false
				break
			}
			if quest.State != constant.QUEST_STATE_FINISHED {
				finishedParentQuest = false
				break
			}
		}
		if finishedParentQuest {
			childQuestList := make([]*proto.ChildQuest, 0)
			for _, subQuestData := range subQuestDataMap {
				childQuestList = append(childQuestList, &proto.ChildQuest{
					State:   constant.QUEST_STATE_FINISHED,
					QuestId: uint32(subQuestData.QuestId),
				})
			}
			parentQuestList = append(parentQuestList, &proto.ParentQuest{
				ParentQuestId:    uint32(questDataConfig.ParentQuestId),
				ParentQuestState: 1,
				IsFinished:       true,
				ChildQuestList:   childQuestList,
				QuestVar:         make([]int32, 5),
			})
		}
	}
	return parentQuestList
}
