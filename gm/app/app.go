package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"hk4e/common/config"
	"hk4e/common/mq"
	"hk4e/common/rpc"
	"hk4e/gm/controller"
	"hk4e/node/api"
	"hk4e/pkg/logger"
)

func Run(ctx context.Context, configFile string) error {
	config.InitConfig(configFile)

	logger.InitLogger("gm")
	logger.Warn("gm start")
	defer func() {
		logger.CloseLogger()
	}()

	// natsrpc client
	discoveryClient, err := rpc.NewDiscoveryClient()
	if err != nil {
		return err
	}

	messageQueue := mq.NewMessageQueue(api.GM, "gm", nil)
	defer messageQueue.Close()

	_ = controller.NewController(discoveryClient, messageQueue)

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
				logger.Warn("gm exit")
				return nil
			case syscall.SIGHUP:
			default:
				return nil
			}
		}
	}
}
