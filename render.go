package main

import (
	"errors"
	"image"
	"image/color"
	"math"
	"strings"

	"golang.org/x/image/draw"
)

const maxRenderSamples = 2_000_000

// imageToBraille samples eight dot positions per output character.
func imageToBraille(img image.Image, config Config) (string, error) {
	if err := config.validate(); err != nil {
		return "", err
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return "", errors.New("image has no pixels")
	}
	height := math.Max(1, math.Round(float64(bounds.Dy())/float64(bounds.Dx())*float64(config.width)*config.aspect))
	// Check the sample grid, not UTF-8 byte length, before allocating anything.
	if height > float64(maxRenderSamples/(config.width*8)) {
		return "", errors.New("render grid is too large; reduce -w or -aspect")
	}
	rows := int(height)
	width, heightPixels := config.width*2, rows*4
	resized := image.NewRGBA64(image.Rect(0, 0, width, heightPixels))
	// The cubic kernel retains crisper edges than the previous tent filter.
	// Keep premultiplied alpha while filtering to avoid transparent color halos.
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Src, nil)
	pixels := make([]float64, width*heightPixels)
	for y := 0; y < heightPixels; y++ {
		for x := 0; x < width; x++ {
			pixels[y*width+x] = luminance(resized.RGBA64At(x, y), config.background)
		}
	}
	if config.autoContrast {
		autoLevels(pixels)
	}
	for i, value := range pixels {
		pixels[i] = adjustTone(value, config)
	}
	if config.sharpen > 0 {
		pixels = sharpenPixels(pixels, width, heightPixels, config.sharpen)
	}
	if config.invert {
		for i := range pixels {
			pixels[i] = 1 - pixels[i]
		}
	}
	dots := quantizeDots(pixels, width, heightPixels, config.dither != "none")
	return renderBraille(dots, config.width, rows), nil
}

func renderBraille(dots []uint8, columns, rows int) string {
	// Unicode Braille dot numbering: 1 4 / 2 5 / 3 6 / 7 8.
	bits := [4][2]uint8{{1, 8}, {2, 16}, {4, 32}, {64, 128}}
	width := columns * 2
	var builder strings.Builder
	builder.Grow((columns*3 + 1) * rows)
	for y := 0; y < rows; y++ {
		for x := 0; x < columns; x++ {
			var mask uint8
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					if dots[(y*4+dy)*width+x*2+dx] != 0 {
						mask |= bits[dy][dx]
					}
				}
			}
			builder.WriteRune(0x2800 + rune(mask))
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}

func luminance(c color.Color, background string) float64 {
	r, g, b, a := c.RGBA()
	// RGBA() is premultiplied. Add the uncovered background exactly once.
	if background == "white" {
		r += 65535 - a
		g += 65535 - a
		b += 65535 - a
	}
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535
}

func adjustTone(value float64, config Config) float64 {
	if config.gamma != 1 {
		value = math.Pow(value, 1/config.gamma)
	}
	value += config.brightness
	return clamp((value-0.5)*config.contrast + 0.5)
}

func clamp(value float64) float64 { return math.Max(0, math.Min(1, value)) }
