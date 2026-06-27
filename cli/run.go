package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/tamnd/hagane/emit"
	"github.com/tamnd/hagane/frontend"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <package> [args...]",
		Short: "Transpile, compile, and run a Go program",
		Args:  cobra.MinimumNArgs(1),
		// Disable flag parsing after the first non-flag arg so that args meant
		// for the target binary are not consumed by cobra.
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			// First arg is the Go package; remaining args go to the binary.
			pkgPattern := args[0]
			binArgs := args[1:]

			tmp, err := os.MkdirTemp("", "hagane-run-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp) //nolint:errcheck

			prog, err := frontend.Load(loadConfig([]string{pkgPattern}))
			if err != nil {
				return err
			}
			e := emit.New(prog)
			if err := e.EmitAll(tmp); err != nil {
				return err
			}

			bin := filepath.Join(tmp, "a.out")
			if runtime.GOOS == "windows" {
				bin += ".exe"
			}
			cc := resolveCC("")
			cFiles, err := filepath.Glob(filepath.Join(tmp, "*.c"))
			if err != nil {
				return err
			}
			argv := append([]string{"-O2", "-std=c11", "-o", bin}, cFiles...)
			argv = append(argv, "-lm")
			c := exec.Command(cc, argv...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("compile: %w", err)
			}

			exe := exec.Command(bin, binArgs...)
			exe.Stdout = os.Stdout
			exe.Stderr = os.Stderr
			exe.Stdin = os.Stdin
			return exe.Run()
		},
	}
	return cmd
}
