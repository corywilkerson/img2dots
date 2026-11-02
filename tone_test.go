package main

import (
	"math"
	"testing"
)

func TestAutoLevelsRevealsLowContrastDetail(t *testing.T) {
	pixels := make([]float64, 10000)
	for i := range pixels {
		pixels[i] = 0.4 + 0.2*float64(i)/float64(len(pixels)-1)
	}
	// A handful of extreme pixels should not keep the rest of the image gray.
	pixels[0], pixels[len(pixels)-1] = 0, 1
	autoLevels(pixels)
	if pixels[100] > 0.01 || pixels[9900] < 0.99 || math.Abs(pixels[5000]-0.5) > 0.01 {
		t.Fatalf("contrast range not expanded correctly: %g, %g, %g", pixels[100], pixels[5000], pixels[9900])
	}
	for _, value := range pixels {
		if value < 0 || value > 1 {
			t.Fatalf("unclamped value %g", value)
		}
	}
}

func TestAutoLevelsLeavesFlatImagesAlone(t *testing.T) {
	for _, value := range []float64{0, 0.3, 1} {
		pixels := make([]float64, 200)
		for i := range pixels {
			pixels[i] = value
		}
		autoLevels(pixels)
		for _, got := range pixels {
			if got != value {
				t.Fatalf("flat image changed from %g to %g", value, got)
			}
		}
	}
	autoLevels(nil)
}

func TestSharpeningIncreasesEdgeContrast(t *testing.T) {
	pixels := []float64{0.25, 0.25, 0.75, 0.75}
	got := sharpenPixels(pixels, 4, 1, 1)
	if got[1] >= pixels[1] || got[2] <= pixels[2] {
		t.Fatalf("edge contrast not increased: %v", got)
	}
	if got[0] != pixels[0] || got[3] != pixels[3] {
		t.Fatalf("flat regions changed: %v", got)
	}
	// Rotation should produce the same filter response at a vertical edge.
	vertical := sharpenPixels(pixels, 1, 4, 1)
	for i := range got {
		if got[i] != vertical[i] {
			t.Fatalf("directional bias: %v vs %v", got, vertical)
		}
	}
}

func TestSharpeningFlatRegionsAndClamping(t *testing.T) {
	pixels := []float64{0.4, 0.4, 0.4, 0.4}
	for _, got := range sharpenPixels(pixels, 2, 2, 2) {
		if math.Abs(got-0.4) > 1e-12 {
			t.Fatalf("flat tone changed: %g", got)
		}
	}
	for _, got := range sharpenPixels([]float64{0, 1, 0, 1}, 2, 2, 2) {
		if got < 0 || got > 1 || math.IsNaN(got) {
			t.Fatalf("invalid sharpened pixel: %g", got)
		}
	}
}

func TestDitheringPreservesShading(t *testing.T) {
	const width, height = 128, 64
	pixels := make([]float64, width*height)
	for _, tone := range []float64{0.1, 0.3, 0.5, 0.7, 0.9} {
		for i := range pixels {
			pixels[i] = tone
		}
		dots := quantizeDots(pixels, width, height, true)
		total := 0
		for _, dot := range dots {
			total += int(dot)
		}
		average := float64(total) / float64(len(dots))
		if math.Abs(average-tone) > 0.01 {
			t.Errorf("tone %g became %g", tone, average)
		}
	}
}

func TestDitheringPreservesLocalGradients(t *testing.T) {
	const width, height = 128, 64
	pixels := make([]float64, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixels[y*width+x] = float64(x) / float64(width-1)
		}
	}
	dots := quantizeDots(pixels, width, height, true)
	for block := 0; block < width; block += 16 {
		var expected, actual float64
		for y := 0; y < height; y++ {
			for x := block; x < block+16; x++ {
				expected += pixels[y*width+x]
				actual += float64(dots[y*width+x])
			}
		}
		if math.Abs(expected-actual)/(16*height) > 0.02 {
			t.Errorf("local shading changed in block %d", block)
		}
	}
}

func TestQuantizationEndpointsAndNarrowImages(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {1, 10}, {10, 1}, {2, 2}} {
		for _, dither := range []bool{false, true} {
			for _, value := range []float64{0, 1} {
				pixels := make([]float64, size[0]*size[1])
				for i := range pixels {
					pixels[i] = value
				}
				for _, index := range quantizeDots(pixels, size[0], size[1], dither) {
					if index != uint8(value) {
						t.Errorf("endpoint %g changed to %d", value, index)
					}
				}
			}
		}
	}
}
