// Package frontend loads Go source using go/packages and builds go/ssa.
package frontend

import (
	"fmt"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Config holds options for loading Go source.
type Config struct {
	Patterns   []string
	Dir        string // working directory for package loading (default: "")
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

	// Collect ALL packages (root + transitive imports) in post-order so
	// dependencies always appear before the packages that import them.
	var all []*ssa.Package
	seen := make(map[string]bool)
	packages.Visit(pkgs, func(p *packages.Package) bool {
		return true
	}, func(p *packages.Package) {
		if p.Types == nil {
			return
		}
		path := p.Types.Path()
		if seen[path] {
			return
		}
		seen[path] = true
		if sp := prog.Package(p.Types); sp != nil {
			all = append(all, sp)
		}
	})

	// When loading with Tests: true, ssautil.AllPackages also returns test and
	// test-binary packages that packages.Visit may not reach via the import graph.
	// Append any that we haven't already seen. If a package path is already in
	// all but a later ssaPkg with the same path has test functions, replace it
	// so the test-augmented version (which carries internal test functions) wins.
	if cfg.Tests {
		for _, sp := range ssaPkgs {
			if sp == nil {
				continue
			}
			path := sp.Pkg.Path()
			if !seen[path] {
				seen[path] = true
				all = append(all, sp)
				continue
			}
			// path already seen — promote this package if it has Test* members
			hasTests := false
			for name := range sp.Members {
				if strings.HasPrefix(name, "Test") {
					hasTests = true
					break
				}
			}
			if hasTests {
				for i, existing := range all {
					if existing != nil && existing.Pkg.Path() == path {
						all[i] = sp
						break
					}
				}
			}
		}
	}

	var mainPkg *ssa.Package
	for _, p := range all {
		if p != nil && p.Pkg.Name() == "main" && !strings.HasSuffix(p.Pkg.Path(), ".test") {
			mainPkg = p
			break
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
