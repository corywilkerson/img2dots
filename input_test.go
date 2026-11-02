package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOversizedDimensionsRejectedBeforeDecode(t *testing.T) {
	data := encodedImage(t, "png")
	// Replace the PNG IHDR dimensions and checksum without allocating pixels.
	binary.BigEndian.PutUint32(data[16:20], 100_000)
	binary.BigEndian.PutUint32(data[20:24], 100_000)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	_, err := readImage("-", bytes.NewReader(data))
	if err == nil || !strings.Contains(err.Error(), "64 megapixel") {
		t.Fatalf("size guard failed: %v", err)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestEncodedInputLimit(t *testing.T) {
	_, err := readImage("-", zeroReader{})
	if err == nil || !strings.Contains(err.Error(), "64 MiB") {
		t.Fatalf("encoded size guard failed: %v", err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestInputReadFailure(t *testing.T) {
	_, err := readImage("-", failingReader{})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("read error lost: %v", err)
	}
}

func TestTruncatedImagePreservesExistingOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "art.txt")
	if err := os.WriteFile(path, []byte("previous artwork"), 0600); err != nil {
		t.Fatal(err)
	}
	data := encodedImage(t, "png")[:33] // Valid IHDR with no image data.
	err := run([]string{"-o", path, "-"}, bytes.NewReader(data), io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "decode image") {
		t.Fatalf("truncated image accepted: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "previous artwork" {
		t.Fatal("decode failure changed existing output")
	}
}
