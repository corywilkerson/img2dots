package main

import "math"

// autoLevels uses a histogram so a few bright or dark outliers do not squash
// most of a photograph into a narrow part of the character ramp.
func autoLevels(pixels []float64) {
	const bins = 4096
	var histogram [bins]int
	for _, value := range pixels {
		histogram[int(math.Round(clamp(value)*(bins-1)))]++
	}
	cutoff := len(pixels) / 100
	low, high, count := 0, bins-1, 0
	for low < bins-1 && count+histogram[low] <= cutoff {
		count += histogram[low]
		low++
	}
	count = 0
	for high > low && count+histogram[high] <= cutoff {
		count += histogram[high]
		high--
	}
	black, white := float64(low)/(bins-1), float64(high)/(bins-1)
	// Do not amplify noise or change the tone of an almost-flat image.
	if white-black < 16.0/255 {
		return
	}
	for i, value := range pixels {
		pixels[i] = clamp((value - black) / (white - black))
	}
}

// sharpenPixels adds a small amount of local contrast after resampling. A
// separable 3x3 Gaussian blur defines the unsharp mask; borders are replicated.
func sharpenPixels(pixels []float64, width, height int, amount float64) []float64 {
	blurred := make([]float64, len(pixels))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			row := y * width
			left, right := max(0, x-1), min(width-1, x+1)
			blurred[row+x] = (pixels[row+left] + 2*pixels[row+x] + pixels[row+right]) / 4
		}
	}
	result := make([]float64, len(pixels))
	for y := 0; y < height; y++ {
		above, below := max(0, y-1)*width, min(height-1, y+1)*width
		for x := 0; x < width; x++ {
			i := y*width + x
			blur := (blurred[above+x] + 2*blurred[i] + blurred[below+x]) / 4
			result[i] = clamp(pixels[i] + amount*(pixels[i]-blur))
		}
	}
	return result
}

// quantizeDots converts grayscale samples to off/on dots. Error diffusion
// alternates scan direction to avoid diagonal bias, with only two error rows.
func quantizeDots(pixels []float64, width, height int, dither bool) []uint8 {
	result := make([]uint8, len(pixels))
	if !dither {
		for i, value := range pixels {
			result[i] = uint8(math.Round(clamp(value)))
		}
		return result
	}
	current, next := make([]float64, width+2), make([]float64, width+2)
	for y := 0; y < height; y++ {
		start, end, step := 0, width, 1
		if y%2 == 1 {
			start, end, step = width-1, -1, -1
		}
		for x := start; x != end; x += step {
			i := y*width + x
			value := pixels[i] + current[x+1]
			index := uint8(math.Round(clamp(value)))
			result[i] = index
			residual := value - float64(index)
			current[x+1+step] += residual * 7 / 16
			next[x+1-step] += residual * 3 / 16
			next[x+1] += residual * 5 / 16
			next[x+1+step] += residual / 16
		}
		current, next = next, current
		clear(next)
	}
	return result
}
