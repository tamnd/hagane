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

//go:embed crt/hagane_map.h
var mapHeader []byte

//go:embed crt/hagane_map.c
var mapImpl []byte

func writeRuntimeFiles(outDir string) error {
	for name, data := range map[string][]byte{
		"hagane_rt.h":  runtimeHeader,
		"hagane_rt.c":  runtimeImpl,
		"hagane_map.h": mapHeader,
		"hagane_map.c": mapImpl,
	} {
		if err := os.WriteFile(filepath.Join(outDir, name), data, 0644); err != nil {
			return err
		}
	}
	return nil
}
