package services_test

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoheitorgarcia/Mezzotone/internal/global"
	"github.com/joaoheitorgarcia/Mezzotone/internal/services"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

func mustRenderOptions(
	t *testing.T,
	textSize int,
	fontAspect float64,
	renderColor bool,

	directional bool,
	edgeThreshold float64,
	reverse bool,
	highContrast bool,
	runeMode string,

	density float64,
	dithering bool,
) services.RenderOptions {
	t.Helper()
	opts, err := services.NewRenderOptions(global.CHARACTER_RAMP, textSize, fontAspect, renderColor, directional, edgeThreshold, reverse, highContrast, runeMode, density, dithering)
	if err != nil {
		t.Fatalf("failed creating render options: %v", err)
	}
	return opts
}

func mustConvertImageToString(t *testing.T, imagePath string, opts services.RenderOptions) string {
	t.Helper()
	f, err := os.Open(imagePath)
	if err != nil {
		t.Fatalf("failed opening image: %v", err)
	}
	defer func() { _ = f.Close() }()

	inputImg, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("failed decoding image: %v", err)
	}

	out, colors, err := services.ConvertImageToString(inputImg, opts)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected non-empty rune grid")
	}
	if len(out[0]) == 0 {
		t.Fatalf("expected non-empty rune grid row")
	}
	return services.ImageRuneArrayIntoString(out, colors, opts.RenderColor)
}

func TestNewRenderOptionsRejectsInvalidRuneMode(t *testing.T) {
	_, err := services.NewRenderOptions(global.CHARACTER_RAMP, 10, 2.3, false, false, 0.6, false, false, "INVALID", 0.5, false)
	if err == nil {
		t.Fatalf("expected error for invalid rune mode")
	}
}

func TestNewRenderOptionsValidatesUnitIntervalOptions(t *testing.T) {
	validCases := []struct {
		name          string
		edgeThreshold float64
		density       float64
	}{
		{name: "lower bounds", edgeThreshold: 0, density: 0},
		{name: "upper bounds", edgeThreshold: 1, density: 1},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := services.NewRenderOptions(global.CHARACTER_RAMP, 10, 2.3, false, false, tc.edgeThreshold, false, false, global.ASCII, tc.density, false)
			if err != nil {
				t.Fatalf("expected valid bounds, got error: %v", err)
			}
		})
	}

	invalidCases := []struct {
		name          string
		edgeThreshold float64
		density       float64
	}{
		{name: "edge threshold below lower bound", edgeThreshold: -0.1, density: 0.5},
		{name: "edge threshold above upper bound", edgeThreshold: 1.1, density: 0.5},
		{name: "edge threshold NaN", edgeThreshold: math.NaN(), density: 0.5},
		{name: "edge threshold infinity", edgeThreshold: math.Inf(1), density: 0.5},
		{name: "density below lower bound", edgeThreshold: 0.5, density: -0.1},
		{name: "density above upper bound", edgeThreshold: 0.5, density: 1.1},
		{name: "density NaN", edgeThreshold: 0.5, density: math.NaN()},
		{name: "density infinity", edgeThreshold: 0.5, density: math.Inf(1)},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := services.NewRenderOptions(global.CHARACTER_RAMP, 10, 2.3, false, false, tc.edgeThreshold, false, false, global.ASCII, tc.density, false)
			if err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestConvertImageToStringGeneratedFixtureHasContent(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	opts := mustRenderOptions(t, 8, 2.0, false, false, 0.6, false, false, "ASCII", 0.5, false)

	output := mustConvertImageToString(t, imagePath, opts)
	if len(output) < 10 {
		t.Fatalf("expected output text to have meaningful length, got %d", len(output))
	}
}

func TestConvertImageToStringDifferentRuneModesProduceDifferentOutput(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	ascii := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, false, 0.6, false, false, "ASCII", 0.5, false))
	dots := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, false, 0.6, false, false, "DOTS", 0.5, false))

	if ascii == dots {
		t.Fatalf("expected ASCII and DOTS outputs to differ")
	}
}

func TestConvertImageToStringReverseCharsChangesOutput(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	normal := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, false, 0.6, false, false, "ASCII", 0.5, false))
	reversed := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, false, 0.6, true, false, "ASCII", 0.5, false))

	if normal == reversed {
		t.Fatalf("expected reverse chars option to change output")
	}
}

func TestConvertImageToStringDirectionalRenderChangesOutput(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	plain := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, false, 0.6, false, false, "ASCII", 0.5, false))
	directional := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, true, 0.4, false, false, "ASCII", 0.5, false))

	if plain == directional {
		t.Fatalf("expected directional render to change output")
	}
}

func TestConvertImageToStringOptionVariantsChangeOutput(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)

	cases := []struct {
		name string
		a    services.RenderOptions
		b    services.RenderOptions
	}{
		{
			name: "high contrast toggled",
			a:    mustRenderOptions(t, 8, 2.0, false, false, 0.6, false, false, "ASCII", 0.5, false),
			b:    mustRenderOptions(t, 8, 2.0, true, false, 0.6, false, false, "ASCII", 0.5, false),
		},
		{
			name: "edge threshold changed under directional mode",
			a:    mustRenderOptions(t, 8, 2.0, false, true, 0.2, false, false, "ASCII", 0.5, false),
			b:    mustRenderOptions(t, 8, 2.0, false, true, 0.9, false, false, "ASCII", 0.5, false),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outA := mustConvertImageToString(t, imagePath, tc.a)
			outB := mustConvertImageToString(t, imagePath, tc.b)
			if outA == outB {
				t.Fatalf("expected output to differ for variant %q", tc.name)
			}
		})
	}
}

func TestConvertImageToStringFileErrors(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		missingPath := filepath.Join("testdata", "does-not-exist.png")
		_, err := os.Open(missingPath)
		if err == nil {
			t.Fatalf("expected error for missing file")
		}
	})

	t.Run("corrupt file returns decode error", func(t *testing.T) {
		corruptPath := ensureCorruptFixture(t)
		f, err := os.Open(corruptPath)
		if err != nil {
			t.Fatalf("failed opening corrupt fixture: %v", err)
		}
		defer func() { _ = f.Close() }()

		_, _, err = image.Decode(f)
		if err == nil {
			t.Fatalf("expected error for corrupt image")
		}
	})
}

func TestImageRuneArrayIntoStringAddsLineBreaks(t *testing.T) {
	in := [][]rune{
		[]rune("ab"),
		[]rune("cd"),
	}

	out := services.ImageRuneArrayIntoString(in, nil, false)
	expected := "ab\ncd\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestConvertImageToStringRenderColorBuildsAverageColorGrid(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	expected := color.NRGBA{R: 12, G: 34, B: 56, A: 255}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, expected)
		}
	}

	opts := mustRenderOptions(t, 8, 2.0, true, false, 0.6, false, true, "ASCII", 0.5, false)
	runes, colors, err := services.ConvertImageToString(img, opts)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if len(runes) != 1 || len(runes[0]) != 1 {
		t.Fatalf("expected 1x1 rune grid, got %dx%d", len(runes), len(runes[0]))
	}
	if len(colors) != 1 || len(colors[0]) != 1 {
		t.Fatalf("expected 1x1 color grid, got %dx%d", len(colors), len(colors[0]))
	}

	if got := colors[0][0]; got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestImageRuneArrayIntoStringRenderColorAddsANSIForeground(t *testing.T) {
	in := [][]rune{{'X'}}
	colors := [][]color.NRGBA{{{R: 255, G: 0, B: 0, A: 255}}}

	lipgloss.Writer.Profile = colorprofile.TrueColor

	plain := services.ImageRuneArrayIntoString(in, colors, false)
	if plain != "X\n" {
		t.Fatalf("expected plain output %q, got %q", "X\n", plain)
	}

	colored := services.ImageRuneArrayIntoString(in, colors, true)
	if !strings.Contains(colored, "\x1b[38;2;255;0;0m") {
		t.Fatalf("expected ANSI truecolor foreground in output, got %q", colored)
	}
	if !strings.Contains(colored, "X") {
		t.Fatalf("expected rendered rune in colored output, got %q", colored)
	}
}
