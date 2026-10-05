package cli

import (
	"fmt"

	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/tui"
)

func runTUI() error {
	a, err := app.Open()
	if err != nil {
		return err
	}
	defer a.Close()
	return tui.Run(a)
}

func withApp(fn func(*app.App) error) error {
	a, err := app.Open()
	if err != nil {
		return err
	}
	defer a.Close()
	return fn(a)
}

func failf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
