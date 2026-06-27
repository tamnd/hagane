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

func newBuildCmd() *cobra.Command {
	var (
		cc      string
		cflags  []string
		output  string
		emitDir string
	)

	cmd := &cobra.Command{
		Use:   "build [packages]",
		Short: "Transpile Go to C and compile with gcc/clang",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := "."
			if len(args) > 0 {
				pattern = args[0]
			}

			prog, err := frontend.Load(loadConfig([]string{pattern}))
			if err != nil {
				return fmt.Errorf("load: %w", err)
			}

			dir := emitDir
			if dir == "" {
				tmp, err := os.MkdirTemp("", "hagane-*")
				if err != nil {
					return fmt.Errorf("tmpdir: %w", err)
				}
				defer os.RemoveAll(tmp) //nolint:errcheck
				dir = tmp
			}

			e := emit.New(prog)
			if err := e.EmitAll(dir); err != nil {
				return fmt.Errorf("emit: %w", err)
			}

			// collect emitted .c files
			cFiles, err := filepath.Glob(filepath.Join(dir, "*.c"))
			if err != nil {
				return err
			}

			cc = resolveCC(cc)
			out := output
			if out == "" {
				out = "a.out"
				if prog.MainPkg != nil {
					out = prog.MainPkg.Pkg.Name()
				}
			}

			argv := append([]string{"-O2", "-std=c11", "-o", out}, cFiles...)
			argv = append(argv, cflags...)
			argv = append(argv, "-lm") // for sqrt, pow, etc.
			c := exec.Command(cc, argv...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("compile: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cc, "cc", "", "C compiler to use (default: auto-detect clang > gcc)")
	cmd.Flags().StringSliceVar(&cflags, "cflags", nil, "extra C compiler flags")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output binary name")
	cmd.Flags().StringVar(&emitDir, "emit-dir", "", "directory to write emitted C files (default: temp)")
	return cmd
}

// loadConfig builds a frontend.Config from CLI args.
// If a single directory argument is given, it sets Dir and uses "." as the pattern
// so go/packages resolves modules relative to that directory.
func loadConfig(args []string) *frontend.Config {
	if len(args) == 1 {
		if fi, err := os.Stat(args[0]); err == nil && fi.IsDir() {
			return &frontend.Config{Dir: args[0], Patterns: []string{"."}}
		}
	}
	return &frontend.Config{Patterns: args}
}

func resolveCC(cc string) string {
	if cc != "" {
		return cc
	}
	if _, err := exec.LookPath("clang"); err == nil {
		return "clang"
	}
	return "gcc"
}
