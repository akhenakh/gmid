package config

import (
	"io"
	"os"
	"path/filepath"
)

// SeedDemo copies the bundled demo songs into the library once.
func SeedDemo(assets, library string) {
	if assets == "" || library == "" {
		return
	}
	src := filepath.Join(assets, "demo")
	entries, err := os.ReadDir(src)
	if err != nil {
		return
	}
	_ = os.MkdirAll(library, 0o755)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".mid" && ext != ".midi" && ext != ".kar" {
			continue
		}
		dst := filepath.Join(library, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		copyFile(filepath.Join(src, e.Name()), dst)
	}
}

func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	_, _ = io.Copy(out, in)
	_ = out.Close()
}
