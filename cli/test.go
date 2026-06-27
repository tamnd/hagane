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

func newTestCmd() *cobra.Command {
	var (
		cc      string
		cflags  []string
		emitDir string
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "test [packages]",
		Short: "Run Go tests transpiled to C",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			patterns := args
			if len(patterns) == 0 {
				patterns = []string{"."}
			}

			cfg := loadConfigTest(patterns)

			prog, err := frontend.Load(cfg)
			if err != nil {
				return fmt.Errorf("load: %w", err)
			}

			dir := emitDir
			if dir == "" {
				tmp, err := os.MkdirTemp("", "hagane-test-*")
				if err != nil {
					return fmt.Errorf("tmpdir: %w", err)
				}
				defer os.RemoveAll(tmp) //nolint:errcheck
				dir = tmp
			}

			e := emit.New(prog)
			tests, err := e.EmitAllTest(dir)
			if err != nil {
				return fmt.Errorf("emit: %w", err)
			}

			if len(tests) == 0 {
				fmt.Println("no tests found")
				return nil
			}

			// collect emitted .c files
			cFiles, err := filepath.Glob(filepath.Join(dir, "*.c"))
			if err != nil {
				return err
			}

			cc = resolveCC(cc)
			binPath := filepath.Join(dir, "hg_test_runner")

			argv := append([]string{"-O1", "-std=c11", "-o", binPath}, cFiles...)
			argv = append(argv, cflags...)
			argv = append(argv, "-lm")

			if verbose {
				fmt.Fprintf(os.Stderr, "cc: %s %v\n", cc, argv)
			}

			compileCmd := exec.Command(cc, argv...)
			compileCmd.Stdout = os.Stderr
			compileCmd.Stderr = os.Stderr
			if err := compileCmd.Run(); err != nil {
				return fmt.Errorf("compile: %w", err)
			}

			runCmd := exec.Command(binPath)
			runCmd.Stdout = os.Stdout
			runCmd.Stderr = os.Stderr
			if err := runCmd.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				return fmt.Errorf("run: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cc, "cc", "", "C compiler (default: auto-detect clang > gcc)")
	cmd.Flags().StringSliceVar(&cflags, "cflags", nil, "extra C compiler flags")
	cmd.Flags().StringVar(&emitDir, "emit-dir", "", "directory to write emitted C files (default: temp)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show compiler command")
	return cmd
}

func loadConfigTest(patterns []string) *frontend.Config {
	cfg := loadConfig(patterns)
	cfg.Tests = true
	return cfg
}
