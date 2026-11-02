package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// writeOutput completes the file beside its destination before replacing it.
// A failed write or rename leaves the existing destination intact.
func writeOutput(path, art string) error {
	mode := os.FileMode(0644)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("destination is not a regular file: %s", path)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".webp2ascii-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := io.WriteString(file, art); err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
