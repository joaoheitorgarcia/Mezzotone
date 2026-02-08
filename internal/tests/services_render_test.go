package tests

import (
	"path/filepath"
	"testing"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/services"
)

func mustRenderOptions(
	t *testing.T,
	textSize int,
	fontAspect float64,
	directional bool,
	edgeThreshold float64,
	reverse bool,
	highContrast bool,
	runeMode string,
) services.RenderOptions {
	t.Helper()
	opts, err := services.NewRenderOptions(textSize, fontAspect, directional, edgeThreshold, reverse, highContrast, runeMode)
	if err != nil {
		t.Fatalf("failed creating render options: %v", err)
	}
	return opts
}

func mustConvertImageToString(t *testing.T, imagePath string, opts services.RenderOptions) string {
	t.Helper()
	out, err := services.ConvertImageToString(imagePath, opts)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected non-empty rune grid")
	}
	if len(out[0]) == 0 {
		t.Fatalf("expected non-empty rune grid row")
	}
	return services.ImageRuneArrayIntoString(out)
}

func TestNewRenderOptionsRejectsInvalidRuneMode(t *testing.T) {
	_, err := services.NewRenderOptions(10, 2.3, false, 0.6, false, false, "INVALID")
	if err == nil {
		t.Fatalf("expected error for invalid rune mode")
	}
}

func TestConvertImageToStringGeneratedFixtureHasContent(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	opts := mustRenderOptions(t, 8, 2.0, false, 0.6, false, false, "ASCII")

	output := mustConvertImageToString(t, imagePath, opts)
	if len(output) < 10 {
		t.Fatalf("expected output text to have meaningful length, got %d", len(output))
	}
}

func TestConvertImageToStringDifferentRuneModesProduceDifferentOutput(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	ascii := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, 0.6, false, false, "ASCII"))
	dots := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, 0.6, false, false, "DOTS"))

	if ascii == dots {
		t.Fatalf("expected ASCII and DOTS outputs to differ")
	}
}

func TestConvertImageToStringReverseCharsChangesOutput(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	normal := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, 0.6, false, false, "ASCII"))
	reversed := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, 0.6, true, false, "ASCII"))

	if normal == reversed {
		t.Fatalf("expected reverse chars option to change output")
	}
}

func TestConvertImageToStringDirectionalRenderChangesOutput(t *testing.T) {
	imagePath := ensureGeneratedFixture(t)
	plain := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, false, 0.6, false, false, "ASCII"))
	directional := mustConvertImageToString(t, imagePath, mustRenderOptions(t, 8, 2.0, true, 0.4, false, false, "ASCII"))

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
			a:    mustRenderOptions(t, 8, 2.0, false, 0.6, false, false, "ASCII"),
			b:    mustRenderOptions(t, 8, 2.0, false, 0.6, false, true, "ASCII"),
		},
		{
			name: "edge threshold changed under directional mode",
			a:    mustRenderOptions(t, 8, 2.0, true, 0.2, false, false, "ASCII"),
			b:    mustRenderOptions(t, 8, 2.0, true, 0.9, false, false, "ASCII"),
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
	validOpts := mustRenderOptions(t, 8, 2.0, false, 0.6, false, false, "ASCII")

	t.Run("missing file returns error", func(t *testing.T) {
		missingPath := filepath.Join("testdata", "does-not-exist.png")
		_, err := services.ConvertImageToString(missingPath, validOpts)
		if err == nil {
			t.Fatalf("expected error for missing file")
		}
	})

	t.Run("corrupt file returns decode error", func(t *testing.T) {
		corruptPath := ensureCorruptFixture(t)
		_, err := services.ConvertImageToString(corruptPath, validOpts)
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

	out := services.ImageRuneArrayIntoString(in)
	expected := "ab\ncd\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}
