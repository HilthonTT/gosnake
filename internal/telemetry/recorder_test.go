package telemetry

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRecorderKeepsMostRecentInOrder(t *testing.T) {
	r := NewRecorder(3)
	for _, w := range []int{1, 2, 3, 4, 5} {
		r.Record(tea.WindowSizeMsg{Width: w, Height: w})
	}

	got := r.Snapshot()
	want := []string{"3x3", "4x4", "5x5"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Detail != want[i] {
			t.Fatalf("snapshot[%d] = %q, want %q", i, got[i].Detail, want[i])
		}
	}
}

func TestRecorderPartial(t *testing.T) {
	r := NewRecorder(5)
	r.Record(tea.WindowSizeMsg{Width: 1, Height: 2})

	got := r.Snapshot()
	if len(got) != 1 || got[0].Detail != "1x2" || got[0].Type != "tea.WindowSizeMsg" {
		t.Fatalf("snapshot = %+v", got)
	}
}
