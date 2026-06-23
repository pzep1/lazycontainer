package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pz/lazycont/internal/containercli"
	"github.com/pz/lazycont/internal/tui"
)

func main() {
	client := containercli.New("container")
	program := tea.NewProgram(tui.New(client), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazycont: %v\n", err)
		os.Exit(1)
	}
}
