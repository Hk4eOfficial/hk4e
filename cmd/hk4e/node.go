package main

import (
	"context"

	"hk4e/node/app"

	"github.com/spf13/cobra"
)

func NodeCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "node",
		Short: "node server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(context.Background(), configFile)
		},
	}
	c.Flags().StringVar(&configFile, "config", "application.toml", "config file")
	return c
}
