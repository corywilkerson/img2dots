package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOutputReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "art.txt")
	if err := os.WriteFile(path, []byte("previous artwork"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeOutput(path, "⣿⠀\n"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "⣿⠀\n" {
		t.Fatalf("incorrect output: %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("existing file permissions changed")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}

func TestOutputRejectsNonFiles(t *testing.T) {
	dir := t.TempDir()
	if err := writeOutput(dir, "art"); err == nil {
		t.Fatal("directory accepted as output file")
	}
	if err := writeOutput(filepath.Join(dir, "missing", "art.txt"), "art"); !os.IsNotExist(err) {
		t.Fatalf("missing directory: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("failed write left temporary files behind")
	}
}

func TestOutputDoesNotFollowSymlinks(t *testing.T) {
	dir := t.TempDir()
	target, link := filepath.Join(dir, "target.txt"), filepath.Join(dir, "link.txt")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := writeOutput(link, "replacement"); err == nil {
		t.Fatal("symlink accepted as output file")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatal("symlink target changed")
	}
}
