package main

import (
	"fmt"
	"os"

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
		if err := tui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "atelier:", err)
			os.Exit(1)
		}
		return

	default:
		fmt.Fprintf(os.Stderr, "atelier: unknown command %q\n", cmd)
		fmt.Fprintln(os.Stderr, "Run 'atelier help' for usage.")
		os.Exit(2)
	}
}
