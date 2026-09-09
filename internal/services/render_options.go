package services

import (
	"fmt"
	"math"
	"slices"

	"github.com/joaoheitorgarcia/Mezzotone/internal/global"
)

type RenderOptions struct {
	// renderMode: selected rendering algorithm.
	renderMode string
	// textSize: roughly controls how many pixels map to one character horizontally.
	// both render modes use this value
	textSize int
	// fontAspect: terminal characters are typically taller than they are wide, vertical cell size is textSize * fontAspect.
	// both render modes use this value
	fontAspect float64
	// both render modes use this value
	RenderColor bool

	// directionalRender: optional Edge Awareness. Derive edge magnitude/orientation from luminanceGrid and choose glyphs accordingly.
	// character ramp render mode uses this value
	directionalRender bool
	// character ramp render mode uses this value
	edgeThreshold float64
	// reverseChars: invert ramp direction (useful for dark terminals / preference).
	// character ramp render mode uses this value
	reverseChars bool
	// highContrast: optional contrast curve applied after cell luminance averaging.
	// character ramp render mode uses this value
	highContrast bool
	// character ramp render mode uses this value
	runeMode string

	// density: controls dot density
	// braile render mode uses this value
	density float64
	// dithering: with dithering some dots are intentionally turned on/off in a pattern so the eye perceives an intermediate tone
	// braile render mode uses this value
	dithering bool
}

func NewRenderOptions(
	renderMode string,
	textSize int,
	fontAspect float64,
	renderColor bool,

	directionalRender bool,
	edgeThreshold float64,
	reverseChars bool,
	highContrast bool,
	runeMode string,

	density float64,
	dithering bool,
) (RenderOptions, error) {
	if !slices.Contains(global.AvailableRenderModes(), renderMode) {
		return RenderOptions{}, fmt.Errorf("invalid render mode: %s", renderMode)
	}
	if !slices.Contains(global.AvailableRuneModes(), runeMode) && renderMode == global.CHARACTER_RAMP {
		return RenderOptions{}, fmt.Errorf("invalid rune mode: %s", runeMode)
	}
	if !isUnitInterval(edgeThreshold) {
		return RenderOptions{}, fmt.Errorf("edge threshold must be between 0 and 1")
	}
	if !isUnitInterval(density) {
		return RenderOptions{}, fmt.Errorf("density must be between 0 and 1")
	}

	return RenderOptions{
		renderMode:  renderMode,
		textSize:    textSize,
		fontAspect:  fontAspect,
		RenderColor: renderColor,

		directionalRender: directionalRender,
		edgeThreshold:     edgeThreshold,
		reverseChars:      reverseChars,
		highContrast:      highContrast,
		runeMode:          runeMode,

		density:   density,
		dithering: dithering,
	}, nil
}

func isUnitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}
