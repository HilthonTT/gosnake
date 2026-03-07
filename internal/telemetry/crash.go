package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/adrg/xdg"
)

const Version = "dev"

type CrashReport struct {
	Timestamp  string            `json:"timestamp"`
	Version    string            `json:"version"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	GoVersion  string            `json:"goVersion"`
	TermWidth  int               `json:"termWidth"`
	TermHeight int               `json:"termHeight"`
	GameMode   string            `json:"gameMode,omitempty"`
	GameState  map[string]any    `json:"gameState,omitempty"`
	RecentMsgs []MsgRecord       `json:"recentMessages"`
	PanicValue string            `json:"panic"`
	StackTrace string            `json:"stackTrace"`
	Extra      map[string]string `json:"extra,omitempty"`
}

type Snapshottable interface {
	Snapshot() map[string]any
}

func NewCrashReport(
	panicVal any,
	stack []byte,
	recorder *Recorder,
	termWidth, termHeight int,
	gameMode string,
	gameSnap map[string]any,
) *CrashReport {
	var msgs []MsgRecord
	if recorder != nil {
		msgs = recorder.Snapshot()
	}

	return &CrashReport{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Version:    Version,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		GoVersion:  runtime.Version(),
		TermWidth:  termWidth,
		TermHeight: termHeight,
		GameMode:   gameMode,
		GameState:  gameSnap,
		RecentMsgs: msgs,
		PanicValue: fmt.Sprintf("%v", panicVal),
		StackTrace: string(stack),
	}
}

func WriteCrashReport(report *CrashReport) (string, error) {
	dir, err := xdg.DataFile("gosnake/crashes/.keep")
	if err != nil {
		return "", fmt.Errorf("resolve crash direction: %w", err)
	}
	dir = filepath.Dir(dir)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create crash directory: %w", err)
	}

	filename := fmt.Sprintf("crash-%s.json", time.Now().UTC().Format("20060102-150405"))
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal crash report: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write crash report: %w", err)
	}

	return path, nil
}
