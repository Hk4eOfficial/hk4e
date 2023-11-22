package game

import (
	"time"

	"hk4e/gdconf"
	"hk4e/pkg/logger"
)

// 本地事件队列管理器

const (
	UserLoginLoadFromDbFinish  = iota // 玩家登录从数据库加载完成回调
	UserOfflineSaveToDbFinish         // 玩家离线保存完成
	RunUserCopyAndSave                // 执行一次在线玩家内存数据复制到数据库写入协程
	ExitRunUserCopyAndSave            // 停服时执行全部玩家保存操作
	ReloadGameDataConfig              // 执行热更表
	ReloadGameDataConfigFinish        // 热更表完成
)

type LocalEvent struct {
	EventId int
	Msg     any
}

type LocalEventManager struct {
	localEventChan chan *LocalEvent
}

func NewLocalEventManager() (r *LocalEventManager) {
	r = new(LocalEventManager)
	r.localEventChan = make(chan *LocalEvent, 1000)
	return r
}

func (l *LocalEventManager) GetLocalEventChan() chan *LocalEvent {
	return l.localEventChan
}

func (l *LocalEventManager) LocalEventHandle(localEvent *LocalEvent) {
	switch localEvent.EventId {
	case UserLoginLoadFromDbFinish:
		playerLoginInfo := localEvent.Msg.(*PlayerLoginInfo)
		GAME.OnLogin(playerLoginInfo.UserId, playerLoginInfo.ClientSeq, playerLoginInfo.GateAppId, playerLoginInfo.Player, playerLoginInfo.Req, playerLoginInfo.Ok)
	case UserOfflineSaveToDbFinish:
		playerOfflineInfo := localEvent.Msg.(*PlayerOfflineInfo)
		USER_MANAGER.OfflineUser(playerOfflineInfo.Player, playerOfflineInfo.ChangeGsInfo)
	case RunUserCopyAndSave:
		USER_MANAGER.UserCopyAndSave(false)
	case ExitRunUserCopyAndSave:
		USER_MANAGER.UserCopyAndSave(true)
		// 在此阻塞掉主协程 不再进行任何消息和任务的处理
		logger.Warn("game main loop block")
		select {}
	case ReloadGameDataConfig:
		reloadSceneLua := localEvent.Msg.(bool)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("reload game data config error: %v", err)
				}
			}()
			gdconf.ReloadGameDataConfig(reloadSceneLua)
			LOCAL_EVENT_MANAGER.localEventChan <- &LocalEvent{
				EventId: ReloadGameDataConfigFinish,
			}
		}()
	case ReloadGameDataConfigFinish:
		gdconf.ReplaceGameDataConfig()
		startTime := time.Now().UnixNano()
		WORLD_MANAGER.LoadSceneBlockAoiMap()
		endTime := time.Now().UnixNano()
		costTime := endTime - startTime
		logger.Info("run [LoadSceneBlockAoiMap], cost time: %v ns", costTime)
	}
}
