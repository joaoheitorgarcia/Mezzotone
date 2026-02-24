package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRenderViewShiftUpAndDownGoToBottomAndTop(t *testing.T) {
	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderView.Width = 6
	m.renderView.Height = 2
	m.renderView.SetContent("line0\nline1\nline2\nline3\nline4")

	if m.renderView.YOffset != 0 {
		t.Fatalf("expected initial Y offset 0, got %d", m.renderView.YOffset)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	model := updated.(*MezzotoneModel)
	if got := m.renderView.ScrollPercent(); got != 1 {
		t.Fatalf("expected shift+down to jump to Bottom, got %f", got)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	model = updated.(*MezzotoneModel)
	if got := m.renderView.ScrollPercent(); got != 0 {
		t.Fatalf("expected shift+up to jump to Top, got %f", got)
	}

}

func TestRenderViewShiftLeftAndRightGoToEdges(t *testing.T) {
	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderView.Width = 4
	m.renderView.Height = 2
	m.renderView.SetContent("abcdefghij\nabcdefghij")

	if got := m.renderView.HorizontalScrollPercent(); got != 0 {
		t.Fatalf("expected initial horizontal scroll 0, got %f", got)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftRight})
	model := updated.(*MezzotoneModel)
	if got := model.renderView.HorizontalScrollPercent(); got != 1 {
		t.Fatalf("expected shift+right to jump to right edge, got %f", got)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftLeft})
	model = updated.(*MezzotoneModel)
	if got := model.renderView.HorizontalScrollPercent(); got != 0 {
		t.Fatalf("expected shift+left to jump to left edge, got %f", got)
	}
}

func TestRenderViewPgUpAndPgDownGoToTopAndBottom(t *testing.T) {
	m := NewMezzotoneModel()
	m.currentActiveMenu = renderView
	m.renderView.Width = 6
	m.renderView.Height = 2
	m.renderView.SetContent("line0\nline1\nline2\nline3")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	model := updated.(*MezzotoneModel)
	if !model.renderView.AtBottom() {
		t.Fatalf("expected pgdown to jump to bottom in render view")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	model = updated.(*MezzotoneModel)
	if !model.renderView.AtTop() {
		t.Fatalf("expected pgup to jump to top in render view")
	}
}

func TestRenderViewFullscreenToggleWithFUpdatesWidth(t *testing.T) {
	m := NewMezzotoneModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	model := updated.(*MezzotoneModel)
	model.currentActiveMenu = renderView

	if model.style.isRenderViewFullscreen {
		t.Fatalf("expected fullscreen off by default")
	}
	if got, want := model.renderView.Width, 100; got != want {
		t.Fatalf("expected initial render width %d, got %d", want, got)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(*MezzotoneModel)
	if !model.style.isRenderViewFullscreen {
		t.Fatalf("expected fullscreen on after pressing f")
	}
	if got, want := model.renderView.Width, 138; got != want {
		t.Fatalf("expected fullscreen render width %d, got %d", want, got)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(*MezzotoneModel)
	if model.style.isRenderViewFullscreen {
		t.Fatalf("expected fullscreen off after pressing f again")
	}
	if got, want := model.renderView.Width, 100; got != want {
		t.Fatalf("expected non-fullscreen render width %d, got %d", want, got)
	}
}

func TestFullscreenToggleIgnoredOutsideRenderView(t *testing.T) {
	m := NewMezzotoneModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	model := updated.(*MezzotoneModel)

	if got, want := model.currentActiveMenu, filePickerMenu; got != want {
		t.Fatalf("expected to start in file picker menu: want %d got %d", want, got)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(*MezzotoneModel)
	if model.style.isRenderViewFullscreen {
		t.Fatalf("expected fullscreen to remain off outside render view")
	}
	if got, want := model.renderView.Width, 100; got != want {
		t.Fatalf("expected render width to remain %d outside render view, got %d", want, got)
	}
}

func TestWindowResizeKeepsFullscreenRenderWidth(t *testing.T) {
	m := NewMezzotoneModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(*MezzotoneModel)
	model.currentActiveMenu = renderView

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(*MezzotoneModel)
	if !model.style.isRenderViewFullscreen {
		t.Fatalf("expected fullscreen on before resize")
	}
	if got, want := model.renderView.Width, 118; got != want {
		t.Fatalf("expected fullscreen render width %d, got %d", want, got)
	}

	updated, _ = model.Update(tea.WindowSizeMsg{Width: 150, Height: 45})
	model = updated.(*MezzotoneModel)
	if got, want := model.renderView.Width, 148; got != want {
		t.Fatalf("expected fullscreen render width to track resize (%d), got %d", want, got)
	}
}
