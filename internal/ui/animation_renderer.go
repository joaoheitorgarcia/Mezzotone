package ui

import (
	"slices"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
)

type AnimationFrame struct {
	Frame    string
	Duration time.Duration
}

type AnimationRenderer struct {
	frames            []AnimationFrame
	currentFrameIndex int
	isPlaying         bool
	stopKeys          []string

	id  int
	tag int
}

type TickMsg struct {
	Time time.Time
	ID   int
	tag  int
}

var lastID int64

func NewAnimationRenderer(frames []AnimationFrame, stopKeys []string) AnimationRenderer {
	return AnimationRenderer{
		frames:   frames,
		stopKeys: stopKeys,
		id:       nextID(),
	}
}

func (m *AnimationRenderer) Init() tea.Cmd {
	return nil
}

func (m *AnimationRenderer) Update(msg tea.Msg) (AnimationRenderer, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if slices.Contains(m.stopKeys, msg.String()) {
			m.isPlaying = false
			return *m, nil
		}
	case TickMsg:
		if msg.ID > 0 && msg.ID != m.id {
			return *m, nil
		}

		if msg.tag > 0 && msg.tag != m.tag {
			return *m, nil
		}
		m.currentFrameIndex++
		if (m.currentFrameIndex) >= len(m.frames) {
			m.currentFrameIndex = 0
		}

		m.tag++
		return *m, m.tick(m.id, m.tag)
	}
	return *m, nil
}

func (m *AnimationRenderer) View() string {
	return m.frames[m.currentFrameIndex].Frame
}

func (m *AnimationRenderer) StartAnimation() tea.Msg {
	m.isPlaying = true
	return TickMsg{
		Time: time.Now(),
		ID:   m.id,
		tag:  m.tag,
	}
}

func (m *AnimationRenderer) StopAnimation() {
	m.isPlaying = false
}

func (m *AnimationRenderer) tick(id, tag int) tea.Cmd {
	return tea.Tick(m.frames[m.currentFrameIndex].Duration, func(t time.Time) tea.Msg {
		return TickMsg{
			Time: t,
			ID:   id,
			tag:  tag,
		}
	})
}

func (m *AnimationRenderer) IsAnimationPlaying() bool {
	return m.isPlaying
}

func (m *AnimationRenderer) GetId() int {
	return m.id
}

func (m *AnimationRenderer) GetcurrentFrameIndex() int {
	return m.currentFrameIndex
}

func (m *AnimationRenderer) Frames() []AnimationFrame {
	frames := make([]AnimationFrame, len(m.frames))
	copy(frames, m.frames)
	return frames
}

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}
