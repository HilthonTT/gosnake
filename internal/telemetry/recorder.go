package telemetry

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type MsgRecord struct {
	Type      string    `json:"type"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"ts"`
}

type Recorder struct {
	buf  []MsgRecord
	size int
	mu   sync.Mutex
}

func NewRecorder(size int) *Recorder {
	return &Recorder{buf: make([]MsgRecord, 0, size), size: size}
}

func (r *Recorder) Record(msg tea.Msg) {
	r.mu.Lock()
	defer r.mu.Unlock()

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
		rec.Detail = fmt.Sprintf("mouse %d,%d", m.X, m.Y)
	}

	if len(r.buf) >= r.size {
		r.buf = r.buf[1:]
	}
	r.buf = append(r.buf, rec)
}

func (r *Recorder) Snapshot() []MsgRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]MsgRecord, len(r.buf))
	copy(out, r.buf)
	return out
}
