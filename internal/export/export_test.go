package export

import (
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestASCIIToPNGCreatesValidPNG(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.png")

	err := ASCIIToPNG("hello\nworld", outPath, ASCIIExportOptions{
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
		{ASCII: "frame one", Duration: 40 * time.Millisecond},
		{ASCII: "frame two", Duration: 90 * time.Millisecond},
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
		{ASCII: "short", Duration: 0},
		{ASCII: "this frame is wider\nand taller", Duration: time.Millisecond},
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

	if err := ASCIIToPNG("", outPath, ASCIIExportOptions{
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
	if err := ASCIIToPNG(strings.ReplaceAll("a\r\nb", "\r\n", "\n"), filepath.Join(tmpDir, "normalized.png"), ASCIIExportOptions{
		FontSize: 14,
		DPI:      300,
		BG:       color.Black,
		FG:       color.White,
	}); err != nil {
		t.Fatalf("ASCIIToPNG failed for normalized newlines: %v", err)
	}
}
