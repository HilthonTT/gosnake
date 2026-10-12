// Package starter hosts the root Bubble Tea model that switches between the
// menu, game and leaderboard views.
package starter

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/HilthonTT/gosnake/internal/config"
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/telemetry"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/views"
)

const defaultRecorderSize = 500

var errInvalidInput = errors.New("invalid switch mode input")

// Input configures the starter model.
type Input struct {
	mode      tui.Mode
	repo      *data.LeaderboardRepository
	cfg       *config.Config
	switchIn  tui.SwitchModeInput
	submitter views.ScoreSubmitter
}

// NewInput creates the starter input. submitter may be nil to disable online
// score submission.
func NewInput(
	mode tui.Mode,
	repo *data.LeaderboardRepository,
	cfg *config.Config,
	switchIn tui.SwitchModeInput,
	submitter views.ScoreSubmitter,
) *Input {
	return &Input{
		mode:      mode,
		repo:      repo,
		cfg:       cfg,
		switchIn:  switchIn,
		submitter: submitter,
	}
}

// Crash captures a panic raised while updating or rendering a view.
type Crash struct {
	Value any
	Stack []byte
}

// snapshotter is implemented by views that can describe their game state.
type snapshotter interface {
	GameSnapshot() map[string]any
}

var _ tea.Model = &Model{}

type Model struct {
	child           tea.Model
	cfg             *config.Config
	forceQuitKey    key.Binding
	leaderboardRepo *data.LeaderboardRepository
	submitter       views.ScoreSubmitter
	recorder        *telemetry.Recorder
	currentMode     tui.Mode
	crash           *Crash

	width  int
	height int

	ExitError error
}

func NewModel(in *Input) (*Model, error) {
	m := &Model{
		cfg:             in.cfg,
		leaderboardRepo: in.repo,
		submitter:       in.submitter,
		forceQuitKey:    key.NewBinding(key.WithKeys(in.cfg.Keys.ForceQuit...)),
		recorder:        telemetry.NewRecorder(defaultRecorderSize),
		currentMode:     in.mode,
	}

	if err := m.setChild(in.mode, in.switchIn); err != nil {
		return nil, fmt.Errorf("setting child model: %w", err)
	}

	return m, nil
}

func (m *Model) Recorder() *telemetry.Recorder { return m.recorder }
func (m *Model) CurrentMode() tui.Mode         { return m.currentMode }
func (m *Model) TermSize() (int, int)          { return m.width, m.height }

// Crash returns the panic that stopped the program, if any.
func (m *Model) Crash() *Crash { return m.crash }

func (m *Model) GameSnapshot() map[string]any {
	if s, ok := m.child.(snapshotter); ok {
		return s.GameSnapshot()
	}
	return nil
}

func (m *Model) Init() tea.Cmd {
	return m.initChild()
}

func (m *Model) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	if m.crash != nil {
		return m, tea.Quit
	}

	// Catch panics here rather than letting Bubble Tea swallow them, so the
	// launcher can write a crash report with the game state attached.
	defer func() {
		if r := recover(); r != nil {
			m.recordPanic(r)
			cmd = tea.Quit
		}
	}()

	m.recorder.Record(msg)

	switch msg := msg.(type) {
	case tui.FatalErrorMsg:
		m.ExitError = msg
		return m, tea.Quit

	case tea.KeyPressMsg:
		if key.Matches(msg, m.forceQuitKey) {
			return m, tea.Quit
		}

	case tui.SwitchModeMsg:
		if err := m.setChild(msg.Target, msg.Input); err != nil {
			return m, tui.FatalErrorCmd(fmt.Errorf("setting child model: %w", err))
		}
		return m, m.initChild()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	m.child, cmd = m.child.Update(msg)
	return m, cmd
}

func (m *Model) View() tea.View {
	v := tea.NewView(m.renderChild())
	v.AltScreen = true
	v.WindowTitle = "GoSnake"
	return v
}

func (m *Model) renderChild() (content string) {
	if m.crash != nil {
		return "gosnake crashed — exiting…"
	}

	// A panic during rendering is recorded; the next Update quits.
	defer func() {
		if r := recover(); r != nil {
			m.recordPanic(r)
			content = "gosnake crashed — exiting…"
		}
	}()

	return m.child.View().Content
}

func (m *Model) recordPanic(r any) {
	if m.crash == nil {
		m.crash = &Crash{Value: r, Stack: debug.Stack()}
	}
}

func (m *Model) setChild(mode tui.Mode, switchIn tui.SwitchModeInput) error {
	if rv := reflect.ValueOf(switchIn); !rv.IsValid() || rv.IsNil() {
		return errors.New("switchIn is not valid")
	}

	switch mode {
	case tui.ModeMenu:
		menuIn, ok := switchIn.(*tui.MenuInput)
		if !ok {
			return fmt.Errorf("switchIn is not a MenuInput: %w", errInvalidInput)
		}
		m.child = views.NewMenuModel(menuIn)

	case tui.ModeNormal, tui.ModeCrazy, tui.ModeAI:
		singleIn, ok := switchIn.(*tui.SingleInput)
		if !ok {
			return fmt.Errorf("switchIn is not a SingleInput: %w", errInvalidInput)
		}
		child, err := views.NewSingleModel(singleIn, views.SingleDeps{
			Keys:      m.cfg.Keys,
			Submitter: m.submitter,
			BestScore: m.bestScore(singleIn.Mode),
		})
		if err != nil {
			return fmt.Errorf("creating single model: %w", err)
		}
		m.child = child

	case tui.ModeLeaderboard:
		leaderboardIn, ok := switchIn.(*tui.LeaderboardInput)
		if !ok {
			return fmt.Errorf("switchIn is not a LeaderboardInput: %w", errInvalidInput)
		}
		if err := m.prepareLeaderboard(leaderboardIn); err != nil {
			return err
		}
		m.child = views.NewLeaderboardModel(leaderboardIn)

	default:
		return fmt.Errorf("invalid mode %d", mode)
	}

	m.currentMode = mode
	return nil
}

// prepareLeaderboard saves the just-finished game (if any) and loads entries.
func (m *Model) prepareLeaderboard(in *tui.LeaderboardInput) error {
	if in.NewEntry != nil {
		if in.NewEntry.Name == "" {
			in.NewEntry.Name = "Anonymous"
		}
		id, err := m.leaderboardRepo.Save(in.NewEntry.Name, in.NewEntry.Score, in.NewEntry.Level, in.NewEntry.Mode)
		if err != nil {
			return fmt.Errorf("saving leaderboard entry: %w", err)
		}
		in.NewEntry.ID = id
	}

	entries, err := m.leaderboardRepo.All()
	if err != nil {
		return fmt.Errorf("fetching leaderboard entries: %w", err)
	}
	in.SetEntries(entries)
	return nil
}

// bestScore is the local record for mode, or 0 if unknown.
func (m *Model) bestScore(mode tui.Mode) int {
	top, err := m.leaderboardRepo.GetTopNByMode(1, views.GameModeFromTUI(mode))
	if err != nil || len(top) == 0 {
		return 0
	}
	return top[0].Score
}

func (m *Model) initChild() tea.Cmd {
	initCmd := m.child.Init()
	var sizeCmd tea.Cmd
	m.child, sizeCmd = m.child.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return tea.Batch(initCmd, sizeCmd)
}
