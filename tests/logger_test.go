package tests

import (
	"hk4e/common/config"
	"hk4e/pkg/logger"

	"testing"
)

func TestLogger(t *testing.T) {
	config.CONF = &config.Config{Logger: config.Logger{Level: "DEBUG", Mode: "CONSOLE", Track: true}}

	logger.InitLogger("logger_test")
	defer logger.CloseLogger()
	logger.Warn("logger test ...")
	for i := 0; i < 100; i++ {
		go func(x int) { logger.Info("%d", x) }(i)
	}
}
