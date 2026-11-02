package main

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBrailleDotPositions(t *testing.T) {
	for _, dot := range []struct {
		x, y int
		want rune
	}{
		{0, 0, '⠁'}, {0, 1, '⠂'}, {0, 2, '⠄'}, {1, 0, '⠈'},
		{1, 1, '⠐'}, {1, 2, '⠠'}, {0, 3, '⡀'}, {1, 3, '⢀'},
	} {
		img := image.NewGray(image.Rect(0, 0, 20, 4))
		img.SetGray(dot.x, dot.y, color.Gray{255})
		c := defaultConfig(t)
		c.width, c.aspect, c.sharpen, c.dither = 10, 0.5, 0, "none"
		art, err := imageToBraille(img, c)
		if err != nil {
			t.Fatal(err)
		}
		want := string(dot.want) + strings.Repeat("⠀", 9) + "\n"
		if art != want {
			t.Errorf("dot (%d,%d): got %q, want %q", dot.x, dot.y, art, want)
		}
	}
}

func TestHighFidelityCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"-auto-contrast", "-sharpen", "0.7", "-w", "1000", "-aspect", "0.1", "-"}
	if err := run(args, bytes.NewReader(encodedImage(t, "png")), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected diagnostics: %s", &stderr)
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 100 {
		t.Fatalf("expected 100 lines, got %d", len(lines))
	}
	for _, line := range lines {
		if utf8.RuneCountInString(line) != 1000 {
			t.Fatal("incorrect column count")
		}
		for _, r := range line {
			if r < 0x2800 || r > 0x28ff {
				t.Fatalf("non-Braille output %U", r)
			}
		}
	}
}

func TestBrailleGridLimit(t *testing.T) {
	c := defaultConfig(t)
	// Count eight samples per character before reading or allocating pixels.
	img := &image.Gray{Rect: image.Rect(0, 0, 10, 1000)}
	if _, err := imageToBraille(img, c); err == nil {
		t.Fatal("oversized Braille sample grid accepted")
	}
}

func TestBrailleRetainsSubcellShapes(t *testing.T) {
	vertical, horizontal := image.NewGray(image.Rect(0, 0, 20, 4)), image.NewGray(image.Rect(0, 0, 20, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 20; x++ {
			if x%2 == 0 {
				vertical.SetGray(x, y, color.Gray{255})
			}
			if y < 2 {
				horizontal.SetGray(x, y, color.Gray{255})
			}
		}
	}
	c := defaultConfig(t)
	c.width, c.aspect, c.sharpen, c.dither = 10, 0.5, 0, "none"
	brailleVertical, err := imageToBraille(vertical, c)
	if err != nil {
		t.Fatal(err)
	}
	brailleHorizontal, err := imageToBraille(horizontal, c)
	if err != nil {
		t.Fatal(err)
	}
	if brailleVertical != strings.Repeat("⡇", 10)+"\n" || brailleHorizontal != strings.Repeat("⠛", 10)+"\n" {
		t.Fatalf("subcell shapes were lost: %q / %q", brailleVertical, brailleHorizontal)
	}
}
