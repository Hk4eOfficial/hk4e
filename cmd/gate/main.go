package main

import (
	"context"
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"

	"hk4e/gate/app"
	"hk4e/pkg/statsviz_serve"
)

var (
	config = flag.String("config", "application.toml", "config file")
)

var VERSION = "UNKNOWN"

func main() {
	flag.Parse()
	go func() {
		_ = statsviz_serve.Serve("0.0.0.0:3456")
	}()
	app.APPVERSION = VERSION
	err := app.Run(context.TODO(), *config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
