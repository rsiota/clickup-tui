package main

import (
	"flag"
	"fmt"
	"os"

	"clickup-tui/internal/clickup"
	"clickup-tui/internal/config"
	"clickup-tui/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	token := flag.String("token", "", "ClickUp API token")
	workspace := flag.String("workspace", "", "Workspace ID")
	flag.Parse()

	if flag.NArg() > 0 && flag.Arg(0) == "init" {
		if err := config.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "init: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load(*token, *workspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(clickup.New(cfg.Token), cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
