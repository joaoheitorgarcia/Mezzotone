package export

import (
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func asciiToRunes(s string) [][]rune {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([][]rune, 0, len(lines))
	for _, line := range lines {
		out = append(out, []rune(line))
	}
	return out
}

func imageHasPixelMatching(img image.Image, pred func(r, g, b, a uint32) bool) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if pred(r, g, b, a) {
				return true
			}
		}
	}
	return false
}

func TestASCIIToPNGCreatesValidPNG(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.png")

	err := ASCIIToPNG(asciiToRunes("hello\nworld"), nil, outPath, ASCIIExportOptions{
		FontSize:     14,
		DPI:          300,
		BG:           color.Black,
		FG:           color.White,
		TargetAspect: 1.0 / 2.3,
	})
	if err != nil {
		t.Fatalf("ASCIIToPNG failed: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open png output: %v", err)
	}
	defer f.Close()

	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("failed to decode png config: %v", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 {
		t.Fatalf("invalid png dimensions: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestASCIIFramesToGIFCreatesAnimatedGIF(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.gif")

	frames := []ASCIIGIFFrame{
		{FrameRunes: asciiToRunes("frame one"), Duration: 40 * time.Millisecond},
		{FrameRunes: asciiToRunes("frame two"), Duration: 90 * time.Millisecond},
	}

	err := ASCIIFramesToGIF(frames, outPath, ASCIIExportOptions{
		FontSize:     14,
		DPI:          300,
		BG:           color.Black,
		FG:           color.White,
		TargetAspect: 1.0 / 2.3,
	})
	if err != nil {
		t.Fatalf("ASCIIFramesToGIF failed: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open gif output: %v", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("failed to decode animated gif: %v", err)
	}
	if len(g.Image) != len(frames) {
		t.Fatalf("expected %d gif frames, got %d", len(frames), len(g.Image))
	}

	expectedDelays := []int{4, 9}
	for i, want := range expectedDelays {
		if g.Delay[i] != want {
			t.Fatalf("frame %d delay mismatch: want %d got %d", i, want, g.Delay[i])
		}
	}
}

func TestASCIIFramesToGIFNoFramesReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.gif")

	err := ASCIIFramesToGIF(nil, outPath, ASCIIExportOptions{
		FontSize: 14,
		DPI:      300,
		BG:       color.Black,
		FG:       color.White,
	})
	if err == nil {
		t.Fatalf("expected error when exporting gif with no frames")
	}
}

func TestASCIIToTxtWritesContent(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.txt")

	want := "line one\nline two\n"
	if err := ASCIItToTxT(outPath, want); err != nil {
		t.Fatalf("ASCIItToTxT failed: %v", err)
	}

	gotBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read txt output: %v", err)
	}

	if got := string(gotBytes); got != want {
		t.Fatalf("txt content mismatch: want %q got %q", want, got)
	}
}

func TestASCIIFramesToGIFClampsDelayAndNormalizesFrameBounds(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "normalized.gif")

	frames := []ASCIIGIFFrame{
		{FrameRunes: asciiToRunes("short"), Duration: 0},
		{FrameRunes: asciiToRunes("this frame is wider\nand taller"), Duration: time.Millisecond},
	}

	err := ASCIIFramesToGIF(frames, outPath, ASCIIExportOptions{
		FontSize:     14,
		DPI:          300,
		BG:           color.Black,
		FG:           color.White,
		TargetAspect: 1.0 / 2.3,
	})
	if err != nil {
		t.Fatalf("ASCIIFramesToGIF failed: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open gif output: %v", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("failed to decode gif output: %v", err)
	}
	if len(g.Image) != len(frames) {
		t.Fatalf("expected %d gif frames, got %d", len(frames), len(g.Image))
	}

	baseBounds := g.Image[0].Bounds()
	if baseBounds.Dx() < 1 || baseBounds.Dy() < 1 {
		t.Fatalf("invalid first-frame dimensions: %v", baseBounds)
	}

	for i := range g.Image {
		if g.Delay[i] < 1 {
			t.Fatalf("expected clamped delay >= 1 for frame %d, got %d", i, g.Delay[i])
		}
		if g.Image[i].Bounds() != baseBounds {
			t.Fatalf("expected uniform frame bounds %v, got %v for frame %d", baseBounds, g.Image[i].Bounds(), i)
		}
	}
}

func TestASCIIToPNGEmptyInputStillCreatesImage(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "empty.png")

	if err := ASCIIToPNG(asciiToRunes(""), nil, outPath, ASCIIExportOptions{
		FontSize: 14,
		DPI:      300,
		BG:       color.Black,
		FG:       color.White,
	}); err != nil {
		t.Fatalf("ASCIIToPNG failed for empty input: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open png output: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode png output: %v", err)
	}
	if img.Bounds().Dx() < 1 || img.Bounds().Dy() < 1 {
		t.Fatalf("invalid png dimensions: %v", img.Bounds())
	}

	// Verify the image is not fully transparent.
	var nonTransparent bool
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y && !nonTransparent; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				nonTransparent = true
				break
			}
		}
	}
	if !nonTransparent {
		t.Fatalf("expected at least one non-transparent pixel")
	}

	// Smoke-check that decoding a newline-normalized payload remains valid.
	if err := ASCIIToPNG(asciiToRunes(strings.ReplaceAll("a\r\nb", "\r\n", "\n")), nil, filepath.Join(tmpDir, "normalized.png"), ASCIIExportOptions{
		FontSize: 14,
		DPI:      300,
		BG:       color.Black,
		FG:       color.White,
	}); err != nil {
		t.Fatalf("ASCIIToPNG failed for normalized newlines: %v", err)
	}
}

func TestASCIIToPNGRenderColorUsesPerCellColor(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "color.png")

	runes := asciiToRunes("AB")
	colors := [][]color.NRGBA{
		{
			{R: 255, G: 0, B: 0, A: 255},
			{R: 0, G: 255, B: 0, A: 255},
		},
	}

	if err := ASCIIToPNG(runes, colors, outPath, ASCIIExportOptions{
		FontSize:    20,
		DPI:         300,
		BG:          color.Black,
		FG:          color.White,
		RenderColor: true,
	}); err != nil {
		t.Fatalf("ASCIIToPNG failed: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open png output: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode png output: %v", err)
	}

	hasRed := imageHasPixelMatching(img, func(r, g, b, a uint32) bool {
		return a > 0 && r > g+0x2000 && r > b+0x2000
	})
	hasGreen := imageHasPixelMatching(img, func(r, g, b, a uint32) bool {
		return a > 0 && g > r+0x2000 && g > b+0x2000
	})
	if !hasRed || !hasGreen {
		t.Fatalf("expected rendered image to include both red-dominant and green-dominant glyph pixels (got red=%v green=%v)", hasRed, hasGreen)
	}
}

func TestASCIIFramesToGIFRenderColorUsesPerFrameColors(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "color.gif")

	frames := []ASCIIGIFFrame{
		{
			FrameRunes: asciiToRunes("X"),
			FrameColors: [][]color.NRGBA{
				{{R: 255, G: 0, B: 0, A: 255}},
			},
			Duration: 20 * time.Millisecond,
		},
		{
			FrameRunes: asciiToRunes("X"),
			FrameColors: [][]color.NRGBA{
				{{R: 0, G: 255, B: 0, A: 255}},
			},
			Duration: 20 * time.Millisecond,
		},
	}

	if err := ASCIIFramesToGIF(frames, outPath, ASCIIExportOptions{
		FontSize:    20,
		DPI:         300,
		BG:          color.Black,
		FG:          color.White,
		RenderColor: true,
	}); err != nil {
		t.Fatalf("ASCIIFramesToGIF failed: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open gif output: %v", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("failed to decode gif output: %v", err)
	}
	if len(g.Image) != 2 {
		t.Fatalf("expected 2 gif frames, got %d", len(g.Image))
	}

	frame0HasRed := imageHasPixelMatching(g.Image[0], func(r, g, b, a uint32) bool {
		return a > 0 && r > g+0x1000 && r > b+0x1000
	})
	frame1HasGreen := imageHasPixelMatching(g.Image[1], func(r, g, b, a uint32) bool {
		return a > 0 && g > r+0x1000 && g > b+0x1000
	})
	if !frame0HasRed || !frame1HasGreen {
		t.Fatalf("expected frame 0 to contain red-dominant pixels and frame 1 to contain green-dominant pixels (got red=%v green=%v)", frame0HasRed, frame1HasGreen)
	}
}
