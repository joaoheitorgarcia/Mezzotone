package termtext_test

import (
	"strings"
	"testing"

	"codeberg.org/JoaoGarcia/Mezzotone/internal/termtext"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestTruncateLinesANSI_ZeroOrNegativeWidth(t *testing.T) {
	in := "hello\nworld"

	if got := termtext.TruncateLinesANSI(in, 0); got != "" {
		t.Fatalf("expected empty string for zero width, got %q", got)
	}
	if got := termtext.TruncateLinesANSI(in, -1); got != "" {
		t.Fatalf("expected empty string for negative width, got %q", got)
	}
}

func TestTruncateLinesANSI_TruncatesEachLine(t *testing.T) {
	in := "abcdef\nghijkl"
	got := termtext.TruncateLinesANSI(in, 4)

	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "abc…" {
		t.Fatalf("unexpected first line: %q", lines[0])
	}
	if lines[1] != "ghi…" {
		t.Fatalf("unexpected second line: %q", lines[1])
	}
}

func TestTruncateLinesANSI_ANSIVisibleWidthBounded(t *testing.T) {
	colored := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("abcdefgh")
	got := termtext.TruncateLinesANSI(colored, 5)

	if w := ansi.StringWidth(got); w > 5 {
		t.Fatalf("expected visible width <= 5, got %d", w)
	}
	if !strings.Contains(got, "…") {
		t.Fatalf("expected ellipsis in truncated ANSI output, got %q", got)
	}
}
