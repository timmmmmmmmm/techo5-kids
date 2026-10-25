// Package boot brings TECHO5 Kids up: TECHO5's start-up (internal/boot) cut down to what the kids'
// parts need. The settings are read, every part is put back the way it was left, and the parts run
// under the same supervisor until the process is told to stop.
package boot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/HuskerMinion/techo5/echod/internal/component"
	"github.com/HuskerMinion/techo5/echod/internal/config"
	_ "github.com/HuskerMinion/techo5/echod/internal/kids/all"
	"github.com/HuskerMinion/techo5/echod/internal/layout"
	"github.com/HuskerMinion/techo5/echod/internal/service"
)

// Run brings everything up and stays until ctx is canceled.
func Run(ctx context.Context) error {
	slog.Info("techo5-kids starting", "version", layout.Version, "board", layout.Board, "pid", os.Getpid())
	config.Started(config.Device{Name: name(), Addr: fmt.Sprintf(":%d", layout.Port)})
	if err := config.LoadError(); err != nil {
		slog.Error("reading the saved config failed, continuing with defaults", "err", err)
	}

	component.Default().Restore(config.Get())
	group := service.New()
	component.Default().AddTo(group)
	slog.Info("resident")

	err := group.Run(ctx)
	slog.Info("stopping")
	return err
}

// name is the device's name as TECHO5 recorded it, so it is the same device on the network.
func name() string {
	if b, err := os.ReadFile(layout.NamePath); err == nil {
		if n := strings.TrimSpace(string(b)); n != "" {
			return n
		}
	}
	return layout.DefaultName
}
