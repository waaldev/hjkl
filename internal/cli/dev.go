package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/game"
	"github.com/waaldev/hjkl/internal/keys"
	"github.com/waaldev/hjkl/internal/runner"
)

func newDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Author tools",
	}
	cmd.AddCommand(newDevVerifyCmd())
	return cmd
}

func newDevVerifyCmd() *cobra.Command {
	var nvim string
	cmd := &cobra.Command{
		Use:   "verify [challenge-id]",
		Short: "Headlessly check that every (or one) challenge solution reaches its target within par",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if nvim == "" {
				nvim = "nvim"
			}
			cat, err := curriculum.Load()
			if err != nil {
				return err
			}
			chs := cat.Challenges
			if len(args) == 1 {
				ch, ok := cat.Challenge(args[0])
				if !ok {
					return failf("unknown challenge %q", args[0])
				}
				chs = []curriculum.Challenge{ch}
			}
			// Every variant is its own drill to verify.
			var all []curriculum.Challenge
			for _, ch := range chs {
				all = append(all, ch)
				for v := range ch.Variants {
					vc := ch.WithVariant(v)
					vc.ID = fmt.Sprintf("%s#%d", ch.ID, v+1)
					all = append(all, vc)
				}
			}
			chs = all
			failed := 0
			for i, ch := range chs {
				fmt.Printf("[%d/%d] %s … ", i+1, len(chs), ch.ID)
				res, err := runner.Verify(context.Background(), ch, nvim)
				if err != nil {
					failed++
					fmt.Printf("FAIL %v\n", err)
					continue
				}
				if ok, msg, err := game.CheckTechnique(res.CmdKeys, ch.Require, ch.Forbid); err != nil || !ok {
					failed++
					fmt.Printf("FAIL technique: %v%s\n", err, msg)
					continue
				}
				if ch.Type != curriculum.TypeNavigate && ch.Target != "" && ch.Target == ch.Start {
					failed++
					fmt.Println("FAIL start equals target")
					continue
				}
				want := keys.Count(ch.Solution)
				if ch.Par > 0 && res.KeyCount > ch.Par {
					failed++
					fmt.Printf("FAIL over par (keys %d par %d)\n", res.KeyCount, ch.Par)
					continue
				}
				fmt.Printf("ok  keys %d par %d (notation %d)\n", res.KeyCount, ch.Par, want)
			}
			if failed > 0 {
				fmt.Fprintf(os.Stderr, "\n%d challenge(s) failed\n", failed)
				os.Exit(1)
			}
			fmt.Printf("\n%d challenge(s) verified\n", len(chs))
			return nil
		},
	}
	cmd.Flags().StringVar(&nvim, "nvim", os.Getenv("HJKL_NVIM"), "nvim binary")
	return cmd
}
