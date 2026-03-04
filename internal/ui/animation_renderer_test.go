package ui_test

import (
	"testing"
	"time"

	"github.com/joaoheitorgarcia/Mezzotone/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func newAnimationRendererForTests() ui.AnimationRenderer {
	animationFrames := []ui.AnimationFrame{
		{Frame: "⣿", Duration: time.Duration(2) * 10 * time.Millisecond},
		{Frame: "⣷", Duration: time.Duration(2) * 10 * time.Millisecond},
		{Frame: "⣧", Duration: time.Duration(2) * 10 * time.Millisecond},
		{Frame: "⣇", Duration: time.Duration(2) * 10 * time.Millisecond},
		{Frame: "⣆", Duration: time.Duration(2) * 10 * time.Millisecond},
		{Frame: "⣄", Duration: time.Duration(2) * 10 * time.Millisecond},
		{Frame: "⣀", Duration: time.Duration(2) * 10 * time.Millisecond},
	}

	return ui.NewAnimationRenderer(animationFrames, []string{"esc"})
}

func TestAnimationRenderer_Start_And_Stops_Animation(t *testing.T) {
	m := newAnimationRendererForTests()
	m.StartAnimation()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.IsAnimationPlaying() {
		t.Fatalf("expected animation to be played")
	}

	m.StopAnimation()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.IsAnimationPlaying() {
		t.Fatalf("expected animation to be stopped")
	}
}

func TestAnimationRenderer_TickMsg_Updates_Frames_Correctly(t *testing.T) {
	m := newAnimationRendererForTests()

	startMsg := m.StartAnimation()
	if m.GetcurrentFrameIndex() != 0 {
		t.Fatalf("expected current frame to be 0 at start")
	}
	m, _ = m.Update(startMsg)
	if m.GetcurrentFrameIndex() != 1 {
		t.Fatalf("expected current frame to be 1")
	}

	m, _ = m.Update(ui.TickMsg{Time: time.Now(), ID: m.GetId()})
	if m.GetcurrentFrameIndex() != 2 {
		t.Fatalf("expected current frame to be 2")
	}

	m, _ = m.Update(ui.TickMsg{Time: time.Now(), ID: m.GetId()})
	if m.GetcurrentFrameIndex() != 3 {
		t.Fatalf("expected current frame to be 3")
	}

	m, _ = m.Update(ui.TickMsg{Time: time.Now(), ID: m.GetId()})
	m, _ = m.Update(ui.TickMsg{Time: time.Now(), ID: m.GetId()})
	m, _ = m.Update(ui.TickMsg{Time: time.Now(), ID: m.GetId()})
	m, _ = m.Update(ui.TickMsg{Time: time.Now(), ID: m.GetId()})
	if m.GetcurrentFrameIndex() != 0 {
		t.Fatalf("expected current frame to be 0 after looping")
	}

}

func TestAnimationRenderer_Stop_Key_Stops_Animation(t *testing.T) {
	m := newAnimationRendererForTests()

	startMsg := m.StartAnimation()
	m, _ = m.Update(startMsg)
	if !m.IsAnimationPlaying() {
		t.Fatalf("expected animation to be played")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.IsAnimationPlaying() {
		t.Fatalf("expected animation to be stopped")
	}
}

func TestAnimationRenderer_Non_Stop_Key_Does_Not_Stop_Animation(t *testing.T) {
	m := newAnimationRendererForTests()

	startMsg := m.StartAnimation()
	m, _ = m.Update(startMsg)
	if !m.IsAnimationPlaying() {
		t.Fatalf("expected animation to be played")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if !m.IsAnimationPlaying() {
		t.Fatalf("expected animation to be played")
	}
}

func TestAnimationRenderer_Update_Tick_returnsNextCmd(t *testing.T) {
	m := newAnimationRendererForTests()

	startMsg := m.StartAnimation()
	m, _ = m.Update(startMsg)
	m, cmd := m.Update(ui.TickMsg{Time: time.Now(), ID: m.GetId()})
	if cmd == nil {
		t.Fatalf("expected cmd to be not nil")
	}
}

func TestAnimationRenderer_Starting_New_Animation_Assigns_New_Id(t *testing.T) {
	m := newAnimationRendererForTests()
	startMsg := m.StartAnimation()
	m, _ = m.Update(startMsg)
	oldId := m.GetId()

	m2 := newAnimationRendererForTests()
	startMsg2 := m2.StartAnimation()
	m2, _ = m2.Update(startMsg2)
	newID := m2.GetId()

	if newID == oldId {
		t.Fatalf("expected tag to be %d, got %d", newID, oldId)
	}
}
