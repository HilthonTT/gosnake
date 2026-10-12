package snake

import (
	"testing"
	"time"
)

func TestSteeringRejectsReversal(t *testing.T) {
	s := NewSteering(Right)

	s.Queue(Left)
	if got := s.Advance(); got != Right {
		t.Fatalf("reversal accepted: got %v, want right", got)
	}

	s.Queue(Up)
	if got := s.Advance(); got != Up {
		t.Fatalf("turn rejected: got %v, want up", got)
	}

	// Reversal is judged against the committed direction, not the queued one.
	s.Queue(Down)
	if got := s.Advance(); got != Up {
		t.Fatalf("reversal of committed direction accepted: got %v", got)
	}
}

func TestAdvance(t *testing.T) {
	body := []Point{{2, 0}, {1, 0}, {0, 0}}

	moved := Advance(body, Point{3, 0}, false)
	if want := []Point{{3, 0}, {2, 0}, {1, 0}}; !equal(moved, want) {
		t.Fatalf("Advance without growth = %v, want %v", moved, want)
	}

	grown := Advance(body, Point{3, 0}, true)
	if len(grown) != 4 || grown[3] != (Point{0, 0}) {
		t.Fatalf("Advance with growth = %v", grown)
	}

	if !equal(body, []Point{{2, 0}, {1, 0}, {0, 0}}) {
		t.Fatalf("Advance mutated its input: %v", body)
	}
}

func TestHitsBodyIgnoresMovingTail(t *testing.T) {
	body := []Point{{1, 0}, {1, 1}, {0, 1}, {0, 0}}
	tail := Point{0, 0}

	if HitsBody(body, tail, true) {
		t.Error("moving tail should not be an obstacle")
	}
	if !HitsBody(body, tail, false) {
		t.Error("static tail should be an obstacle")
	}
}

func TestRandomFreePoint(t *testing.T) {
	var all []Point
	for y := range DefaultRows {
		for x := range DefaultCols {
			all = append(all, Point{x, y})
		}
	}

	if _, ok := RandomFreePoint(all); ok {
		t.Fatal("expected no free point on a full board")
	}

	free := all[len(all)/2]
	crowded := append(append([]Point(nil), all[:len(all)/2]...), all[len(all)/2+1:]...)
	got, ok := RandomFreePoint(crowded)
	if !ok || got != free {
		t.Fatalf("RandomFreePoint on nearly full board = %v, %v; want %v", got, ok, free)
	}

	if NewFood(all) != nil {
		t.Fatal("NewFood should return nil when the board is full")
	}
}

func TestScoringLevelsUpRelativeToStart(t *testing.T) {
	s, err := NewDefaultScoring(5)
	if err != nil {
		t.Fatal(err)
	}

	s.AddPoints(PointsPerLevel - 1)
	if s.Level() != 5 {
		t.Fatalf("level = %d, want 5", s.Level())
	}

	s.AddPoints(1)
	if s.Level() != 6 {
		t.Fatalf("level = %d, want 6", s.Level())
	}

	s.AddPoints(100 * PointsPerLevel)
	if s.Level() != MaxLevel {
		t.Fatalf("level = %d, want capped at %d", s.Level(), MaxLevel)
	}
}

func TestScoringValidation(t *testing.T) {
	for _, level := range []int{0, -1, MaxLevel + 1} {
		if _, err := NewDefaultScoring(level); err == nil {
			t.Errorf("NewDefaultScoring(%d) should fail", level)
		}
	}
}

func TestGetTickInterval(t *testing.T) {
	if GetTickInterval(1) != BaseTickInterval {
		t.Error("level 1 should use the base interval")
	}
	if GetTickInterval(2) >= GetTickInterval(1) {
		t.Error("higher levels should tick faster")
	}
	if GetTickInterval(1000) != MinTickInterval {
		t.Error("interval should be clamped")
	}
	if GetTickInterval(MaxLevel) < MinTickInterval || GetTickInterval(MaxLevel) > time.Second {
		t.Error("max level interval out of range")
	}
}

func equal(a, b []Point) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
