package cli

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// Version vars injected by ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var root = &cobra.Command{
	Use:   "hagane",
	Short: "Go→C transpiler",
	Long:  "hagane compiles Go source to readable C11. Write Go, get C out, build with GCC or Clang.",
}

func Execute() {
	if err := fang.Execute(context.Background(), root,
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
	); err != nil {
		os.Exit(1)
	}
}

func init() {
	root.AddCommand(
		newBuildCmd(),
		newEmitCmd(),
		newRunCmd(),
		newCheckCmd(),
		newVersionCmd(),
	)
}
