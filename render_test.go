package main

import (
	"image"
	"image/color"
	"io"
	"math"
	"math/bits"
	"strings"
	"testing"
)

func defaultConfig(t *testing.T) Config {
	t.Helper()
	c, err := parseFlags([]string{"image.png"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTransparencyCompositing(t *testing.T) {
	for _, tc := range []struct {
		name       string
		pixel      color.Color
		background string
		want       uint16
	}{
		{"translucent white on white", color.NRGBA{255, 255, 255, 128}, "white", 65535},
		{"transparent on white", color.NRGBA{255, 0, 0, 0}, "white", 65535},
		{"transparent on black", color.NRGBA{255, 0, 0, 0}, "black", 0},
		{"half black on white", color.NRGBA{0, 0, 0, 128}, "white", 32639},
		{"half white on black", color.NRGBA{255, 255, 255, 128}, "black", 32896},
		{"premultiplied white on white", color.RGBA{128, 128, 128, 128}, "white", 65535},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := float64(tc.want) / 65535
			if got := luminance(tc.pixel, tc.background); math.Abs(got-want) > 1e-12 {
				t.Errorf("got %g, want %g", got, want)
			}
		})
	}
}

func TestDownsamplingPreservesFineDetail(t *testing.T) {
	// Point sampling selects only black pixels here. Filtering should preserve
	// the average tone of this fine black/white checkerboard.
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if (x+y)%2 == 1 {
				img.SetGray(x, y, color.Gray{255})
			}
		}
	}
	c := defaultConfig(t)
	c.width = 10
	art, err := imageToBraille(img, c)
	if err != nil {
		t.Fatal(err)
	}
	lit, total := 0, 0
	for _, char := range art {
		if char == '\n' {
			continue
		}
		if char < 0x2800 || char > 0x28ff {
			t.Fatalf("non-Braille output %U", char)
		}
		lit += bits.OnesCount8(uint8(char - 0x2800))
		total += 8
	}
	if average := float64(lit) / float64(total); math.Abs(average-0.5) > 0.05 {
		t.Fatalf("checkerboard tone lost: lit dot fraction %g", average)
	}
}

func TestRenderDimensionsAndPolarity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bounds     image.Rectangle
		invert     bool
		background string
		rows       int
		char       rune
	}{
		{"default", image.Rect(0, 0, 100, 100), false, "white", 5, '⣿'},
		{"inverted", image.Rect(0, 0, 100, 100), true, "white", 5, '⠀'},
		{"black background", image.Rect(0, 0, 100, 100), false, "black", 5, '⠀'},
		{"very wide", image.Rect(0, 0, 1000, 1), false, "white", 1, '⣿'},
		{"offset bounds", image.Rect(10, 20, 110, 120), false, "white", 5, '⣿'},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := defaultConfig(t)
			c.width, c.invert, c.background = 10, tc.invert, tc.background
			art, err := imageToBraille(image.NewNRGBA(tc.bounds), c)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.Repeat(strings.Repeat(string(tc.char), 10)+"\n", tc.rows)
			if art != want {
				t.Errorf("got %q, want %q", art, want)
			}
		})
	}
}

func TestInvalidDimensions(t *testing.T) {
	c := defaultConfig(t)
	if _, err := imageToBraille(image.NewGray(image.Rectangle{}), c); err == nil {
		t.Fatal("empty image accepted")
	}
	// A large rectangle with no allocated pixel storage: validation must happen
	// before the renderer reads or allocates pixels.
	img := &image.Gray{Rect: image.Rect(0, 0, 1, 1_000_000)}
	if _, err := imageToBraille(img, c); err == nil {
		t.Fatal("excessively tall output accepted")
	}
}

func TestToneAdjustments(t *testing.T) {
	c := defaultConfig(t)
	base := adjustTone(128.0/255, c)
	c.gamma = 2
	if got := adjustTone(128.0/255, c); got <= base {
		t.Fatal("gamma > 1 should lighten midtones")
	}
	c.gamma = 1
	c.brightness = 0.5
	c.contrast = 3
	if got := adjustTone(1, c); got != 1 {
		t.Errorf("highlights not clamped: %g", got)
	}
	c.brightness = -0.5
	if got := adjustTone(0, c); got != 0 {
		t.Errorf("shadows not clamped: %g", got)
	}
}

func TestResamplingTransparencyDoesNotCreateHalos(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			// Hidden color in transparent pixels must not contaminate the edge
			// of opaque white when the image is resampled against white.
			pixel := color.NRGBA{255, 0, 0, 0}
			if x < 20 {
				pixel = color.NRGBA{255, 255, 255, 255}
			}
			img.SetNRGBA(x, y, pixel)
		}
	}
	c := defaultConfig(t)
	c.width = 13
	art, err := imageToBraille(img, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, char := range art {
		if char != '\n' && char != '⣿' {
			t.Fatalf("transparency halo at %U", char)
		}
	}
}
