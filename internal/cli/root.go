package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.2.0"

func Execute() {
	if err := NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func NewRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hjkl",
		Short:         "From zero to Vim hero",
		Long:          "hjkl is a dojo for Neovim: short drills in a real nvim, plus a coach for the files you already edit.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
	cmd.AddCommand(
		newLearnCmd(),
		newDrillCmd(),
		newDailyCmd(),
		newStatsCmd(),
		newCheatCmd(),
		newCoachCmd(),
		newAICmd(),
		newAskCmd(),
		newConfigCmd(),
		newDevCmd(),
		newSuggestCmd(),
		newVersionCmd(),
	)
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print hjkl version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}
