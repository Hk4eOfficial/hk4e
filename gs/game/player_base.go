package game

import (
	"time"

	"hk4e/pkg/object"
	"hk4e/protocol/cmd"

	"hk4e/common/constant"
	"hk4e/gdconf"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

/************************************************** 接口请求 **************************************************/

func (g *Game) PlayerSetPauseReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.PlayerSetPauseReq)
	player.Pause = req.IsPaused
	g.SendMsg(cmd.PlayerSetPauseRsp, player.PlayerId, player.ClientSeq, new(proto.PlayerSetPauseRsp))
}

func (g *Game) SetPlayerPropReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SetPlayerPropReq)
	for _, propValue := range req.PropList {
		logger.Debug("player set prop, key: %v, value: %v, uid: %v", propValue.Type, propValue.Val, player.PlayerId)
		player.PropMap[propValue.Type] = uint32(propValue.Val)
	}
	g.SendMsg(cmd.SetPlayerPropRsp, player.PlayerId, player.ClientSeq, new(proto.SetPlayerPropRsp))
}

func (g *Game) SetOpenStateReq(player *model.Player, payloadMsg pb.Message) {
	req := payloadMsg.(*proto.SetOpenStateReq)
	logger.Debug("player set open state, key: %v, value: %v, uid: %v", req.Key, req.Value, player.PlayerId)
	openStateDataConfig := gdconf.GetOpenStateDataById(int32(req.Key))
	if openStateDataConfig == nil {
		logger.Error("get open state data config is nil, key: %v", req.Key)
		return
	}
	if !object.ConvInt64ToBool(int64(openStateDataConfig.AllowClientReq)) {
		g.SendError(cmd.SetOpenStateRsp, player, &proto.SetOpenStateRsp{})
		return
	}
	g.ChangePlayerOpenState(player.PlayerId, req.Key, req.Value)

	g.SendMsg(cmd.SetOpenStateRsp, player.PlayerId, player.ClientSeq, &proto.SetOpenStateRsp{
		Key:   req.Key,
		Value: req.Value,
	})
}

/************************************************** 游戏功能 **************************************************/

// HandlePlayerExpAdd 玩家冒险阅历增加处理
func (g *Game) HandlePlayerExpAdd(userId uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	// 玩家升级
	for {
		playerLevel := player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL]
		// 读取玩家等级配置表
		playerLevelConfig := gdconf.GetPlayerLevelDataByLevel(int32(playerLevel))
		if playerLevelConfig == nil {
			// 获取不到代表已经到达最大等级
			break
		}
		// 确保拥有下一级的配置表
		if gdconf.GetPlayerLevelDataByLevel(int32(playerLevel+1)) == nil {
			// 获取不到代表已经到达最大等级
			break
		}
		// 玩家冒险阅历不足则跳出循环
		if player.PropMap[constant.PLAYER_PROP_PLAYER_EXP] < uint32(playerLevelConfig.Exp) {
			break
		}
		// 玩家增加冒险等阶
		player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL]++
		player.PropMap[constant.PLAYER_PROP_PLAYER_EXP] -= uint32(playerLevelConfig.Exp)
	}
	// 更新玩家属性
	g.SendMsg(cmd.PlayerPropNotify, player.PlayerId, player.ClientSeq, g.PacketPlayerPropNotify(
		player,
		constant.PLAYER_PROP_PLAYER_LEVEL,
		constant.PLAYER_PROP_PLAYER_EXP,
	))
	g.TriggerOpenState(userId)
}

// TriggerOpenState 触发检测功能开放状态更新
func (g *Game) TriggerOpenState(userId uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	updateOpenStateMap := make(map[uint32]uint32)
	for _, openStateDataConfig := range gdconf.GetOpenStateDataMap() {
		if len(openStateDataConfig.OpenStateCondList) == 0 {
			continue
		}
		if player.OpenStateMap[uint32(openStateDataConfig.OpenStateId)] == 1 {
			continue
		}
		finish := true
		for _, openStateCond := range openStateDataConfig.OpenStateCondList {
			switch openStateCond.Type {
			case constant.OPEN_STATE_COND_PLAYER_LEVEL:
				if len(openStateCond.Param) != 1 {
					finish = false
					continue
				}
				if player.PropMap[constant.PLAYER_PROP_PLAYER_LEVEL] < uint32(openStateCond.Param[0]) {
					finish = false
					continue
				}
			case constant.OPEN_STATE_COND_QUEST:
				if len(openStateCond.Param) != 1 {
					finish = false
					continue
				}
				dbQuest := player.GetDbQuest()
				quest := dbQuest.GetQuestById(uint32(openStateCond.Param[0]))
				if quest == nil {
					finish = false
					continue
				}
				if quest.State != constant.QUEST_STATE_FINISHED {
					finish = false
					continue
				}
			case constant.OPEN_STATE_COND_OFFERING_LEVEL:
				finish = false
				continue
			case constant.OPEN_STATE_COND_CITY_REPUTATION_LEVEL:
				finish = false
				continue
			case constant.OPEN_STATE_COND_PARENT_QUEST:
				finish = false
				continue
			}
		}
		if finish {
			logger.Debug("open state change to open, id: %v, uid: %v", openStateDataConfig.OpenStateId, player.PlayerId)
			updateOpenStateMap[uint32(openStateDataConfig.OpenStateId)] = 1
			player.OpenStateMap[uint32(openStateDataConfig.OpenStateId)] = 1
		}
	}
	g.SendMsg(cmd.OpenStateChangeNotify, player.PlayerId, player.ClientSeq, &proto.OpenStateChangeNotify{
		OpenStateMap: updateOpenStateMap,
	})
}

// ChangePlayerOpenState 修改功能开放状态
func (g *Game) ChangePlayerOpenState(userId uint32, key uint32, value uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	player.OpenStateMap[key] = value
	g.SendMsg(cmd.OpenStateChangeNotify, player.PlayerId, player.ClientSeq, &proto.OpenStateChangeNotify{
		OpenStateMap: map[uint32]uint32{key: value},
	})
}

func (g *Game) AddPlayerNameCard(userId uint32, nameCardId uint32) {
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		return
	}
	dbSocial := player.GetDbSocial()
	dbSocial.UnlockNameCard(nameCardId)
}

/************************************************** 打包封装 **************************************************/

func (g *Game) PacketPlayerDataNotify(player *model.Player) *proto.PlayerDataNotify {
	ntf := &proto.PlayerDataNotify{
		NickName:          player.NickName,
		ServerTime:        uint64(time.Now().UnixMilli()),
		IsFirstLoginToday: true,
		RegionId:          1,
		PropMap:           make(map[uint32]*proto.PropValue),
	}
	for k, v := range player.PropMap {
		ntf.PropMap[k] = g.PacketPropValue(k, v)
	}
	return ntf
}

func (g *Game) PacketPlayerPropNotify(player *model.Player, propList ...uint32) *proto.PlayerPropNotify {
	ntf := &proto.PlayerPropNotify{
		PropMap: make(map[uint32]*proto.PropValue),
	}
	if len(propList) == 0 {
		for k, v := range player.PropMap {
			ntf.PropMap[k] = g.PacketPropValue(k, v)
		}
	} else {
		for _, k := range propList {
			v := player.PropMap[k]
			ntf.PropMap[k] = g.PacketPropValue(k, v)
		}
	}
	return ntf
}

func (g *Game) PacketOpenStateUpdateNotify(player *model.Player) *proto.OpenStateUpdateNotify {
	ntf := &proto.OpenStateUpdateNotify{
		OpenStateMap: player.OpenStateMap,
	}
	return ntf
}
