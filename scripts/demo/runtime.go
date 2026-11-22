package main

import (
	"fmt"
	"os"
	"reflect"
	"time"

	tea "charm.land/bubbletea/v2"
)

// runtime is a tiny Bubble Tea event loop: commands run on goroutines and
// their messages are applied to the model on the caller's goroutine, so the
// recorder can interleave scripted key presses with real timer ticks.
type runtime struct {
	model tea.Model
	msgs  chan tea.Msg
	rec   *recorder
	grid  *grid
	quit  bool
}

func newRuntime(m tea.Model, rec *recorder) *runtime {
	rt := &runtime{model: m, msgs: make(chan tea.Msg, 256), rec: rec}
	rt.spawn(m.Init())
	rt.apply(tea.WindowSizeMsg{Width: termW, Height: termH})
	return rt
}

func (rt *runtime) spawn(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		if msg := cmd(); msg != nil {
			rt.msgs <- msg
		}
	}()
}

// apply delivers msg to the model and records the resulting frame.
func (rt *runtime) apply(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			rt.spawn(c)
		}
		return
	case tea.QuitMsg:
		rt.quit = true
		return
	}
	// tea.Sequence produces an unexported []tea.Cmd: run those in order.
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		cmds := make([]tea.Cmd, v.Len())
		for i := range cmds {
			cmds[i] = v.Index(i).Interface().(tea.Cmd)
		}
		go func() {
			for _, c := range cmds {
				if c == nil {
					continue
				}
				if m := c(); m != nil {
					rt.msgs <- m
				}
			}
		}()
		return
	}
	if debug {
		fmt.Fprintf(os.Stderr, "%T\n", msg)
	}
	var cmd tea.Cmd
	rt.model, cmd = rt.model.Update(msg)
	rt.spawn(cmd)
	rt.capture()
}

func (rt *runtime) capture() {
	rt.grid = parseANSI(rt.model.View().Content, termW, termH)
	rt.rec.add(rt.grid, time.Now())
}

// key presses k as the user would.
func (rt *runtime) key(k tea.KeyPressMsg) {
	rt.apply(k)
}

// wait runs the event loop for d.
func (rt *runtime) wait(d time.Duration) {
	deadline := time.After(d)
	for {
		select {
		case msg := <-rt.msgs:
			rt.apply(msg)
		case <-deadline:
			return
		}
	}
}

// next runs the event loop until the model handles one message or timeout
// passes. It reports whether a message was handled.
func (rt *runtime) next(timeout time.Duration) bool {
	select {
	case msg := <-rt.msgs:
		rt.apply(msg)
		return true
	case <-time.After(timeout):
		return false
	}
}

var debug = os.Getenv("DEMO_DEBUG") != ""
