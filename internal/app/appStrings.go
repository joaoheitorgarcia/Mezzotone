package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func helpBinding(key, description string, keyStyle, descriptionStyle lipgloss.Style) string {
	return "    " + keyStyle.Width(16).Render(key) + " " + descriptionStyle.Render(description)
}

func buildRenderHelpText(style styleVariables) string {
	sectionStyle := lipgloss.NewStyle().Foreground(style.styleColors.primary).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(style.styleColors.selected)
	descriptionStyle := lipgloss.NewStyle().Foreground(style.styleColors.white)
	separatorString := "                                                                                   "
	separator := lipgloss.NewStyle().Underline(true).Render(separatorString)

	return strings.Join([]string{
		sectionStyle.Render("CONTROLS"),
		"",
		sectionStyle.Render("* Global"),
		helpBinding("h", "Toggle help", keyStyle, descriptionStyle),
		helpBinding("esc", "Back / Quit", keyStyle, descriptionStyle),
		helpBinding("ctrl+c", "Quit", keyStyle, descriptionStyle),
		"",
		sectionStyle.Render("* File Picker"),
		helpBinding("j/k or up/down", "Navigate options", keyStyle, descriptionStyle),
		helpBinding("pgdown", "Go To Bottom", keyStyle, descriptionStyle),
		helpBinding("pgup", "Go To Top", keyStyle, descriptionStyle),
		helpBinding("enter/right", "Open directory or select image", keyStyle, descriptionStyle),
		helpBinding("left/backspace", "Go back directory", keyStyle, descriptionStyle),
		"",
		sectionStyle.Render("* Render Options"),
		helpBinding("j/k or up/down", "Navigate options", keyStyle, descriptionStyle),
		helpBinding("pgdown", "Go To Bottom", keyStyle, descriptionStyle),
		helpBinding("pgup", "Go To Top", keyStyle, descriptionStyle),
		helpBinding("enter", "Edit numeric fields / change enum / confirm", keyStyle, descriptionStyle),
		helpBinding("space", "Toggle bool values", keyStyle, descriptionStyle),
		helpBinding("left/right", "Change enum values", keyStyle, descriptionStyle),
		helpBinding("esc", "Cancel edit or go back to file picker", keyStyle, descriptionStyle),
		"",
		sectionStyle.Render("* Render View"),
		helpBinding("arrows", "Scroll output/help", keyStyle, descriptionStyle),
		helpBinding("h", "Hide help", keyStyle, descriptionStyle),
		helpBinding("f", "Toggle Fullscreen", keyStyle, descriptionStyle),
		helpBinding("pgdown", "Go To Bottom", keyStyle, descriptionStyle),
		helpBinding("pgup", "Go To Top", keyStyle, descriptionStyle),
		helpBinding("shift+up", "Go To Bottom", keyStyle, descriptionStyle),
		helpBinding("shift+down", "Go To Top", keyStyle, descriptionStyle),
		helpBinding("shift+left", "Go To Left", keyStyle, descriptionStyle),
		helpBinding("shift+right", "Go To Right", keyStyle, descriptionStyle),
		"",
		helpBinding("c", "Copy to clipboard", keyStyle, descriptionStyle),
		helpBinding("t", "Export to txt", keyStyle, descriptionStyle),
		helpBinding("i", "Export to image", keyStyle, descriptionStyle),
		helpBinding("g", "Export to gif", keyStyle, descriptionStyle),
		helpBinding("v", "Export to video", keyStyle, descriptionStyle),
		"",
		separator,
		"",
		sectionStyle.Render("RENDER OPTIONS HELP"),
		"",
		sectionStyle.Render("Text Size"),
		"  " + descriptionStyle.Render("Character cell width in pixels."),
		"  " + descriptionStyle.Render("Larger values reduce detail."),
		"",
		sectionStyle.Render("Font Aspect"),
		"  " + descriptionStyle.Render("Character height ratio vs width to match terminal font shape."),
		"",
		sectionStyle.Render("Directional Render"),
		"  " + descriptionStyle.Render("Uses edge direction to place oriented glyphs on strong edges."),
		"  " + descriptionStyle.Render("Works better for font aspect = 1"),
		"",
		sectionStyle.Render("Edge Threshold"),
		"  " + descriptionStyle.Render("Edge cutoff (0..1) for directional glyph replacement."),
		"  " + descriptionStyle.Render("Determines the threshold of is considered an edge."),
		"",
		sectionStyle.Render("Reverse Chars"),
		"  " + descriptionStyle.Render("Inverts ramp mapping for terminals/themes"),
		"",
		sectionStyle.Render("High Contrast"),
		"  " + descriptionStyle.Render("Applies stronger luminance contrast before glyph mapping."),
		"  " + descriptionStyle.Render("May improve render quality or edge detection depending on images."),
		"",
		sectionStyle.Render("Rune Mode"),
		"  " + descriptionStyle.Render("Selector for what type of characters will be renderer."),
		"  " + descriptionStyle.Render("Available options: ASCII, UNICODE, DOTS, RECTANGLES, BARS."),
	}, "\n")
}
