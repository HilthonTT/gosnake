// Command demo records the README GIFs. It drives the real TUI models
// in-process, with an autopilot that steers the snake by reading the rendered
// board, and draws every frame with a monospace font into an animated GIF.
//
//	go run ./scripts/demo -out docs/assets
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/HilthonTT/gosnake/internal/config"
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/starter"
)

const (
	termW = 124
	termH = 46
)

func main() {
	out := flag.String("out", "docs/assets", "directory to write the GIFs to")
	fontPath := flag.String("font", defaultFont(), "monospace TTF used to draw text")
	fallback := flag.String("fallback", defaultFallbackFont(), "TTF used for symbols the main font lacks")
	only := flag.String("only", "", "record just this scene (demo, crazy, versus)")
	flag.Parse()

	r, err := newRenderer(*fontPath, *fallback, 14)
	if err != nil {
		log.Fatal(err)
	}

	scenes := []struct {
		name string
		run  func(*recorder) error
	}{
		{"demo", sceneDemo},
		{"crazy", sceneMode(tui.ModeCrazy, 6, 14*time.Second)},
		{"versus", sceneMode(tui.ModeAI, 6, 14*time.Second)},
	}
	for _, s := range scenes {
		if *only != "" && *only != s.name {
			continue
		}
		rec := &recorder{r: r}
		log.Printf("recording %s…", s.name)
		if err := s.run(rec); err != nil {
			log.Fatalf("%s: %v", s.name, err)
		}
		path := filepath.Join(*out, s.name+".gif")
		if err := rec.writeGIF(path); err != nil {
			log.Fatal(err)
		}
		st, _ := os.Stat(path)
		log.Printf("wrote %s (%d frames, %d KB)", path, len(rec.frames), st.Size()/1024)
	}
}

// newApp builds the real root model on a throwaway database seeded with a
// few scores, so the leaderboard has something to show.
func newApp(mode tui.Mode, in tui.SwitchModeInput) (*starter.Model, func(), error) {
	dir, err := os.MkdirTemp("", "gosnake-demo")
	if err != nil {
		return nil, nil, err
	}
	db, err := data.NewDB(filepath.Join(dir, "demo.db"))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = db.Close(); _ = os.RemoveAll(dir) }

	repo := data.NewLeaderboardRepository(db)
	for _, e := range []struct {
		name         string
		score, level int
		mode         data.GameMode
	}{
		{"trinity", 420, 5, data.GameModeNormal},
		{"morpheus", 260, 3, data.GameModeNormal},
		{"tank", 150, 2, data.GameModeNormal},
		{"switch", 90, 1, data.GameModeNormal},
		{"apoc", 310, 4, data.GameModeCrazy},
		{"oracle", 180, 3, data.GameModeAI},
	} {
		if _, err := repo.Save(e.name, e.score, e.level, e.mode); err != nil {
			cleanup()
			return nil, nil, err
		}
	}

	m, err := starter.NewModel(starter.NewInput(mode, repo, config.Default(), in, nil))
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return m, cleanup, nil
}

// sceneDemo walks through the menu, plays a round of normal mode and ends
// on the leaderboard.
func sceneDemo(rec *recorder) error {
	app, cleanup, err := newApp(tui.ModeMenu, tui.NewMenuInput())
	if err != nil {
		return err
	}
	defer cleanup()

	rt := newRuntime(app, rec)
	rt.wait(1200 * time.Millisecond)
	for _, ch := range "neo" {
		rt.key(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		rt.wait(180 * time.Millisecond)
	}
	rt.wait(400 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyEnter}) // → mode
	rt.wait(700 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyRight}) // Crazy
	rt.wait(700 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyRight}) // Versus AI
	rt.wait(700 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyRight}) // back to Normal
	rt.wait(600 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyTab}) // → level
	rt.wait(400 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyRight})
	rt.wait(300 * time.Millisecond)
	for range 2 {
		rt.key(tea.KeyPressMsg{Code: tea.KeyRight})
		rt.wait(300 * time.Millisecond)
	}
	rt.key(tea.KeyPressMsg{Code: tea.KeyRight}) // level 5
	rt.wait(600 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyEnter}) // play
	rt.wait(1200 * time.Millisecond)
	rt.key(tea.KeyPressMsg{Code: tea.KeyRight}) // the board is dimmed until the first move

	pilot := &autopilot{rt: rt}
	pilot.play(func() bool { return pilot.eaten >= 5 }, 30*time.Second)
	pilot.crash(10 * time.Second)
	rt.wait(2500 * time.Millisecond)

	rt.key(tea.KeyPressMsg{Code: tea.KeyEnter}) // → leaderboard
	rt.wait(3500 * time.Millisecond)
	return nil
}

// sceneMode plays one mode directly for about d.
func sceneMode(mode tui.Mode, level int, d time.Duration) func(*recorder) error {
	return func(rec *recorder) error {
		app, cleanup, err := newApp(mode, tui.NewSingleInput(mode, level, "neo"))
		if err != nil {
			return err
		}
		defer cleanup()

		rt := newRuntime(app, rec)
		rt.wait(1000 * time.Millisecond)
		rt.key(tea.KeyPressMsg{Code: tea.KeyRight})
		pilot := &autopilot{rt: rt}
		start := time.Now()
		pilot.play(func() bool { return time.Since(start) > d }, d+5*time.Second)
		if !pilot.over() {
			rt.key(tea.KeyPressMsg{Code: 'p', Text: "p"})
		}
		rt.wait(1800 * time.Millisecond)
		return nil
	}
}

func defaultFont() string {
	for _, p := range []string{
		`C:\Windows\Fonts\CascadiaMono.ttf`,
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/System/Library/Fonts/Menlo.ttc",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func defaultFallbackFont() string {
	for _, p := range []string{
		`C:\Windows\Fonts\seguisym.ttf`,
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func init() {
	log.SetFlags(0)
	log.SetPrefix("demo: ")
	if defaultFont() == "" {
		fmt.Fprintln(os.Stderr, "demo: no default font found; pass -font")
	}
}
