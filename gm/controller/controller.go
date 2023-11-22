package controller

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"hk4e/common/config"
	"hk4e/common/mq"
	"hk4e/common/rpc"
	"hk4e/pkg/logger"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	gmClientMap           map[uint32]*rpc.GMClient
	gmClientMapLock       sync.RWMutex
	discoveryClient       *rpc.DiscoveryClient
	messageQueue          *mq.MessageQueue
	globalGsOnlineMap     map[uint32]string // 全服玩家在线表
	globalGsOnlineMapLock sync.RWMutex
}

func NewController(discoveryClient *rpc.DiscoveryClient, messageQueue *mq.MessageQueue) (r *Controller) {
	r = new(Controller)
	r.gmClientMap = make(map[uint32]*rpc.GMClient)
	r.discoveryClient = discoveryClient
	r.messageQueue = messageQueue
	go func() {
		for {
			_, ok := <-r.messageQueue.GetNetMsg()
			if !ok {
				return
			}
		}
	}()
	r.globalGsOnlineMap = make(map[uint32]string)
	r.syncGlobalGsOnlineMap()
	go r.autoSyncGlobalGsOnlineMap()
	go r.registerRouter()
	return r
}

func (c *Controller) autoSyncGlobalGsOnlineMap() {
	ticker := time.NewTicker(time.Second * 60)
	for {
		<-ticker.C
		c.syncGlobalGsOnlineMap()
	}
}

func (c *Controller) syncGlobalGsOnlineMap() {
	rsp, err := c.discoveryClient.GetGlobalGsOnlineMap(context.TODO(), nil)
	if err != nil {
		logger.Error("get global gs online map error: %v", err)
		return
	}
	copyMap := make(map[uint32]string)
	for k, v := range rsp.OnlineMap {
		copyMap[k] = v
	}
	copyMapLen := len(copyMap)
	c.globalGsOnlineMapLock.Lock()
	c.globalGsOnlineMap = copyMap
	c.globalGsOnlineMapLock.Unlock()
	logger.Info("sync global gs online map finish, len: %v", copyMapLen)
}

func (c *Controller) authorize() gin.HandlerFunc {
	return func(context *gin.Context) {
		if context.GetHeader("GmAuthKey") == config.GetConfig().Hk4e.GmAuthKey {
			// 验证通过
			context.Next()
			return
		}
		// 验证不通过
		context.Abort()
		context.JSON(http.StatusOK, gin.H{
			"code": "10001",
			"msg":  "没有访问权限",
		})
	}
}

type CommonRsp struct {
	Code int32  `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (c *Controller) registerRouter() {
	if config.GetConfig().Logger.Level == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.Default()
	engine.GET("/server/online/stats", c.serverOnlineStats)
	engine.Use(c.authorize())
	engine.POST("/gm/cmd", c.gmCmd)
	engine.GET("/server/stop/state", c.serverStopState)
	engine.POST("/server/stop/change", c.serverStopChange)
	engine.GET("/server/white/list", c.serverWhiteList)
	engine.POST("/server/white/add", c.serverWhiteAdd)
	engine.POST("/server/white/del", c.serverWhiteDel)
	engine.POST("/server/dispatch/cancel", c.serverDispatchCancel)
	port := config.GetConfig().HttpPort
	addr := ":" + strconv.Itoa(int(port))
	err := engine.Run(addr)
	if err != nil {
		logger.Error("gin run error: %v", err)
	}
}
