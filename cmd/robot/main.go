package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"hk4e/robot/app"
)

var (
	config = flag.String("config", "application.toml", "config file")
)

var VERSION = "UNKNOWN"

func main() {
	flag.Parse()
	app.APPVERSION = VERSION
	err := app.Run(context.TODO(), *config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
