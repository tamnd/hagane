// Package frontend loads Go source using go/packages and builds go/ssa.
package frontend

import (
	"fmt"
	"go/token"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Config holds options for loading Go source.
type Config struct {
	Patterns   []string
	Dir        string   // working directory for package loading (default: "")
	BuildFlags []string
	Tests      bool
	GOOS       string
	GOARCH     string
}

// Program holds the loaded SSA program.
type Program struct {
	SSA     *ssa.Program
	MainPkg *ssa.Package
	AllPkgs []*ssa.Package
	Fset    *token.FileSet
}

// Load loads the Go packages matching cfg.Patterns and builds SSA.
func Load(cfg *Config) (*Program, error) {
	fset := token.NewFileSet()
	pkgCfg := &packages.Config{
		Fset: fset,
		Dir:  cfg.Dir,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedDeps |
			packages.NeedImports,
		Tests:      cfg.Tests,
		BuildFlags: cfg.BuildFlags,
	}
	if cfg.GOOS != "" {
		pkgCfg.Env = append(pkgCfg.Env, "GOOS="+cfg.GOOS)
	}
	if cfg.GOARCH != "" {
		pkgCfg.Env = append(pkgCfg.Env, "GOARCH="+cfg.GOARCH)
	}

	pkgs, err := packages.Load(pkgCfg, cfg.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("packages contain errors")
	}

	// Build SSA. SanityCheckFunctions validates the SSA invariants.
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	prog, ssaPkgs := ssautil.AllPackages(pkgs, mode)
	prog.Build()

	var mainPkg *ssa.Package
	for _, p := range ssaPkgs {
		if p != nil && p.Pkg.Name() == "main" {
			mainPkg = p
			break
		}
	}

	// filter out nil packages (external test packages etc.)
	var all []*ssa.Package
	for _, p := range ssaPkgs {
		if p != nil {
			all = append(all, p)
		}
	}

	return &Program{
		SSA:     prog,
		MainPkg: mainPkg,
		AllPkgs: all,
		Fset:    fset,
	}, nil
}

// IsFmtPrint returns true if fn is fmt.Println, fmt.Printf, or fmt.Print.
func IsFmtPrint(fn *ssa.Function) bool {
	if fn == nil || fn.Package() == nil {
		return false
	}
	path := fn.Package().Pkg.Path()
	if path != "fmt" {
		return false
	}
	switch fn.Name() {
	case "Println", "Printf", "Print", "Fprintln", "Fprintf", "Fprint",
		"Sprintf", "Errorf":
		return true
	}
	return false
}
