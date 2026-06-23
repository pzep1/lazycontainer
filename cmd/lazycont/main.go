package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pz/lazycont/internal/containercli"
	"github.com/pz/lazycont/internal/tui"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		if len(args) > 1 {
			fmt.Fprintf(stderr, "lazycont: unexpected argument %q\n", args[1])
			printUsage(stderr)
			return 2
		}

		switch args[0] {
		case "--help", "-h", "help":
			printUsage(stdout)
			return 0
		case "--version", "-v", "version":
			fmt.Fprintf(stdout, "lazycont %s\n", version)
			return 0
		default:
			fmt.Fprintf(stderr, "lazycont: unexpected argument %q\n", args[0])
			printUsage(stderr)
			return 2
		}
	}

	return runTUI(stderr)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `lazycont - terminal UI for Apple's container CLI

Usage:
  lazycont [--help] [--version]

Options:
  --help     Show this help.
  --version  Print the lazycont version.
`)
}

func runTUI(stderr io.Writer) int {
	client := containercli.New("container")
	program := tea.NewProgram(tui.New(client), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "lazycont: %v\n", err)
		return 1
	}
	return 0
}
