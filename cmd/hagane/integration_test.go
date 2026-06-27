package main_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// integrationMilestone controls which milestone's tests run. Override via
// HAGANE_MILESTONE env var. Default is "m0".
var integrationMilestone = func() string {
	if m := os.Getenv("HAGANE_MILESTONE"); m != "" {
		return strings.ToLower(m)
	}
	return "m0"
}()

var milestoneOrder = map[string]int{
	"m0": 0, "m1": 1, "m2": 2, "m3": 3,
	"m4": 4, "m5": 5, "m6": 6, "m7": 7, "m8": 8,
}

var wantPanicRe = regexp.MustCompile(`(?m)^//\s*wantpanic:\s*(.+)$`)
var buildTagRe = regexp.MustCompile(`(?m)^//go:build\s+(.+)$`)

// haganeBin is built once for the entire test run.
var haganeBin = func() string {
	tmp, err := os.MkdirTemp("", "hagane-inttest-bin-*")
	if err != nil {
		panic("MkdirTemp: " + err.Error())
	}
	bin := filepath.Join(tmp, "hagane")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	// Build from the parent of cmd/hagane/ (the module root).
	repoRoot := filepath.Join("..", "..")
	out, err := exec.Command("go", "build", "-o", bin, "./cmd/hagane").CombinedOutput()
	if err != nil {
		// Try with explicit dir
		out, err = exec.Command("go", "build", "-C", repoRoot, "-o", bin, "./cmd/hagane").CombinedOutput()
	}
	if err != nil {
		panic("cannot build hagane binary: " + string(out))
	}
	return bin
}()

func TestIntegration(t *testing.T) {
	// testdata lives at ../../testdata relative to cmd/hagane/
	testdataDir := filepath.Join("..", "..", "testdata")

	entries, err := filepath.Glob(filepath.Join(testdataDir, "*/main.go"))
	if err != nil {
		t.Fatal("glob testdata:", err)
	}
	if len(entries) == 0 {
		t.Skip("no testdata programs found")
	}

	for _, mainGo := range entries {
		dir := filepath.Dir(mainGo)
		name := filepath.Base(dir)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			src, err := os.ReadFile(mainGo)
			if err != nil {
				t.Fatalf("read %s: %v", mainGo, err)
			}

			// Skip programs tagged for a higher milestone.
			if skip, reason := shouldSkip(string(src)); skip {
				t.Skip(reason)
			}

			// Detect wantpanic annotation.
			wantPanic := ""
			if m := wantPanicRe.FindStringSubmatch(string(src)); m != nil {
				wantPanic = strings.TrimSpace(m[1])
			}

			absDir, err := filepath.Abs(dir)
			if err != nil {
				t.Fatalf("abs dir: %v", err)
			}

			refOut, refErr, refCode := runGoRun(t, absDir)
			hagOut, hagErr, hagCode := runHagane(t, absDir)

			if wantPanic != "" {
				if refCode == 0 {
					t.Fatalf("go run exited 0 but wantpanic=%q is set", wantPanic)
				}
				if hagCode == 0 {
					t.Errorf("hagane exited 0; expected non-zero (panic)")
				}
				if !strings.Contains(hagErr, wantPanic) {
					t.Errorf("hagane stderr %q does not contain %q", hagErr, wantPanic)
				}
				return
			}

			if hagCode != refCode {
				t.Errorf("exit code: go run=%d hagane=%d", refCode, hagCode)
			}
			if hagOut != refOut {
				t.Errorf("stdout mismatch:\nwant: %q\n got: %q", refOut, hagOut)
			}
			if refErr != "" && hagErr != refErr {
				t.Errorf("stderr mismatch:\nwant: %q\n got: %q", refErr, hagErr)
			}
		})
	}
}

func shouldSkip(src string) (bool, string) {
	m := buildTagRe.FindStringSubmatch(src)
	if m == nil {
		return false, ""
	}
	constraint := m[1]
	if strings.Contains(constraint, "ignore") {
		return true, "tagged //go:build ignore"
	}
	requiredLevel := 0
	for tag, level := range milestoneOrder {
		if strings.Contains(constraint, "!"+tag) {
			if level+1 > requiredLevel {
				requiredLevel = level + 1
			}
		}
	}
	currentLevel := milestoneOrder[integrationMilestone]
	if requiredLevel > currentLevel {
		return true, "requires milestone m" + strconv.Itoa(requiredLevel) + ", running " + integrationMilestone
	}
	return false, ""
}

func runGoRun(t *testing.T, dir string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errBuf bytes.Buffer
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return out.String(), errBuf.String(), ee.ExitCode()
		}
		t.Fatalf("go run in %s: %v", dir, err)
	}
	return out.String(), errBuf.String(), 0
}

func runHagane(t *testing.T, dir string) (stdout, stderr string, code int) {
	t.Helper()

	tmp, err := os.MkdirTemp("", "hagane-emit-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp) //nolint:errcheck

	// Step 1: emit C files.
	emitOut, emitErr := exec.Command(haganeBin, "emit", "--out-dir", tmp, dir).CombinedOutput()
	if emitErr != nil {
		t.Fatalf("hagane emit failed (%v):\n%s", emitErr, emitOut)
	}
	if strings.Contains(string(emitOut), "ERROR") {
		t.Fatalf("hagane emit reported error:\n%s", emitOut)
	}

	cFiles, err := filepath.Glob(filepath.Join(tmp, "*.c"))
	if err != nil || len(cFiles) == 0 {
		t.Fatalf("no .c files in %s (emit output: %s)", tmp, emitOut)
	}

	// Step 2: compile with cc.
	cc := resolveCC()
	binary := filepath.Join(tmp, "testprog")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	gccArgs := append([]string{"-O1", "-std=c11", "-o", binary}, cFiles...)
	if runtime.GOOS == "linux" {
		gccArgs = append(gccArgs, "-lm")
	}
	gccOut, err := exec.Command(cc, gccArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s compile failed:\n%s", cc, gccOut)
	}

	// Step 3: run the binary.
	var outBuf, errBuf2 bytes.Buffer
	runCmd := exec.Command(binary)
	runCmd.Stdout = &outBuf
	runCmd.Stderr = &errBuf2
	runErr := runCmd.Run()
	code = 0
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			code = ee.ExitCode()
		}
	}
	return outBuf.String(), errBuf2.String(), code
}

func resolveCC() string {
	if _, err := exec.LookPath("clang"); err == nil {
		return "clang"
	}
	return "gcc"
}
