package components

import (
	"time"

	"charm.land/bubbles/v2/stopwatch"
	tea "charm.land/bubbletea/v2"
)

// Stopwatch wraps bubbles' stopwatch behind an interface so it can be
// swapped out in tests and so its interval can be changed while running.
type Stopwatch interface {
	Init() tea.Cmd
	Update(tea.Msg) (Stopwatch, tea.Cmd)
	View() string
	Elapsed() time.Duration
	SetInterval(time.Duration)
	ID() int
	Reset() tea.Cmd
	Toggle() tea.Cmd
	Stop() tea.Cmd
}

type stopwatchImpl struct {
	model stopwatch.Model
}

func NewStopwatchWithInterval(interval time.Duration) Stopwatch {
	return &stopwatchImpl{model: stopwatch.New(stopwatch.WithInterval(interval))}
}

func (s *stopwatchImpl) Elapsed() time.Duration      { return s.model.Elapsed() }
func (s *stopwatchImpl) ID() int                     { return s.model.ID() }
func (s *stopwatchImpl) Init() tea.Cmd               { return s.model.Init() }
func (s *stopwatchImpl) Reset() tea.Cmd              { return s.model.Reset() }
func (s *stopwatchImpl) SetInterval(d time.Duration) { s.model.Interval = d }
func (s *stopwatchImpl) Stop() tea.Cmd               { return s.model.Stop() }
func (s *stopwatchImpl) Toggle() tea.Cmd             { return s.model.Toggle() }
func (s *stopwatchImpl) View() string                { return s.model.View() }

func (s *stopwatchImpl) Update(msg tea.Msg) (Stopwatch, tea.Cmd) {
	var cmd tea.Cmd
	s.model, cmd = s.model.Update(msg)
	return s, cmd
}
