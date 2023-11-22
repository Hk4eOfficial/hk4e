package main

import (
	"context"

	"hk4e/gs/app"

	"github.com/spf13/cobra"
)

func GSCmd() *cobra.Command {
	var configFile string
	app.APPVERSION = VERSION
	c := &cobra.Command{
		Use:   "gs",
		Short: "game server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(context.Background(), configFile)
		},
	}
	c.Flags().StringVar(&configFile, "config", "application.toml", "config file")
	return c
}
