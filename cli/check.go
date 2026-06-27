package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tamnd/hagane/frontend"
)

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [packages]",
		Short: "Type-check Go packages and report errors without emitting C",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := frontend.Load(&frontend.Config{Patterns: args})
			if err != nil {
				return err
			}
			fmt.Println("ok")
			return nil
		},
	}
	return cmd
}
