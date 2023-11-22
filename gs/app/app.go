package app

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"hk4e/common/config"
	"hk4e/common/mq"
	"hk4e/common/rpc"
	"hk4e/gdconf"
	"hk4e/gs/dao"
	"hk4e/gs/game"
	"hk4e/gs/service"
	"hk4e/node/api"
	"hk4e/pkg/logger"

	"github.com/nats-io/nats.go"
)

var APPID string
var APPVERSION string
var GSID uint32

func Run(ctx context.Context, configFile string) error {
	config.InitConfig(configFile)

	// natsrpc client
	discoveryClient, err := rpc.NewDiscoveryClient()
	if err != nil {
		return err
	}

	// 注册到节点服务器
	rsp, err := discoveryClient.RegisterServer(context.TODO(), &api.RegisterServerReq{
		ServerType: api.GS,
		AppVersion: APPVERSION,
	})
	if err != nil {
		return err
	}
	APPID = rsp.GetAppId()
	go func() {
		ticker := time.NewTicker(time.Second * 15)
		for {
			<-ticker.C
			_, err := discoveryClient.KeepaliveServer(context.TODO(), &api.KeepaliveServerReq{
				ServerType: api.GS,
				AppId:      APPID,
				LoadCount:  uint32(atomic.LoadInt32(&game.ONLINE_PLAYER_NUM)),
			})
			if err != nil {
				logger.Error("keepalive error: %v", err)
			}
		}
	}()
	GSID = rsp.GetGsId()
	defer func() {
		_, _ = discoveryClient.CancelServer(context.TODO(), &api.CancelServerReq{
			ServerType: api.GS,
			AppId:      APPID,
		})
	}()

	logger.InitLogger("gs_" + strconv.Itoa(int(GSID)) + "_" + APPID)
	logger.Warn("gs start, appid: %v, gsid: %v", APPID, GSID)
	defer func() {
		logger.CloseLogger()
	}()

	gdconf.InitGameDataConfig()

	db, err := dao.NewDao()
	if err != nil {
		return err
	}
	defer db.CloseDao()

	messageQueue := mq.NewMessageQueue(api.GS, APPID, discoveryClient)
	defer messageQueue.Close()

	gameCore := game.NewGameCore(discoveryClient, db, messageQueue, GSID, APPID, APPVERSION)
	defer gameCore.Close()

	// natsrpc server
	conn, err := nats.Connect(config.GetConfig().MQ.NatsUrl)
	if err != nil {
		logger.Error("connect nats error: %v", err)
		return err
	}
	defer conn.Close()
	s, err := service.NewService(conn, GSID)
	if err != nil {
		return err
	}
	defer s.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case <-ctx.Done():
			return nil
		case s := <-c:
			logger.Warn("get a signal %s", s.String())
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				logger.Warn("gs exit, appid: %v", APPID)
				return nil
			case syscall.SIGHUP:
			default:
				return nil
			}
		}
	}
}
