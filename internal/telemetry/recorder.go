package telemetry

import (
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

type MsgRecord struct {
	Type      string    `json:"type"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"ts"`
}

// Recorder keeps the most recent Bubble Tea messages in a fixed-size ring
// buffer so a crash report can show what led up to a panic.
type Recorder struct {
	mu   sync.Mutex
	buf  []MsgRecord
	next int  // index the next record is written to
	full bool // true once the buffer has wrapped
}

func NewRecorder(size int) *Recorder {
	return &Recorder{buf: make([]MsgRecord, max(size, 1))}
}

func (r *Recorder) Record(msg tea.Msg) {
	rec := MsgRecord{
		Type:      fmt.Sprintf("%T", msg),
		Timestamp: time.Now(),
	}

	switch m := msg.(type) {
	case tea.KeyMsg:
		rec.Detail = m.String()
	case tea.WindowSizeMsg:
		rec.Detail = fmt.Sprintf("%dx%d", m.Width, m.Height)
	case tea.MouseMsg:
		mouse := m.Mouse()
		rec.Detail = fmt.Sprintf("mouse %d,%d", mouse.X, mouse.Y)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.buf[r.next] = rec
	r.next = (r.next + 1) % len(r.buf)
	if r.next == 0 {
		r.full = true
	}
}

// Snapshot returns the recorded messages, oldest first.
func (r *Recorder) Snapshot() []MsgRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.full {
		return append([]MsgRecord(nil), r.buf[:r.next]...)
	}
	out := make([]MsgRecord, 0, len(r.buf))
	out = append(out, r.buf[r.next:]...)
	return append(out, r.buf[:r.next]...)
}
