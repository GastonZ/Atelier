package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gastonz/atelier/internal/actions"
	"github.com/gastonz/atelier/internal/config"
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

	if err := tui.RunWithMonitor(reg, op, cb, scanner, watcher, prices, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "atelier:", err)
		os.Exit(1)
	}
}
