package data

import (
	"path/filepath"
	"testing"
)

func newTestRepo(t *testing.T) *LeaderboardRepository {
	t.Helper()
	db, err := NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewLeaderboardRepository(db)
}

func TestLeaderboardRepository(t *testing.T) {
	repo := newTestRepo(t)

	seed := []struct {
		name  string
		score int
		mode  GameMode
	}{
		{"ann", 30, GameModeNormal},
		{"bob", 90, GameModeCrazy},
		{"ann", 60, GameModeAI},
	}
	var ids []int
	for _, s := range seed {
		id, err := repo.Save(s.name, s.score, 1, s.mode)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}

	all, err := repo.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Score != 90 || all[2].Score != 30 {
		t.Fatalf("All() not ordered by score: %+v", all)
	}

	top, err := repo.GetTopN(1)
	if err != nil || len(top) != 1 || top[0].Name != "bob" {
		t.Fatalf("GetTopN(1) = %+v, %v", top, err)
	}

	ai, err := repo.GetTopNByMode(10, GameModeAI)
	if err != nil || len(ai) != 1 || ai[0].Score != 60 {
		t.Fatalf("GetTopNByMode = %+v, %v", ai, err)
	}

	ann, err := repo.GetByName("ann")
	if err != nil || len(ann) != 2 || ann[0].Score != 60 {
		t.Fatalf("GetByName = %+v, %v", ann, err)
	}

	if err := repo.Delete(ids[0]); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ids[0]); err == nil {
		t.Fatal("deleting a missing entry should fail")
	}
}
