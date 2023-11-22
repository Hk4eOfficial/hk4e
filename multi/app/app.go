package app

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hk4e/common/config"
	"hk4e/common/mq"
	"hk4e/common/rpc"
	"hk4e/gdconf"
	"hk4e/multi/handle"
	"hk4e/node/api"
	"hk4e/pkg/logger"
)

var APPID string
var APPVERSION string

func Run(ctx context.Context, configFile string) error {
	config.InitConfig(configFile)

	// natsrpc client
	discoveryClient, err := rpc.NewDiscoveryClient()
	if err != nil {
		return err
	}

	// 注册到节点服务器
	rsp, err := discoveryClient.RegisterServer(context.TODO(), &api.RegisterServerReq{
		ServerType: api.MULTI,
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
				ServerType: api.MULTI,
				AppId:      APPID,
			})
			if err != nil {
				logger.Error("keepalive error: %v", err)
			}
		}
	}()
	defer func() {
		_, _ = discoveryClient.CancelServer(context.TODO(), &api.CancelServerReq{
			ServerType: api.MULTI,
			AppId:      APPID,
		})
	}()

	logger.InitLogger("multi_" + APPID)
	logger.Warn("multi start, appid: %v", APPID)
	defer func() {
		logger.CloseLogger()
	}()

	gdconf.InitGameDataConfig()

	messageQueue := mq.NewMessageQueue(api.MULTI, APPID, discoveryClient)
	defer messageQueue.Close()

	_ = handle.NewHandle(messageQueue)

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
				logger.Warn("multi exit, appid: %v", APPID)
				return nil
			case syscall.SIGHUP:
			default:
				return nil
			}
		}
	}
}
