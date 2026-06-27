package emit

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed crt/hagane_rt.h
var runtimeHeader []byte

//go:embed crt/hagane_rt.c
var runtimeImpl []byte

func writeRuntimeFiles(outDir string) error {
	if err := os.WriteFile(filepath.Join(outDir, "hagane_rt.h"), runtimeHeader, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "hagane_rt.c"), runtimeImpl, 0644)
}
