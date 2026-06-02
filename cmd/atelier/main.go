package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gastonz/atelier/internal/actions"
	"github.com/gastonz/atelier/internal/audio"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/git"
	"github.com/gastonz/atelier/internal/nowplaying"
	"github.com/gastonz/atelier/internal/registry"
	"github.com/gastonz/atelier/internal/transcripts"
	"github.com/gastonz/atelier/internal/tui"
	"github.com/gastonz/atelier/internal/version"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "version", "--version", "-v":
		fmt.Println("atelier", version.Version)
		return

	case "help", "--help", "-h":
		fmt.Println("Dragon Atelier — Mission Control for AI Workflows")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  atelier            Launch the TUI")
		fmt.Println("  atelier tui        Launch the TUI (explicit)")
		fmt.Println("  atelier version    Print version and exit")
		fmt.Println("  atelier help       Show this help")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --help, -h         Show this help")
		fmt.Println("  --version, -v      Print version and exit")
		return

	case "", "tui":
		runTUI()
		return

	default:
		fmt.Fprintf(os.Stderr, "atelier: unknown command %q\n", cmd)
		fmt.Fprintln(os.Stderr, "Run 'atelier help' for usage.")
		os.Exit(2)
	}
}

// runTUI builds concrete dependencies and launches the Bubble Tea program.
// It is extracted from main() to keep the switch statement readable.
func runTUI() {
	reg := registry.NewFileRegistry()
	op := actions.NewOpener()
	cb := actions.NewClipboard()

	// Config — non-fatal: fall back to defaults and surface warning later via flash.
	cfg, cfgErr := config.LoadAtelierConfig(config.DefaultAtelierConfigPath())
	if cfgErr != nil {
		cfg = config.DefaultAtelierConfig()
	}

	// Agent monitor concrete dependencies.
	home, _ := os.UserHomeDir()
	claudeRoot := filepath.Join(home, ".claude", "projects")
	prices := transcripts.DefaultPriceTable()
	scanner := transcripts.NewFileScanner(claudeRoot, transcripts.RealClock{}, prices)
	watcher := transcripts.NewFsnotifyWatcher(cfg.PollingInterval())

	// Daily driver pack dependencies (NEW — Batch 4).
	// engramClient may be nil if the DB is missing; the TUI is nil-safe and will
	// surface the error as a flash message when the user opens ScreenMemoryBrowser.
	dbPath, dbPathErr := engram.DefaultDBPath()
	var engramClient engram.Client
	if dbPathErr == nil {
		engramClient, _ = engram.NewClient(dbPath)
	}
	gitStatusReader := git.NewStatusReader()
	gitLogReader := git.NewLogReader()

	// Now-playing provider (SMTC on Windows, no-op elsewhere) — ambient widget
	// on the welcome screen. nil-safe: the TUI hides the card when absent.
	npProvider := nowplaying.NewProvider()

	// Audio analyzer (WASAPI loopback on Windows) drives the live waveform.
	// Closed on exit to stop the capture goroutine cleanly.
	analyzer := audio.NewAnalyzer()
	defer analyzer.Close()

	if err := tui.RunWithDailyPack(
		reg, op, cb,
		scanner, watcher, prices, cfg,
		engramClient, gitStatusReader, gitLogReader,
		npProvider, analyzer,
	); err != nil {
		fmt.Fprintln(os.Stderr, "atelier:", err)
		os.Exit(1)
	}
}
