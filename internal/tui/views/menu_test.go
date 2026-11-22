package views

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/HilthonTT/gosnake/internal/tui"
)

func press(m *MenuModel, keys ...tea.KeyPressMsg) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range keys {
		_, cmd = m.Update(k)
	}
	return cmd
}

var (
	keyEnter = tea.KeyPressMsg{Code: tea.KeyEnter}
	keyTab   = tea.KeyPressMsg{Code: tea.KeyTab}
	keyRight = tea.KeyPressMsg{Code: tea.KeyRight}
)

func TestMenuRequiresName(t *testing.T) {
	m := NewMenuModel(tui.NewMenuInput())
	m.Init()

	// Jump to the Play button and try to start without a name.
	press(m, keyTab, keyTab, keyTab)
	if cmd := press(m, keyEnter); cmd != nil {
		if _, ok := cmd().(tui.SwitchModeMsg); ok {
			t.Fatal("game started without a name")
		}
	}
	if m.focus != fieldName || m.nameErr == "" {
		t.Fatalf("focus = %v, err = %q; want the name field with an error", m.focus, m.nameErr)
	}
}

func TestMenuStartsChosenGame(t *testing.T) {
	m := NewMenuModel(tui.NewMenuInput())
	m.Init()

	press(m, tea.KeyPressMsg{Code: 'n', Text: "n"}, tea.KeyPressMsg{Code: 'e', Text: "e"}, tea.KeyPressMsg{Code: 'o', Text: "o"})
	press(m, keyEnter)                                      // name → mode
	press(m, keyRight, keyRight)                            // Normal → Crazy → Versus AI
	press(m, keyTab, tea.KeyPressMsg{Code: '7', Text: "7"}) // level 7

	cmd := press(m, keyEnter)
	msg, ok := cmd().(tui.SwitchModeMsg)
	if !ok {
		t.Fatalf("enter should start the game, got %#v", cmd())
	}
	in := msg.Input.(*tui.SingleInput)
	if msg.Target != tui.ModeAI || in.Level != 7 || in.Username != "neo" {
		t.Fatalf("started %v with %+v", msg.Target, in)
	}
}

func TestMenuPrefillFocusesPlay(t *testing.T) {
	m := NewMenuModel(&tui.MenuInput{Username: "neo", Mode: tui.ModeCrazy, Level: 3})
	if m.focus != fieldPlay || modeTiles[m.mode].mode != tui.ModeCrazy || m.level != 3 {
		t.Fatalf("focus = %v, mode = %v, level = %d", m.focus, modeTiles[m.mode].mode, m.level)
	}
}
