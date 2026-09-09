package global

const (
	ASCII      = "ASCII"
	UNICODE    = "UNICODE"
	DOTS       = "DOTS"
	RECTANGLES = "RECTANGLES"
	BARS       = "BARS"

	CHARACTER_RAMP = "CHAR RAMP"
	BRAILLE        = "BRAILLE"

	INITIAL_RENDER_MODE = CHARACTER_RAMP
)

func AvailableRuneModes() []string {
	return []string{ASCII, UNICODE, DOTS, RECTANGLES, BARS}
}

func AvailableRenderModes() []string {
	return []string{CHARACTER_RAMP, BRAILLE}
}

func NextRenderMode(currentMode string) string {
	modes := AvailableRenderModes()
	for i, mode := range modes {
		if mode == currentMode {
			return modes[(i+1)%len(modes)]
		}
	}

	return modes[0]
}
