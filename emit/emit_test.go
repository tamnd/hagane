package emit_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tamnd/hagane/emit"
	"github.com/tamnd/hagane/frontend"
)

// writeTestPkg writes a Go source file to a temp directory with a go.mod and returns the path.
func writeTestPkg(t *testing.T, name, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/"+name+"\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func loadAndEmit(t *testing.T, dir string) string {
	t.Helper()
	prog, err := frontend.Load(&frontend.Config{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	e := emit.New(prog)
	var sb strings.Builder
	if err := e.EmitPkg(prog.MainPkg, &sb); err != nil {
		t.Fatalf("emit: %v", err)
	}
	return sb.String()
}

func TestEmitAdd(t *testing.T) {
	src := `package main
func add(a, b int) int { return a + b }
func main() {}
`
	dir := writeTestPkg(t, "add", src)
	c := loadAndEmit(t, dir)

	if !strings.Contains(c, "hg_add_i64") {
		t.Errorf("expected wrapping add macro in output; got:\n%s", c)
	}
	if !strings.Contains(c, "int64_t") {
		t.Errorf("expected int64_t in output")
	}
}

func TestEmitString(t *testing.T) {
	src := `package main
func greet(name string) string { return "hello " + name }
func main() {}
`
	dir := writeTestPkg(t, "str", src)
	c := loadAndEmit(t, dir)

	if !strings.Contains(c, "hg_string_concat") {
		t.Errorf("expected hg_string_concat for string +; got:\n%s", c)
	}
	if !strings.Contains(c, "hg_string_t") {
		t.Errorf("expected hg_string_t in output")
	}
}

func TestEmitSlice(t *testing.T) {
	src := `package main
func sum(xs []int) int {
	total := 0
	for i := 0; i < len(xs); i++ {
		total += xs[i]
	}
	return total
}
func main() {}
`
	dir := writeTestPkg(t, "slice", src)
	c := loadAndEmit(t, dir)

	if !strings.Contains(c, "hg_slice_int64_t") {
		t.Errorf("expected hg_slice_int64_t; got:\n%s", c)
	}
}

func TestEmitMultiReturn(t *testing.T) {
	src := `package main
func divmod(a, b int) (int, int) { return a / b, a % b }
func main() {}
`
	dir := writeTestPkg(t, "divmod", src)
	c := loadAndEmit(t, dir)

	if !strings.Contains(c, "_ret_t") {
		t.Errorf("expected multi-return struct in output; got:\n%s", c)
	}
}

func TestEmitStruct(t *testing.T) {
	src := `package main
type Point struct { X, Y int }
func newPoint(x, y int) Point { return Point{X: x, Y: y} }
func main() {}
`
	dir := writeTestPkg(t, "struct", src)
	c := loadAndEmit(t, dir)

	if !strings.Contains(c, "Point_t") {
		t.Errorf("expected Point_t in output; got:\n%s", c)
	}
}

// TestRunHello is an integration test that compiles the emitted C and executes it.
// Skipped when no C compiler is available.
func TestRunHello(t *testing.T) {
	cc := ""
	for _, try := range []string{"clang", "gcc", "cc"} {
		if _, err := exec.LookPath(try); err == nil {
			cc = try
			break
		}
	}
	if cc == "" {
		t.Skip("no C compiler found")
	}

	src := `package main
import "fmt"
func main() { fmt.Println(42) }
`
	pkgDir := writeTestPkg(t, "hello", src)
	outDir := t.TempDir()

	prog, err := frontend.Load(&frontend.Config{Dir: pkgDir, Patterns: []string{"."}})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	e := emit.New(prog)
	if err := e.EmitAll(outDir); err != nil {
		t.Fatalf("emit: %v", err)
	}

	cFiles, _ := filepath.Glob(filepath.Join(outDir, "*.c"))
	bin := filepath.Join(outDir, "hello")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	argv := append([]string{"-O0", "-std=c11", "-o", bin}, cFiles...)
	out, err := exec.Command(cc, argv...).CombinedOutput()
	if err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}

	result, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	got := strings.TrimSpace(string(result))
	if got != "42" {
		t.Errorf("expected output '42', got %q", got)
	}
}
