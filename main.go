package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"

	_ "golang.org/x/image/webp"
)

// Set by build.sh for release binaries.
var version = "dev"

const maxWidth = 1000

const (
	maxInputBytes  = 64 << 20
	maxInputPixels = 64_000_000
)

type Config struct {
	imagePath    string
	width        int
	invert       bool
	output       string
	contrast     float64
	gamma        float64
	brightness   float64
	aspect       float64
	background   string
	showVersion  bool
	autoContrast bool
	sharpen      float64
	dither       string
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "img2dots: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	config, err := parseFlags(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if config.showVersion {
		_, err := fmt.Fprintln(stdout, version)
		return err
	}
	img, err := readImage(config.imagePath, stdin)
	if err != nil {
		return err
	}
	art, err := imageToBraille(img, config)
	if err != nil {
		return err
	}
	if config.output != "" && config.output != "-" {
		if err := writeOutput(config.output, art); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		// Keep stdout exclusively for Braille art so redirection and pipes work.
		_, err = fmt.Fprintf(stderr, "Braille art saved to %s\n", config.output)
		return err
	}
	_, err = io.WriteString(stdout, art)
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func readImage(path string, stdin io.Reader) (image.Image, error) {
	input := stdin
	if path != "-" {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open image: %w", err)
		}
		defer file.Close()
		input = file
	}
	// Read at most one byte past the encoded limit, then check dimensions before
	// asking a decoder to allocate the full image. The same limits apply to stdin.
	data, err := io.ReadAll(io.LimitReader(input, maxInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) > maxInputBytes {
		return nil, errors.New("encoded image exceeds the 64 MiB limit")
	}
	header, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image header (expected WebP, JPEG, or PNG): %w", err)
	}
	if header.Width <= 0 || header.Height <= 0 || header.Width > maxInputPixels/header.Height {
		return nil, errors.New("image exceeds the 64 megapixel limit")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image (expected WebP, JPEG, or PNG): %w", err)
	}
	return img, nil
}

func parseFlags(args []string, stderr io.Writer) (Config, error) {
	var config Config
	flags := flag.NewFlagSet("img2dots", flag.ContinueOnError)
	// Return parse errors to main instead of printing the same error twice.
	flags.SetOutput(io.Discard)
	flags.IntVar(&config.width, "w", 80, "Output width in characters (10-1000)")
	flags.IntVar(&config.width, "width", 80, "Output width in characters (10-1000)")
	flags.BoolVar(&config.invert, "i", false, "Invert light and dark dots")
	flags.BoolVar(&config.invert, "invert", false, "Invert light and dark dots")
	flags.StringVar(&config.output, "o", "", "Output file (- for stdout)")
	flags.StringVar(&config.output, "output", "", "Output file (- for stdout)")
	flags.Float64Var(&config.contrast, "c", 1, "Contrast (0.5-3.0)")
	flags.Float64Var(&config.contrast, "contrast", 1, "Contrast (0.5-3.0)")
	flags.Float64Var(&config.gamma, "g", 1, "Gamma (0.5-2.0)")
	flags.Float64Var(&config.gamma, "gamma", 1, "Gamma (0.5-2.0)")
	flags.Float64Var(&config.brightness, "b", 0, "Brightness (-0.5 to 0.5)")
	flags.Float64Var(&config.brightness, "brightness", 0, "Brightness (-0.5 to 0.5)")
	flags.Float64Var(&config.aspect, "aspect", 0.45, "Character width/height ratio (0.1-1.0)")
	flags.StringVar(&config.background, "background", "white", "Background for transparent pixels: white or black")
	flags.BoolVar(&config.autoContrast, "auto-contrast", false, "Expand the tonal range, ignoring the darkest/lightest 1 percent")
	flags.Float64Var(&config.sharpen, "sharpen", 0.35, "Edge sharpening strength (0-2; 0 disables it)")
	flags.StringVar(&config.dither, "dither", "floyd-steinberg", "Dithering: floyd-steinberg or none")
	flags.BoolVar(&config.showVersion, "version", false, "Print the version")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Convert WebP, JPEG, or PNG images to Unicode Braille art.")
		fmt.Fprintln(stderr, "\nUsage: img2dots [options] image\n       img2dots [options] -  (read from stdin)")
		fmt.Fprintln(stderr, "\nOptions:")
		flags.SetOutput(stderr)
		flags.PrintDefaults()
		flags.SetOutput(io.Discard)
		fmt.Fprintln(stderr, "\nExamples:\n  img2dots -w 160 -auto-contrast photo.webp\n  img2dots -w 120 -auto-contrast -g 1.4 photo.png\n  img2dots -dither none -o art.txt logo.png\n  cat photo.jpg | img2dots -")
	}
	if err := flags.Parse(args); err != nil {
		return config, err
	}
	if config.showVersion {
		return config, nil
	}
	if flags.NArg() != 1 {
		return config, errors.New("provide one image path, or - for stdin; use -h for help")
	}
	config.imagePath = flags.Arg(0)
	return config, config.validate()
}

func (config Config) validate() error {
	if config.width < 10 || config.width > maxWidth {
		return fmt.Errorf("width must be between 10 and %d characters", maxWidth)
	}
	for _, param := range []struct {
		name            string
		value, min, max float64
	}{
		{"contrast", config.contrast, 0.5, 3},
		{"gamma", config.gamma, 0.5, 2},
		{"brightness", config.brightness, -0.5, 0.5},
		{"aspect", config.aspect, 0.1, 1},
		{"sharpen", config.sharpen, 0, 2},
	} {
		if math.IsNaN(param.value) || math.IsInf(param.value, 0) || param.value < param.min || param.value > param.max {
			return fmt.Errorf("%s must be a finite number between %g and %g", param.name, param.min, param.max)
		}
	}
	if config.background != "white" && config.background != "black" {
		return errors.New("background must be white or black")
	}
	if config.dither != "none" && config.dither != "floyd-steinberg" {
		return errors.New("dither must be floyd-steinberg or none")
	}
	return nil
}
