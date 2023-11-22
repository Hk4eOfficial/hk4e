package app

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"hk4e/common/config"
	"hk4e/common/mq"
	"hk4e/common/rpc"
	"hk4e/gate/dao"
	"hk4e/gate/net"
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
		ServerType: api.GATE,
		AppVersion: APPVERSION,
		GateServerAddr: &api.GateServerAddr{
			KcpAddr: config.GetConfig().Hk4e.KcpAddr,
			KcpPort: uint32(config.GetConfig().Hk4e.KcpPort),
			MqAddr:  config.GetConfig().Hk4e.GateTcpMqAddr,
			MqPort:  uint32(config.GetConfig().Hk4e.GateTcpMqPort),
		},
		GameVersionList: strings.Split(config.GetConfig().Hk4e.Version, ","),
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
				ServerType: api.GATE,
				AppId:      APPID,
				LoadCount:  uint32(atomic.LoadInt32(&net.CLIENT_CONN_NUM)),
			})
			if err != nil {
				logger.Error("keepalive error: %v", err)
			}
		}
	}()
	defer func() {
		_, _ = discoveryClient.CancelServer(context.TODO(), &api.CancelServerReq{
			ServerType: api.GATE,
			AppId:      APPID,
		})
	}()

	logger.InitLogger("gate_" + APPID)
	logger.Warn("gate start, appid: %v", APPID)
	defer func() {
		logger.CloseLogger()
	}()

	messageQueue := mq.NewMessageQueue(api.GATE, APPID, nil)
	defer messageQueue.Close()

	db, err := dao.NewDao()
	if err != nil {
		return err
	}
	defer db.CloseDao()

	kcpConnManager, err := net.NewKcpConnManager(db, messageQueue, discoveryClient)
	if err != nil {
		return err
	}
	defer kcpConnManager.Close()

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
				logger.Warn("gate exit, appid: %v", APPID)
				return nil
			case syscall.SIGHUP:
			default:
				return nil
			}
		}

	}
}
