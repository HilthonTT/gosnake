package components

import (
	"time"

	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
)

type Stopwatch interface {
	Init() tea.Cmd
	Update(tea.Msg) (tea.Model, tea.Cmd)
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
	return &stopwatchImpl{model: stopwatch.NewWithInterval(interval)}
}

func (s *stopwatchImpl) Elapsed() time.Duration {
	return s.model.Elapsed()
}

func (s *stopwatchImpl) ID() int {
	return s.model.ID()
}

func (s *stopwatchImpl) Init() tea.Cmd {
	return s.model.Init()
}

func (s *stopwatchImpl) Reset() tea.Cmd {
	return s.model.Reset()
}

func (s *stopwatchImpl) SetInterval(d time.Duration) {
	s.model.Interval = d
}

func (s *stopwatchImpl) Stop() tea.Cmd {
	return s.model.Stop()
}

func (s *stopwatchImpl) Toggle() tea.Cmd {
	return s.model.Toggle()
}

func (s *stopwatchImpl) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := s.model.Update(msg)
	s.model = m
	return s, cmd
}

func (s *stopwatchImpl) View() string {
	return s.model.View()
}
