package game

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"hk4e/common/mq"
	"hk4e/gs/dao"
	"hk4e/gs/model"
	"hk4e/pkg/logger"
	"hk4e/protocol/proto"

	"github.com/vmihailenco/msgpack/v5"
)

// 玩家管理器

// 玩家登录 从db查询出来然后写入redis并异步回调返回玩家对象
// 玩家离线 写入db和redis
// 玩家定时保存 写入db和redis

type UserManager struct {
	db                  *dao.Dao                 // db对象
	playerMap           map[uint32]*model.Player // 内存玩家数据
	saveUserChan        chan *SaveUserData       // 用于主协程发送玩家数据给定时保存协程
	remotePlayerMap     map[uint32]string        // 远程玩家 key:userId value:玩家所在gs的appid
	remotePlayerMapLock sync.RWMutex
}

func NewUserManager(db *dao.Dao) (r *UserManager) {
	r = new(UserManager)
	r.db = db
	r.playerMap = make(map[uint32]*model.Player)
	r.saveUserChan = make(chan *SaveUserData, 1)
	r.remotePlayerMap = make(map[uint32]string)
	go r.saveUserHandle()
	r.syncRemotePlayerMap()
	go r.autoSyncRemotePlayerMap()
	return r
}

// 在线玩家相关操作

// GetUserOnlineState 获取玩家在线状态
func (u *UserManager) GetUserOnlineState(userId uint32) bool {
	player, exist := u.playerMap[userId]
	if !exist {
		return false
	} else {
		return player.Online
	}
}

// GetOnlineUser 获取在线玩家对象
func (u *UserManager) GetOnlineUser(userId uint32) *model.Player {
	player, exist := u.playerMap[userId]
	if !exist {
		return nil
	} else {
		if player.Online {
			return player
		} else {
			return nil
		}
	}
}

// GetAllOnlineUserList 获取全部在线玩家
func (u *UserManager) GetAllOnlineUserList() map[uint32]*model.Player {
	onlinePlayerMap := make(map[uint32]*model.Player)
	for userId, player := range u.playerMap {
		if player.Online == false {
			continue
		}
		onlinePlayerMap[userId] = player
	}
	return onlinePlayerMap
}

// AddUser 向内存玩家数据里添加一个玩家
func (u *UserManager) AddUser(player *model.Player) {
	if player == nil {
		return
	}
	u.playerMap[player.PlayerId] = player
}

// DeleteUser 从内存玩家数据里删除一个玩家
func (u *UserManager) DeleteUser(userId uint32) {
	delete(u.playerMap, userId)
}

type PlayerLoginInfo struct {
	UserId    uint32
	Player    *model.Player
	ClientSeq uint32
	GateAppId string
	Req       *proto.PlayerLoginReq
	Ok        bool
}

// UserLoginLoad 玩家登录数据库异步加载
func (u *UserManager) UserLoginLoad(userId uint32, clientSeq uint32, gateAppId string, req *proto.PlayerLoginReq) {
	_, exist := u.playerMap[userId]
	// 正常登录
	if exist {
		// 每次玩家上线必须从数据库加载最新的档 如果之前存在于内存则删掉
		u.DeleteUser(userId)
	}
	go func() {
		// 加离线玩家数据分布式锁
		ok := u.db.DistLockSync(userId)
		if !ok {
			logger.Error("lock redis offline player data error, uid: %v", userId)
			LOCAL_EVENT_MANAGER.GetLocalEventChan() <- &LocalEvent{
				EventId: UserLoginLoadFromDbFinish,
				Msg: &PlayerLoginInfo{
					UserId:    userId,
					ClientSeq: clientSeq,
					GateAppId: gateAppId,
					Ok:        false,
				},
			}
			return
		}
		player, err := u.LoadUserFromDbSync(userId)
		if err != nil {
			logger.Error("can not load user from db, uid: %v", userId)
			LOCAL_EVENT_MANAGER.GetLocalEventChan() <- &LocalEvent{
				EventId: UserLoginLoadFromDbFinish,
				Msg: &PlayerLoginInfo{
					UserId:    userId,
					ClientSeq: clientSeq,
					GateAppId: gateAppId,
					Ok:        false,
				},
			}
			// 解离线玩家数据分布式锁
			u.db.DistUnlock(userId)
			return
		}
		if player != nil {
			u.SaveUserToRedisSync(player)
			u.ChangeUserDbState(player, model.DbNormal)
			player.ChatMsgMap = u.LoadUserChatMsgFromDbSync(userId)
		}
		LOCAL_EVENT_MANAGER.GetLocalEventChan() <- &LocalEvent{
			EventId: UserLoginLoadFromDbFinish,
			Msg: &PlayerLoginInfo{
				UserId:    userId,
				Player:    player,
				ClientSeq: clientSeq,
				GateAppId: gateAppId,
				Req:       req,
				Ok:        true,
			},
		}
		// 解离线玩家数据分布式锁
		u.db.DistUnlock(userId)
	}()
}

// OnlineUser 玩家上线
func (u *UserManager) OnlineUser(player *model.Player) {
	player.Online = true
	player.OnlineTime = uint32(time.Now().Unix())
	USER_MANAGER.AddUser(player)
	GAME.messageQueue.SendToAll(&mq.NetMsg{
		MsgType: mq.MsgTypeServer,
		EventId: mq.ServerUserOnlineStateChangeNotify,
		ServerMsg: &mq.ServerMsg{
			UserId:   player.PlayerId,
			IsOnline: true,
		},
	})
	atomic.AddInt32(&ONLINE_PLAYER_NUM, 1)
}

type ChangeGsInfo struct {
	IsChangeGs     bool
	JoinHostUserId uint32
}

type PlayerOfflineInfo struct {
	Player       *model.Player
	ChangeGsInfo *ChangeGsInfo
}

// UserOfflineSave 玩家离线数据库保存
func (u *UserManager) UserOfflineSave(player *model.Player, changeGsInfo *ChangeGsInfo) {
	player.Online = false
	player.OfflineTime = uint32(time.Now().Unix())
	player.TotalOnlineTime += uint32(time.Now().Unix()) - player.OnlineTime
	if player.NotSave {
		LOCAL_EVENT_MANAGER.GetLocalEventChan() <- &LocalEvent{
			EventId: UserOfflineSaveToDbFinish,
			Msg: &PlayerOfflineInfo{
				Player:       player,
				ChangeGsInfo: changeGsInfo,
			},
		}
		return
	}
	if player.OfflineClear {
		go u.DeleteUserAllChatMsgToDbSync(player.PlayerId)
		newPlayer := GAME.CreatePlayer(player.PlayerId)
		newPlayer.DbState = player.DbState
		player = newPlayer
	}
	startTime := time.Now().UnixNano()
	playerData, err := msgpack.Marshal(player)
	if err != nil {
		logger.Error("marshal player data error: %v", err)
		return
	}
	endTime := time.Now().UnixNano()
	costTime := endTime - startTime
	logger.Info("offline copy player data cost time: %v ns", costTime)
	go func() {
		playerCopy := new(model.Player)
		err := msgpack.Unmarshal(playerData, playerCopy)
		if err != nil {
			logger.Error("unmarshal player data error: %v", err)
			return
		}
		playerCopy.DbState = player.DbState
		u.SaveUserToDbSync(playerCopy)
		u.SaveUserToRedisSync(playerCopy)
		LOCAL_EVENT_MANAGER.GetLocalEventChan() <- &LocalEvent{
			EventId: UserOfflineSaveToDbFinish,
			Msg: &PlayerOfflineInfo{
				Player:       player,
				ChangeGsInfo: changeGsInfo,
			},
		}
	}()
}

// OfflineUser 玩家离线
func (u *UserManager) OfflineUser(player *model.Player, changeGsInfo *ChangeGsInfo) {
	USER_MANAGER.DeleteUser(player.PlayerId)
	GAME.messageQueue.SendToAll(&mq.NetMsg{
		MsgType: mq.MsgTypeServer,
		EventId: mq.ServerUserOnlineStateChangeNotify,
		ServerMsg: &mq.ServerMsg{
			UserId:   player.PlayerId,
			IsOnline: false,
		},
	})
	atomic.AddInt32(&ONLINE_PLAYER_NUM, -1)
	if changeGsInfo.IsChangeGs {
		gsAppId := USER_MANAGER.GetRemoteUserGsAppId(changeGsInfo.JoinHostUserId)
		GAME.messageQueue.SendToGate(player.GateAppId, &mq.NetMsg{
			MsgType: mq.MsgTypeServer,
			EventId: mq.ServerUserGsChangeNotify,
			ServerMsg: &mq.ServerMsg{
				UserId:          player.PlayerId,
				GameServerAppId: gsAppId,
				JoinHostUserId:  changeGsInfo.JoinHostUserId,
			},
		})
		logger.Info("user change gs notify to gate, uid: %v, gate appid: %v, gs appid: %v, host uid: %v",
			player.PlayerId, player.GateAppId, gsAppId, changeGsInfo.JoinHostUserId)
	}
}

// ChangeUserDbState 玩家存档状态机 主要用于玩家定时保存时进行分类处理
func (u *UserManager) ChangeUserDbState(player *model.Player, state int) {
	if player == nil {
		return
	}
	switch player.DbState {
	case model.DbNone:
		if state == model.DbInsert {
			player.DbState = model.DbInsert
		} else if state == model.DbDelete {
			player.DbState = model.DbDelete
		} else if state == model.DbNormal {
			player.DbState = model.DbNormal
		} else {
			logger.Error("player db state change not allow, before: %v, after: %v", player.DbState, state)
		}
	case model.DbInsert:
		logger.Error("player db state change not allow, before: %v, after: %v", player.DbState, state)
		break
	case model.DbDelete:
		if state == model.DbNormal {
			player.DbState = model.DbNormal
		} else {
			logger.Error("player db state change not allow, before: %v, after: %v", player.DbState, state)
		}
	case model.DbNormal:
		if state == model.DbDelete {
			player.DbState = model.DbDelete
		} else {
			logger.Error("player db state change not allow, before: %v, after: %v", player.DbState, state)
		}
	}
}

// 远程玩家相关操作

func (u *UserManager) autoSyncRemotePlayerMap() {
	ticker := time.NewTicker(time.Second * 60)
	for {
		<-ticker.C
		u.syncRemotePlayerMap()
	}
}

func (u *UserManager) syncRemotePlayerMap() {
	rsp, err := GAME.discoveryClient.GetGlobalGsOnlineMap(context.TODO(), nil)
	if err != nil {
		logger.Error("get global gs online map error: %v", err)
		return
	}
	copyMap := make(map[uint32]string)
	for k, v := range rsp.OnlineMap {
		player, exist := u.playerMap[k]
		if exist && player.Online {
			continue
		}
		copyMap[k] = v
	}
	copyMapLen := len(copyMap)
	u.remotePlayerMapLock.Lock()
	u.remotePlayerMap = copyMap
	u.remotePlayerMapLock.Unlock()
	logger.Info("sync remote player map finish, len: %v", copyMapLen)
}

func (u *UserManager) GetRemoteUserOnlineState(userId uint32) bool {
	u.remotePlayerMapLock.RLock()
	_, exist := u.remotePlayerMap[userId]
	u.remotePlayerMapLock.RUnlock()
	if !exist {
		return false
	} else {
		return true
	}
}

func (u *UserManager) GetRemoteUserGsAppId(userId uint32) string {
	u.remotePlayerMapLock.RLock()
	appId, exist := u.remotePlayerMap[userId]
	u.remotePlayerMapLock.RUnlock()
	if !exist {
		return ""
	} else {
		return appId
	}
}

func (u *UserManager) SetRemoteUserOnlineState(userId uint32, isOnline bool, appId string) {
	u.remotePlayerMapLock.Lock()
	if isOnline {
		u.remotePlayerMap[userId] = appId
	} else {
		delete(u.remotePlayerMap, userId)
		u.DeleteUser(userId)
	}
	u.remotePlayerMapLock.Unlock()
}

func (u *UserManager) GetAllRemoteAiUidList() []uint32 {
	userIdList := make([]uint32, 0)
	u.remotePlayerMapLock.RLock()
	for userId := uint32(AiBaseUid); userId < AiBaseUid+1000; userId++ {
		_, exist := u.remotePlayerMap[userId]
		if !exist {
			continue
		}
		userIdList = append(userIdList, userId)
	}
	u.remotePlayerMapLock.RUnlock()
	return userIdList
}

// GetRemoteOnlineUserList 获取指定数量的远程在线玩家 玩家数据只读禁止修改
func (u *UserManager) GetRemoteOnlineUserList(total int) map[uint32]*model.Player {
	if total > 50 {
		return nil
	}
	onlinePlayerMap := make(map[uint32]*model.Player)
	count := 0
	userIdList := make([]uint32, 0)
	u.remotePlayerMapLock.RLock()
	for userId := range u.remotePlayerMap {
		if userId < PlayerBaseUid || userId > MaxPlayerBaseUid {
			continue
		}
		userIdList = append(userIdList, userId)
		count++
		if count >= total {
			break
		}
	}
	u.remotePlayerMapLock.RUnlock()
	for _, userId := range userIdList {
		player := u.LoadTempOfflineUser(userId, false)
		if player == nil {
			continue
		}
		onlinePlayerMap[player.PlayerId] = player
	}
	return onlinePlayerMap
}

// LoadGlobalPlayer 加载并返回一个全服玩家及其在线状态 玩家数据只读禁止修改
// 参见LoadTempOfflineUser说明
func (u *UserManager) LoadGlobalPlayer(userId uint32) (player *model.Player, online bool, remote bool) {
	online = u.GetUserOnlineState(userId)
	remote = false
	if !online {
		// 本地不在线就看看远程在不在线
		online = u.GetRemoteUserOnlineState(userId)
		if online {
			remote = true
		}
	}
	if online {
		if remote {
			// 远程在线玩家 为了简化实现流程 直接加载数据库临时档
			player = u.LoadTempOfflineUser(userId, false)
		} else {
			// 本地在线玩家
			player = u.GetOnlineUser(userId)
		}
	} else {
		// 全服离线玩家
		player = u.LoadTempOfflineUser(userId, false)
	}
	return player, online, remote
}

// 离线玩家相关操作

// LoadTempOfflineUser 加载临时离线玩家
// 正常情况速度较快可以同步阻塞调用
func (u *UserManager) LoadTempOfflineUser(userId uint32, lock bool) *model.Player {
	if userId < PlayerBaseUid || userId > MaxPlayerBaseUid {
		logger.Error("try to load a not exist uid, uid: %v", userId)
		return nil
	}
	player := u.GetOnlineUser(userId)
	if player != nil {
		logger.Error("not allow get a online player as offline player, uid: %v", userId)
		return nil
	}
	if lock {
		// 加离线玩家数据分布式锁
		ok := u.db.DistLockSync(userId)
		if !ok {
			logger.Error("lock redis offline player data error, uid: %v", userId)
			return nil
		}
	}
	player = u.LoadUserFromRedisSync(userId)
	if player == nil {
		// 玩家可能不存在于redis 尝试从db查询出来然后写入redis
		// 大多数情况下活跃玩家都在redis 所以不会走到下面
		// TODO 防止恶意攻击造成redis缓存穿透
		startTime := time.Now().UnixNano()
		player, _ = u.LoadUserFromDbSync(userId)
		endTime := time.Now().UnixNano()
		costTime := endTime - startTime
		logger.Info("try to load player from db sync in game main loop, cost time: %v ns", costTime)
		if player == nil {
			// 玩家根本就不存在
			logger.Error("try to load a not exist player from db, uid: %v", userId)
			return nil
		}
		u.SaveUserToRedisSync(player)
	}
	u.ChangeUserDbState(player, model.DbDelete)
	u.playerMap[player.PlayerId] = player
	return player
}

// SaveTempOfflineUser 保存临时离线玩家
// 如果调用LoadTempOfflineUser获取了离线玩家数据 则必须在逻辑完成后立即调用此函数回写并解锁
func (u *UserManager) SaveTempOfflineUser(player *model.Player) {
	if player.PlayerId < PlayerBaseUid || player.PlayerId > MaxPlayerBaseUid {
		logger.Error("try to save a not exist uid, uid: %v", player.PlayerId)
		return
	}
	// 主协程同步写入redis
	u.SaveUserToRedisSync(player)
	// 另一个协程异步的写回db
	playerData, err := msgpack.Marshal(player)
	if err != nil {
		logger.Error("marshal player data error: %v", err)
		// 解离线玩家数据分布式锁
		u.db.DistUnlock(player.PlayerId)
		return
	}
	go func() {
		defer func() {
			// 解离线玩家数据分布式锁
			u.db.DistUnlock(player.PlayerId)
		}()
		playerCopy := new(model.Player)
		err := msgpack.Unmarshal(playerData, playerCopy)
		if err != nil {
			logger.Error("unmarshal player data error: %v", err)
			return
		}
		playerCopy.DbState = player.DbState
		u.SaveUserToDbSync(playerCopy)
	}()
}

// db和redis相关操作

func (u *UserManager) GetSaveUserChan() chan *SaveUserData {
	return u.saveUserChan
}

type SaveUserData struct {
	insertPlayerList [][]byte
	updatePlayerList [][]byte
	exitSave         bool
}

func (u *UserManager) saveUserHandle() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		for {
			<-ticker.C
			// 保存玩家数据
			LOCAL_EVENT_MANAGER.GetLocalEventChan() <- &LocalEvent{
				EventId: RunUserCopyAndSave,
			}
		}
	}()
	go func() {
		for {
			saveUserData := <-u.saveUserChan
			insertPlayerList := make([]*model.Player, 0)
			updatePlayerList := make([]*model.Player, 0)
			setPlayerList := make([]*model.Player, 0)
			for _, playerData := range saveUserData.insertPlayerList {
				player := new(model.Player)
				err := msgpack.Unmarshal(playerData, player)
				if err != nil {
					logger.Error("unmarshal player data error: %v", err)
					continue
				}
				insertPlayerList = append(insertPlayerList, player)
				setPlayerList = append(setPlayerList, player)
			}
			for _, playerData := range saveUserData.updatePlayerList {
				player := new(model.Player)
				err := msgpack.Unmarshal(playerData, player)
				if err != nil {
					logger.Error("unmarshal player data error: %v", err)
					continue
				}
				updatePlayerList = append(updatePlayerList, player)
				setPlayerList = append(setPlayerList, player)
			}
			u.SaveUserListToDbSync(insertPlayerList, updatePlayerList)
			u.SaveUserListToRedisSync(setPlayerList)
			if saveUserData.exitSave {
				// 停服落地玩家数据完毕 通知APP主协程关闭程序
				EXIT_SAVE_FIN_CHAN <- true
			}
		}
	}()
}

const (
	UserCopyGoroutineLimit = 4
)

type PlayerLastSaveTimeSortList []*model.Player

func (p PlayerLastSaveTimeSortList) Len() int {
	return len(p)
}

func (p PlayerLastSaveTimeSortList) Less(i, j int) bool {
	return p[i].LastSaveTime < p[j].LastSaveTime
}

func (p PlayerLastSaveTimeSortList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (u *UserManager) UserCopyAndSave(exitSave bool) {
	startTime := time.Now().UnixNano()
	playerList := make(PlayerLastSaveTimeSortList, 0)
	for _, player := range USER_MANAGER.GetAllOnlineUserList() {
		if player.PlayerId < PlayerBaseUid {
			continue
		}
		if player.NotSave {
			continue
		}
		playerList = append(playerList, player)
	}
	sort.Stable(playerList)
	// 拷贝一份数据避免并发访问
	insertPlayerList := make([][]byte, 0)
	updatePlayerList := make([][]byte, 0)
	saveCount := 0
	times := len(playerList) / UserCopyGoroutineLimit
	if times == 0 && len(playerList) > 0 {
		times = 1
	}
	for index := 0; index < times; index++ {
		totalCostTime := time.Now().UnixNano() - startTime
		if totalCostTime > time.Millisecond.Nanoseconds()*10 {
			// 总耗时超过10ms就中止本轮保存
			logger.Info("user copy loop overtime exit, total cost time: %v ns", totalCostTime)
			break
		}
		// 分批次并发序列化玩家数据
		oncePlayerListEndIndex := 0
		if index < times-1 {
			oncePlayerListEndIndex = (index + 1) * UserCopyGoroutineLimit
		} else {
			oncePlayerListEndIndex = len(playerList)
		}
		oncePlayerList := playerList[index*UserCopyGoroutineLimit : oncePlayerListEndIndex]
		var playerDataMapLock sync.Mutex
		playerDataMap := make(map[uint32][]byte)
		var wg sync.WaitGroup
		for _, player := range oncePlayerList {
			wg.Add(1)
			go func(player *model.Player) {
				defer func() {
					wg.Done()
				}()
				playerData, err := msgpack.Marshal(player)
				if err != nil {
					logger.Error("marshal player data error: %v", err)
					return
				}
				playerDataMapLock.Lock()
				playerDataMap[player.PlayerId] = playerData
				playerDataMapLock.Unlock()
			}(player)
		}
		wg.Wait()
		for _, player := range oncePlayerList {
			playerData, exist := playerDataMap[player.PlayerId]
			if !exist {
				continue
			}
			switch player.DbState {
			case model.DbNone:
				break
			case model.DbInsert:
				insertPlayerList = append(insertPlayerList, playerData)
				player.DbState = model.DbNormal
				player.LastSaveTime = uint32(time.Now().UnixMilli())
				saveCount++
			case model.DbDelete:
				USER_MANAGER.DeleteUser(player.PlayerId)
			case model.DbNormal:
				updatePlayerList = append(updatePlayerList, playerData)
				player.LastSaveTime = uint32(time.Now().UnixMilli())
				saveCount++
			}
		}
	}
	saveUserData := &SaveUserData{
		insertPlayerList: insertPlayerList,
		updatePlayerList: updatePlayerList,
		exitSave:         exitSave,
	}
	USER_MANAGER.GetSaveUserChan() <- saveUserData
	endTime := time.Now().UnixNano()
	costTime := endTime - startTime
	logger.Info("run save user copy cost time: %v ns, save user count: %v", costTime, saveCount)
}

func (u *UserManager) LoadUserFromDbSync(userId uint32) (*model.Player, error) {
	player, err := u.db.QueryPlayerById(userId)
	if err != nil {
		logger.Error("query player error: %v", err)
		return nil, err
	}
	return player, nil
}

func (u *UserManager) SaveUserToDbSync(player *model.Player) {
	if player.DbState == model.DbInsert {
		err := u.db.InsertPlayer(player)
		if err != nil {
			logger.Error("insert player error: %v", err)
			return
		}
	} else if player.DbState == model.DbNormal || player.DbState == model.DbDelete {
		err := u.db.UpdatePlayer(player)
		if err != nil {
			logger.Error("update player error: %v", err)
			return
		}
	} else {
		logger.Error("invalid player db state: %v", player.DbState)
	}
}

func (u *UserManager) SaveUserListToDbSync(insertPlayerList []*model.Player, updatePlayerList []*model.Player) {
	err := u.db.InsertPlayerList(insertPlayerList)
	if err != nil {
		logger.Error("insert player list error: %v", err)
		return
	}
	err = u.db.UpdatePlayerList(updatePlayerList)
	if err != nil {
		logger.Error("update player list error: %v", err)
		return
	}
	logger.Info("save user finish, insert user count: %v, update user count: %v", len(insertPlayerList), len(updatePlayerList))
}

func (u *UserManager) LoadUserChatMsgFromDbSync(userId uint32) map[uint32][]*model.ChatMsg {
	chatMsgMap := make(map[uint32][]*model.ChatMsg)
	chatMsgList, err := u.db.QueryChatMsgListByUid(userId)
	if err != nil {
		logger.Error("query chat msg list error: %v", err)
		return chatMsgMap
	}
	for _, chatMsg := range chatMsgList {
		otherUid := uint32(0)
		if chatMsg.Uid == userId {
			otherUid = chatMsg.ToUid
		} else if chatMsg.ToUid == userId {
			otherUid = chatMsg.Uid
		} else {
			continue
		}
		msgList, exist := chatMsgMap[otherUid]
		if !exist {
			msgList = make([]*model.ChatMsg, 0)
		}
		msgList = append(msgList, chatMsg)
		chatMsgMap[otherUid] = msgList
	}
	for otherUid, msgList := range chatMsgMap {
		if len(msgList) > MaxMsgListLen {
			msgList = msgList[len(msgList)-MaxMsgListLen:]
		}
		for index, chatMsg := range msgList {
			chatMsg.Sequence = uint32(index) + 101
		}
		chatMsgMap[otherUid] = msgList
	}
	return chatMsgMap
}

func (u *UserManager) SaveUserChatMsgToDbSync(chatMsg *model.ChatMsg) {
	err := u.db.InsertChatMsg(chatMsg)
	if err != nil {
		logger.Error("insert chat msg error: %v", err)
		return
	}
}

func (u *UserManager) ReadUserChatMsgToDbSync(uid uint32, targetUid uint32) {
	err := u.db.ReadUpdateChatMsgByUid(uid, targetUid)
	if err != nil {
		logger.Error("read chat msg error: %v", err)
		return
	}
}

func (u *UserManager) DeleteUserAllChatMsgToDbSync(uid uint32) {
	err := u.db.DeleteAllUpdateChatMsgByUid(uid)
	if err != nil {
		logger.Error("delete chat msg error: %v", err)
		return
	}
}

func (u *UserManager) LoadUserFromRedisSync(userId uint32) *model.Player {
	player := u.db.GetRedisPlayer(userId)
	return player
}

func (u *UserManager) SaveUserToRedisSync(player *model.Player) {
	u.db.SetRedisPlayer(player)
}

func (u *UserManager) SaveUserListToRedisSync(setPlayerList []*model.Player) {
	u.db.SetRedisPlayerList(setPlayerList)
}
