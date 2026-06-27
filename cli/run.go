package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tamnd/hagane/emit"
	"github.com/tamnd/hagane/frontend"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [packages]",
		Short: "Transpile, compile, and run a Go program",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tmp, err := os.MkdirTemp("", "hagane-run-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)

			prog, err := frontend.Load(loadConfig(args))
			if err != nil {
				return err
			}
			e := emit.New(prog)
			if err := e.EmitAll(tmp); err != nil {
				return err
			}

			bin := filepath.Join(tmp, "a.out")
			cc := resolveCC("")
			cFiles, err := filepath.Glob(filepath.Join(tmp, "*.c"))
			if err != nil {
				return err
			}
			argv := append([]string{"-O2", "-std=c11", "-o", bin}, cFiles...)
			c := exec.Command(cc, argv...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("compile: %w", err)
			}

			exe := exec.Command(bin)
			exe.Stdout = os.Stdout
			exe.Stderr = os.Stderr
			exe.Stdin = os.Stdin
			return exe.Run()
		},
	}
	return cmd
}
