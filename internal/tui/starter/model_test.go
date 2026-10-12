package starter

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/HilthonTT/gosnake/internal/config"
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/tui"
)

func newTestModel(t *testing.T, mode tui.Mode, in tui.SwitchModeInput) *Model {
	t.Helper()
	db, err := data.NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	m, err := NewModel(NewInput(mode, data.NewLeaderboardRepository(db), config.Default(), in, nil))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// isQuit reports whether cmd (possibly a batch) resolves to tea.Quit.
func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if isQuit(c) {
				return true
			}
		}
	}
	return false
}

func TestForceQuitQuits(t *testing.T) {
	m := newTestModel(t, tui.ModeMenu, tui.NewMenuInput())

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if !isQuit(cmd) {
		t.Fatal("ctrl+c should quit the program")
	}
}

func TestPanicInChildIsCaptured(t *testing.T) {
	m := newTestModel(t, tui.ModeMenu, tui.NewMenuInput())
	m.child = panicModel{}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if m.Crash() == nil || m.Crash().Value != "boom" || len(m.Crash().Stack) == 0 {
		t.Fatalf("crash = %+v; panic should be captured", m.Crash())
	}
	if !isQuit(cmd) {
		t.Fatal("a panic should quit the program")
	}
}

func TestSwitchToGameHonoursLevel(t *testing.T) {
	m := newTestModel(t, tui.ModeNormal, tui.NewSingleInput(tui.ModeNormal, 7, "tester"))

	snap := m.GameSnapshot()
	if snap["level"] != 7 {
		t.Fatalf("level = %v, want 7", snap["level"])
	}
	if m.CurrentMode() != tui.ModeNormal {
		t.Fatalf("mode = %v", m.CurrentMode())
	}
}

func TestLeaderboardSavesNewEntry(t *testing.T) {
	entry := &data.LeaderboardEntry{Score: 120, Level: 2, Mode: data.GameModeAI}
	m := newTestModel(t, tui.ModeLeaderboard, tui.NewLeaderboardInput(tui.WithNewEntry(entry)))

	if entry.ID == 0 || entry.Name != "Anonymous" {
		t.Fatalf("entry = %+v; should be saved with a default name", entry)
	}
	if m.CurrentMode() != tui.ModeLeaderboard {
		t.Fatalf("mode = %v", m.CurrentMode())
	}
}

type panicModel struct{}

func (panicModel) Init() tea.Cmd                       { return nil }
func (panicModel) Update(tea.Msg) (tea.Model, tea.Cmd) { panic("boom") }
func (panicModel) View() tea.View                      { return tea.NewView("") }

func TestViewsRenderInAltScreen(t *testing.T) {
	cases := []struct {
		name string
		mode tui.Mode
		in   tui.SwitchModeInput
		want string
	}{
		{"menu", tui.ModeMenu, tui.NewMenuInput(), "eat · grow · survive"},
		{"normal", tui.ModeNormal, tui.NewSingleInput(tui.ModeNormal, 1, "t"), "NORMAL"},
		{"crazy", tui.ModeCrazy, tui.NewSingleInput(tui.ModeCrazy, 1, "t"), "CRAZY"},
		{"ai", tui.ModeAI, tui.NewSingleInput(tui.ModeAI, 1, "t"), "VS AI"},
		{"leaderboard", tui.ModeLeaderboard, tui.NewLeaderboardInput(), "L E A D E R B O A R D"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t, tc.mode, tc.in)
			m.Init()
			m.Update(tea.WindowSizeMsg{Width: 160, Height: 60})

			v := m.View()
			if !v.AltScreen {
				t.Error("view should use the alt screen")
			}
			if !strings.Contains(v.Content, tc.want) {
				t.Errorf("view does not contain %q", tc.want)
			}
		})
	}
}

func TestMenuRemembersLastGame(t *testing.T) {
	m := newTestModel(t, tui.ModeCrazy, tui.NewSingleInput(tui.ModeCrazy, 4, "hans"))

	m.Update(tui.SwitchModeMsg{Target: tui.ModeLeaderboard, Input: tui.NewLeaderboardInput()})
	m.Update(tui.SwitchModeMsg{Target: tui.ModeMenu, Input: tui.NewMenuInput()})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 45})

	if v := m.View().Content; !strings.Contains(v, "hans") {
		t.Error("menu should pre-fill the last player name")
	}
}

func TestLeaderboardOffersReplay(t *testing.T) {
	m := newTestModel(t, tui.ModeNormal, tui.NewSingleInput(tui.ModeNormal, 2, "hans"))
	m.Update(tui.SwitchModeMsg{Target: tui.ModeLeaderboard, Input: tui.NewLeaderboardInput()})

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should start a replay")
	}
	msg, ok := cmd().(tui.SwitchModeMsg)
	if !ok || msg.Target != tui.ModeNormal {
		t.Fatalf("got %#v, want a switch to the normal game", msg)
	}
	if in := msg.Input.(*tui.SingleInput); in.Level != 2 || in.Username != "hans" {
		t.Fatalf("replay input = %+v", in)
	}
}

func TestGameWaitsForFirstMove(t *testing.T) {
	m := newTestModel(t, tui.ModeNormal, tui.NewSingleInput(tui.ModeNormal, 1, "t"))
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 45})
	if v := m.View().Content; !strings.Contains(v, "press a direction to start") {
		t.Error("game should wait for the first direction key")
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v := m.View().Content; strings.Contains(v, "press a direction to start") {
		t.Error("a direction key should start the game")
	}
}
