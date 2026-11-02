package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func encodedImage(t *testing.T, format string) []byte {
	t.Helper()
	if format == "webp" {
		// A 2x2 white lossless WebP, generated once with ImageMagick.
		data, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAUAAAAfQ//73v/+BiOh/AAA=")
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	img := image.NewGray(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.SetGray(x, y, color.Gray{uint8(x * 255 / 19)})
		}
	}
	var buf bytes.Buffer
	var err error
	if format == "jpeg" {
		err = jpeg.Encode(&buf, img, nil)
	} else {
		err = png.Encode(&buf, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCLIFileAndStdin(t *testing.T) {
	for _, format := range []string{"png", "jpeg", "webp"} {
		t.Run(format, func(t *testing.T) {
			data := encodedImage(t, format)
			path := filepath.Join(t.TempDir(), "input."+format)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			var fileArt, stdinArt, stderr bytes.Buffer
			if err := run([]string{"-w", "10", path}, nil, &fileArt, &stderr); err != nil {
				t.Fatal(err)
			}
			if err := run([]string{"-w", "10", "-o", "-", "-"}, bytes.NewReader(data), &stdinArt, &stderr); err != nil {
				t.Fatal(err)
			}
			if fileArt.String() != stdinArt.String() {
				t.Fatal("file and stdin output differ")
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected diagnostics: %s", &stderr)
			}
			art := fileArt.String()
			if !strings.HasSuffix(art, "\n") || strings.HasSuffix(art, "\n\n") {
				t.Fatalf("expected exactly one trailing newline: %q", art)
			}
		})
	}
}

func TestCLIOutputFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "art.txt")
	var stdout, stderr bytes.Buffer
	err := run([]string{"-o", path, "-"}, bytes.NewReader(encodedImage(t, "png")), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	art, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(art) == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), path) {
		t.Fatal("file output or stream routing incorrect")
	}
}

func TestCLIHelpAndVersion(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "--version"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{arg}, nil, &stdout, &stderr); err != nil {
			t.Fatal(err)
		}
		if arg == "--version" {
			if stdout.String() != version+"\n" || stderr.Len() != 0 {
				t.Fatal("incorrect version output")
			}
		} else if !strings.Contains(stderr.String(), "Usage:") || stdout.Len() != 0 {
			t.Fatal("incorrect help output")
		}
	}
}

func TestCLIInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{}, {"a.png", "b.png"}, {"-unknown"}, {"-w", "9", "-"}, {"-w", "1001", "-"},
		{"-c", "NaN", "-"}, {"-g", "+Inf", "-"}, {"-b", "-Inf", "-"},
		{"-c", "0.4", "-"}, {"-g", "0", "-"}, {"-b", "0.6", "-"},
		{"-aspect", "NaN", "-"}, {"-aspect", "0", "-"}, {"-background", "purple", "-"},
		{"-mode", "braille", "-"}, {"-dither", "unknown", "-"}, {"-dither", "auto", "-"},
		{"-sharpen", "NaN", "-"}, {"-sharpen", "-1", "-"}, {"-sharpen", "3", "-"},
		{"-d", "-"}, {"-detailed", "-"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if err := run(args, nil, io.Discard, io.Discard); err == nil {
				t.Fatal("invalid arguments accepted")
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCLIIOErrors(t *testing.T) {
	if err := run([]string{"-"}, strings.NewReader("not an image"), io.Discard, io.Discard); err == nil || !strings.Contains(err.Error(), "decode image") {
		t.Fatalf("bad decode error: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "missing.png")
	if err := run([]string{missing}, nil, io.Discard, io.Discard); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bad open error: %v", err)
	}
	data := encodedImage(t, "png")
	if err := run([]string{"-"}, bytes.NewReader(data), failingWriter{}, io.Discard); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("bad write error: %v", err)
	}
	if err := run([]string{"-o", t.TempDir(), "-"}, bytes.NewReader(data), io.Discard, io.Discard); err == nil {
		t.Fatal("expected output file error")
	}
}
