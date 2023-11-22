package main

import (
	"context"

	"hk4e/gm/app"

	"github.com/spf13/cobra"
)

func GMCmd() *cobra.Command {
	var configFile string
	c := &cobra.Command{
		Use:   "gm",
		Short: "gm server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(context.Background(), configFile)
		},
	}
	c.Flags().StringVar(&configFile, "config", "application.toml", "config file")
	return c
}
