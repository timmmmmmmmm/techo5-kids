// Package cli is the techo5-kids command. TECHO5's image starts its daemon as `techo5 run` and asks it
// for --version, so those two are all it has: installed as /usr/local/bin/techo5 it starts and
// restarts exactly as TECHO5's own does.
package cli

import (
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/HuskerMinion/techo5/echod/internal/android/logd"
	"github.com/HuskerMinion/techo5/echod/internal/kids/boot"
	"github.com/HuskerMinion/techo5/echod/internal/layout"
)

// Execute runs the command line; no arguments is run.
func Execute() {
	root := &cobra.Command{
		Use:          "techo5-kids",
		Short:        "TECHO5 Kids: the Echo Spot as a learning clock for a child's room",
		SilenceUsage: true,
		Version:      layout.VersionString(),
	}
	root.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Run the clock",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			h := logd.NewHandler(layout.LogTag, cmd.ErrOrStderr())
			defer h.Close()
			slog.SetDefault(slog.New(h))
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic", "panic", r, "stack", string(debug.Stack()))
					h.Close()
					panic(r)
				}
			}()
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			signal.Ignore(syscall.SIGHUP)
			return boot.Run(ctx)
		},
	})
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "run")
	}
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
