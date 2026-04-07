package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

const usage = `DDSM — Deadlock Dedicated Server Manager v%s

Usage:
  ddsm                     Launch interactive TUI
  ddsm --help              Show this help message
  ddsm status              List all servers with status
  ddsm create              Create a new server (interactive)
  ddsm start [id|all]      Start server(s)
  ddsm stop [id|all]       Stop server(s)
  ddsm restart [id|all]    Restart server(s)
  ddsm delete [id]         Delete a server
  ddsm logs [id]           Tail live server logs
  ddsm rcon [id] [cmd]     Execute RCON command
  ddsm attach [id]         Attach to server container
  ddsm config              Show current configuration
  ddsm config edit         Open config in $EDITOR
  ddsm doctor              Run health diagnostics
`

func main() {
	if len(os.Args) < 2 {
		fmt.Println("TUI not yet implemented")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "--help", "-h", "help":
		fmt.Printf(usage, version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'ddsm --help' for usage.\n", os.Args[1])
		os.Exit(1)
	}
}
