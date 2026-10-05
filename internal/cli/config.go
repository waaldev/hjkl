package cli

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show config paths and values",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			p, _ := config.Path()
			db, _ := config.DBPath()
			fmt.Println("config", p)
			fmt.Println("db    ", db)
			raw, err := toml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Println()
			fmt.Print(string(raw))
			return nil
		},
	}
	return cmd
}
