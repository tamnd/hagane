package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tamnd/hagane/emit"
	"github.com/tamnd/hagane/frontend"
)

func newEmitCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:   "emit [packages]",
		Short: "Emit C source files without compiling",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prog, err := frontend.Load(loadConfig(args))
			if err != nil {
				return err
			}
			e := emit.New(prog)
			if outDir != "" {
				return e.EmitAll(outDir)
			}
			// no output dir: write to stdout
			var buf bytes.Buffer
			if err := e.EmitPkg(prog.MainPkg, &buf); err != nil {
				return err
			}
			_, err = fmt.Fprint(os.Stdout, buf.String())
			return err
		},
	}
	cmd.Flags().StringVarP(&outDir, "out-dir", "d", "", "output directory for C files (default: stdout)")
	return cmd
}
