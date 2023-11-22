package game

import (
	"hk4e/common/mq"
	"hk4e/gate/kcp"
	"hk4e/gs/model"
	"hk4e/node/api"
	"hk4e/pkg/logger"
	"hk4e/protocol/cmd"

	pb "google.golang.org/protobuf/proto"
)

// 接口路由管理器

func (r *RouteManager) initRoute() {
	r.handlerFuncRouteMap = map[uint16]HandlerFunc{
		cmd.PingReq:                           GAME.PingReq,
		cmd.SetPlayerBornDataReq:              GAME.SetPlayerBornDataReq,
		cmd.QueryPathReq:                      GAME.QueryPathReq,
		cmd.UnionCmdNotify:                    GAME.UnionCmdNotify,
		cmd.MassiveEntityElementOpBatchNotify: GAME.MassiveEntityElementOpBatchNotify,
		cmd.ToTheMoonEnterSceneReq:            GAME.ToTheMoonEnterSceneReq,
		cmd.PlayerSetPauseReq:                 GAME.PlayerSetPauseReq,
		cmd.EnterSceneReadyReq:                GAME.EnterSceneReadyReq,
		cmd.PathfindingEnterSceneReq:          GAME.PathfindingEnterSceneReq,
		cmd.GetScenePointReq:                  GAME.GetScenePointReq,
		cmd.GetSceneAreaReq:                   GAME.GetSceneAreaReq,
		cmd.SceneInitFinishReq:                GAME.SceneInitFinishReq,
		cmd.EnterSceneDoneReq:                 GAME.EnterSceneDoneReq,
		cmd.EnterWorldAreaReq:                 GAME.EnterWorldAreaReq,
		cmd.PostEnterSceneReq:                 GAME.PostEnterSceneReq,
		cmd.TowerAllDataReq:                   GAME.TowerAllDataReq,
		cmd.SceneTransToPointReq:              GAME.SceneTransToPointReq,
		cmd.UnlockTransPointReq:               GAME.UnlockTransPointReq,
		cmd.MarkMapReq:                        GAME.MarkMapReq,
		cmd.ChangeAvatarReq:                   GAME.ChangeAvatarReq,
		cmd.SetUpAvatarTeamReq:                GAME.SetUpAvatarTeamReq,
		cmd.ChooseCurAvatarTeamReq:            GAME.ChooseCurAvatarTeamReq,
		cmd.GetGachaInfoReq:                   GAME.GetGachaInfoReq,
		cmd.DoGachaReq:                        GAME.DoGachaReq,
		cmd.CombatInvocationsNotify:           GAME.CombatInvocationsNotify,
		cmd.AbilityInvocationsNotify:          GAME.AbilityInvocationsNotify,
		cmd.ClientAbilityInitFinishNotify:     GAME.ClientAbilityInitFinishNotify,
		cmd.EvtDoSkillSuccNotify:              GAME.EvtDoSkillSuccNotify,
		cmd.ClientAbilityChangeNotify:         GAME.ClientAbilityChangeNotify,
		cmd.EntityAiSyncNotify:                GAME.EntityAiSyncNotify,
		cmd.WearEquipReq:                      GAME.WearEquipReq,
		cmd.ChangeGameTimeReq:                 GAME.ChangeGameTimeReq,
		cmd.GetPlayerSocialDetailReq:          GAME.GetPlayerSocialDetailReq,
		cmd.SetPlayerBirthdayReq:              GAME.SetPlayerBirthdayReq,
		cmd.SetNameCardReq:                    GAME.SetNameCardReq,
		cmd.SetPlayerSignatureReq:             GAME.SetPlayerSignatureReq,
		cmd.SetPlayerNameReq:                  GAME.SetPlayerNameReq,
		cmd.SetPlayerHeadImageReq:             GAME.SetPlayerHeadImageReq,
		cmd.GetAllUnlockNameCardReq:           GAME.GetAllUnlockNameCardReq,
		cmd.GetPlayerFriendListReq:            GAME.GetPlayerFriendListReq,
		cmd.GetPlayerAskFriendListReq:         GAME.GetPlayerAskFriendListReq,
		cmd.AskAddFriendReq:                   GAME.AskAddFriendReq,
		cmd.DealAddFriendReq:                  GAME.DealAddFriendReq,
		cmd.GetOnlinePlayerListReq:            GAME.GetOnlinePlayerListReq,
		cmd.PlayerApplyEnterMpReq:             GAME.PlayerApplyEnterMpReq,
		cmd.PlayerApplyEnterMpResultReq:       GAME.PlayerApplyEnterMpResultReq,
		cmd.PlayerGetForceQuitBanInfoReq:      GAME.PlayerGetForceQuitBanInfoReq,
		cmd.GetShopmallDataReq:                GAME.GetShopmallDataReq,
		cmd.GetShopReq:                        GAME.GetShopReq,
		cmd.BuyGoodsReq:                       GAME.BuyGoodsReq,
		cmd.McoinExchangeHcoinReq:             GAME.McoinExchangeHcoinReq,
		cmd.AvatarChangeCostumeReq:            GAME.AvatarChangeCostumeReq,
		cmd.AvatarWearFlycloakReq:             GAME.AvatarWearFlycloakReq,
		cmd.PullRecentChatReq:                 GAME.PullRecentChatReq,
		cmd.PullPrivateChatReq:                GAME.PullPrivateChatReq,
		cmd.PrivateChatReq:                    GAME.PrivateChatReq,
		cmd.ReadPrivateChatReq:                GAME.ReadPrivateChatReq,
		cmd.PlayerChatReq:                     GAME.PlayerChatReq,
		cmd.BackMyWorldReq:                    GAME.BackMyWorldReq,
		cmd.ChangeWorldToSingleModeReq:        GAME.ChangeWorldToSingleModeReq,
		cmd.SceneKickPlayerReq:                GAME.SceneKickPlayerReq,
		cmd.ChangeMpTeamAvatarReq:             GAME.ChangeMpTeamAvatarReq,
		cmd.SceneAvatarStaminaStepReq:         GAME.SceneAvatarStaminaStepReq,
		cmd.JoinPlayerSceneReq:                GAME.JoinPlayerSceneReq,
		cmd.EvtAvatarEnterFocusNotify:         GAME.EvtAvatarEnterFocusNotify,
		cmd.EvtAvatarUpdateFocusNotify:        GAME.EvtAvatarUpdateFocusNotify,
		cmd.EvtAvatarExitFocusNotify:          GAME.EvtAvatarExitFocusNotify,
		cmd.EvtEntityRenderersChangedNotify:   GAME.EvtEntityRenderersChangedNotify,
		cmd.EvtBulletDeactiveNotify:           GAME.EvtBulletDeactiveNotify,
		cmd.EvtBulletHitNotify:                GAME.EvtBulletHitNotify,
		cmd.EvtBulletMoveNotify:               GAME.EvtBulletMoveNotify,
		cmd.EvtCreateGadgetNotify:             GAME.EvtCreateGadgetNotify,
		cmd.EvtDestroyGadgetNotify:            GAME.EvtDestroyGadgetNotify,
		cmd.CreateVehicleReq:                  GAME.CreateVehicleReq,
		cmd.VehicleInteractReq:                GAME.VehicleInteractReq,
		cmd.SceneEntityDrownReq:               GAME.SceneEntityDrownReq,
		cmd.GetOnlinePlayerInfoReq:            GAME.GetOnlinePlayerInfoReq,
		cmd.GCGAskDuelReq:                     GAME.GCGAskDuelReq,
		cmd.GCGInitFinishReq:                  GAME.GCGInitFinishReq,
		cmd.GCGOperationReq:                   GAME.GCGOperationReq,
		cmd.ObstacleModifyNotify:              GAME.ObstacleModifyNotify,
		cmd.AvatarUpgradeReq:                  GAME.AvatarUpgradeReq,
		cmd.AvatarPromoteReq:                  GAME.AvatarPromoteReq,
		cmd.CalcWeaponUpgradeReturnItemsReq:   GAME.CalcWeaponUpgradeReturnItemsReq,
		cmd.WeaponUpgradeReq:                  GAME.WeaponUpgradeReq,
		cmd.WeaponPromoteReq:                  GAME.WeaponPromoteReq,
		cmd.WeaponAwakenReq:                   GAME.WeaponAwakenReq,
		cmd.AvatarPromoteGetRewardReq:         GAME.AvatarPromoteGetRewardReq,
		cmd.SetEquipLockStateReq:              GAME.SetEquipLockStateReq,
		cmd.TakeoffEquipReq:                   GAME.TakeoffEquipReq,
		cmd.AddQuestContentProgressReq:        GAME.AddQuestContentProgressReq,
		cmd.NpcTalkReq:                        GAME.NpcTalkReq,
		cmd.EvtAiSyncSkillCdNotify:            GAME.EvtAiSyncSkillCdNotify,
		cmd.EvtAiSyncCombatThreatInfoNotify:   GAME.EvtAiSyncCombatThreatInfoNotify,
		cmd.EntityConfigHashNotify:            GAME.EntityConfigHashNotify,
		cmd.MonsterAIConfigHashNotify:         GAME.MonsterAIConfigHashNotify,
		cmd.DungeonEntryInfoReq:               GAME.DungeonEntryInfoReq,
		cmd.PlayerEnterDungeonReq:             GAME.PlayerEnterDungeonReq,
		cmd.PlayerQuitDungeonReq:              GAME.PlayerQuitDungeonReq,
		cmd.GadgetInteractReq:                 GAME.GadgetInteractReq,
		cmd.GmTalkReq:                         GAME.GmTalkReq,
		cmd.SetEntityClientDataNotify:         GAME.SetEntityClientDataNotify,
		cmd.EntityForceSyncReq:                GAME.EntityForceSyncReq,
		cmd.AvatarDieAnimationEndReq:          GAME.AvatarDieAnimationEndReq,
		cmd.WorldPlayerReviveReq:              GAME.WorldPlayerReviveReq,
		cmd.UseItemReq:                        GAME.UseItemReq,
		cmd.EnterTransPointRegionNotify:       GAME.EnterTransPointRegionNotify,
		cmd.ExitTransPointRegionNotify:        GAME.ExitTransPointRegionNotify,
		cmd.GetPlayerBlacklistReq:             GAME.GetPlayerBlacklistReq,
		cmd.GetChatEmojiCollectionReq:         GAME.GetChatEmojiCollectionReq,
		cmd.SetPlayerPropReq:                  GAME.SetPlayerPropReq,
		cmd.SetOpenStateReq:                   GAME.SetOpenStateReq,
		cmd.PlayerStartMatchReq:               GAME.PlayerStartMatchReq,
		cmd.PlayerCancelMatchReq:              GAME.PlayerCancelMatchReq,
		cmd.PlayerConfirmMatchReq:             GAME.PlayerConfirmMatchReq,
		cmd.QuestCreateEntityReq:              GAME.QuestCreateEntityReq,
		cmd.QuestDestroyEntityReq:             GAME.QuestDestroyEntityReq,
		cmd.QuestDestroyNpcReq:                GAME.QuestDestroyNpcReq,
		cmd.AvatarSkillUpgradeReq:             GAME.AvatarSkillUpgradeReq,
		cmd.UnlockAvatarTalentReq:             GAME.UnlockAvatarTalentReq,
		cmd.ReliquaryUpgradeReq:               GAME.ReliquaryUpgradeReq,
		cmd.ReliquaryPromoteReq:               GAME.ReliquaryPromoteReq,
	}
}

type HandlerFunc func(player *model.Player, payloadMsg pb.Message)

type RouteManager struct {
	// k:cmdId v:HandlerFunc
	handlerFuncRouteMap map[uint16]HandlerFunc
}

func NewRouteManager() (r *RouteManager) {
	r = new(RouteManager)
	r.initRoute()
	return r
}

func (r *RouteManager) doRoute(cmdId uint16, userId uint32, clientSeq uint32, payloadMsg pb.Message) {
	handlerFunc, ok := r.handlerFuncRouteMap[cmdId]
	if !ok {
		logger.Error("no route for msg, cmdId: %v", cmdId)
		return
	}
	player := USER_MANAGER.GetOnlineUser(userId)
	if player == nil {
		logger.Error("player is nil, uid: %v", userId)
		GAME.KickPlayer(userId, kcp.EnetNotFoundSession)
		return
	}
	if !player.Online {
		logger.Error("player not online, uid: %v", userId)
		return
	}
	if player.NetFreeze {
		return
	}
	player.ClientSeq = clientSeq
	SELF = player
	handlerFunc(player, payloadMsg)
	SELF = nil
}

func (r *RouteManager) RouteHandle(netMsg *mq.NetMsg) {
	switch netMsg.MsgType {
	case mq.MsgTypeGame:
		if netMsg.OriginServerType != api.GATE {
			return
		}
		gameMsg := netMsg.GameMsg
		switch netMsg.EventId {
		case mq.NormalMsg:
			if gameMsg.CmdId == cmd.PlayerLoginReq {
				GAME.PlayerLoginReq(gameMsg.UserId, gameMsg.ClientSeq, netMsg.OriginServerAppId, gameMsg.PayloadMessage)
				return
			}
			r.doRoute(gameMsg.CmdId, gameMsg.UserId, gameMsg.ClientSeq, gameMsg.PayloadMessage)
		}
	case mq.MsgTypeConnCtrl:
		if netMsg.OriginServerType != api.GATE {
			return
		}
		connCtrlMsg := netMsg.ConnCtrlMsg
		switch netMsg.EventId {
		case mq.ClientRttNotify:
			GAME.ClientRttNotify(connCtrlMsg.UserId, connCtrlMsg.ClientRtt)
		case mq.UserOfflineNotify:
			GAME.OnOffline(connCtrlMsg.UserId, &ChangeGsInfo{
				IsChangeGs: false,
			})
		}
	case mq.MsgTypeServer:
		serverMsg := netMsg.ServerMsg
		switch netMsg.EventId {
		case mq.ServerUserOnlineStateChangeNotify:
			logger.Debug("remote user online state change, uid: %v, online: %v", serverMsg.UserId, serverMsg.IsOnline)
			USER_MANAGER.SetRemoteUserOnlineState(serverMsg.UserId, serverMsg.IsOnline, netMsg.OriginServerAppId)
		case mq.ServerAppidBindNotify:
			GAME.ServerAppidBindNotify(serverMsg.UserId, serverMsg.MultiServerAppId)
		case mq.ServerPlayerMpReq:
			GAME.ServerPlayerMpReq(serverMsg.PlayerMpInfo, netMsg.OriginServerAppId)
		case mq.ServerPlayerMpRsp:
			GAME.ServerPlayerMpRsp(serverMsg.PlayerMpInfo)
		case mq.ServerChatMsgNotify:
			GAME.ServerChatMsgNotify(serverMsg.ChatMsgInfo)
		case mq.ServerAddFriendNotify:
			GAME.ServerAddFriendNotify(serverMsg.AddFriendInfo)
		case mq.ServerStopNotify:
			GAME.ServerStopNotify()
		case mq.ServerDispatchCancelNotify:
			GAME.ServerDispatchCancelNotify(serverMsg.AppVersion)
		case mq.ServerGmCmdNotify:
			commandTextInput := COMMAND_MANAGER.GetCommandMessageInput()
			commandTextInput <- &CommandMessage{
				GMType:     SystemFuncGM,
				FuncName:   serverMsg.GmCmdFuncName,
				ParamList:  serverMsg.GmCmdParamList,
				ResultChan: nil,
			}
		}
	}
}
