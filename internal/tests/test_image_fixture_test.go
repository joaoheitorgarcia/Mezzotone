package tests

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const generatedFixtureName = "gradient_edges.png"
const corruptFixtureName = "corrupt_image.png"

func ensureGeneratedFixture(t *testing.T) string {
	t.Helper()

	testDataDir := filepath.Join("testdata")
	if err := os.MkdirAll(testDataDir, 0o755); err != nil {
		t.Fatalf("failed creating testdata dir: %v", err)
	}

	imagePath := filepath.Join(testDataDir, generatedFixtureName)
	if _, err := os.Stat(imagePath); err == nil {
		return imagePath
	}

	const width = 160
	const height = 96
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	// Smooth gradient background.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / (width - 1))
			g := uint8((y * 255) / (height - 1))
			b := uint8(((x + y) * 255) / (width + height - 2))
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// Add high-contrast edges.
	for y := 12; y < 46; y++ {
		for x := 16; x < 70; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
		}
	}
	for y := 50; y < 86; y++ {
		for x := 95; x < 150; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for i := 0; i < 96; i++ {
		x := 20 + i
		y := 95 - i
		if x >= 0 && x < width && y >= 0 && y < height {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	f, err := os.Create(imagePath)
	if err != nil {
		t.Fatalf("failed creating generated image: %v", err)
	}
	defer func() { _ = f.Close() }()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed writing generated image: %v", err)
	}

	return imagePath
}

func ensureCorruptFixture(t *testing.T) string {
	t.Helper()

	testDataDir := filepath.Join("testdata")
	if err := os.MkdirAll(testDataDir, 0o755); err != nil {
		t.Fatalf("failed creating testdata dir: %v", err)
	}

	imagePath := filepath.Join(testDataDir, corruptFixtureName)
	if _, err := os.Stat(imagePath); err == nil {
		return imagePath
	}

	// Intentionally invalid PNG bytes for decode error testing.
	data := []byte("this-is-not-a-valid-png")
	if err := os.WriteFile(imagePath, data, 0o644); err != nil {
		t.Fatalf("failed writing corrupt image fixture: %v", err)
	}

	return imagePath
}
