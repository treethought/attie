package main

import (
	"log/slog"
	"os"

	"github.com/treethought/attie/ui"

	"github.com/bluesky-social/indigo/atproto/identity"
	tea "github.com/charmbracelet/bubbletea"
)

type Client struct {
	dir identity.Directory
}

func main() {
	f, err := os.Create("/tmp/attie.log")
	if err != nil {
		slog.Error("failed to create log file", "error", err)
		os.Exit(1)
	}
	defer f.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
	slog.Info("starting attie")

	query := ""
	if len(os.Args) > 1 {
		query = os.Args[1]
	}

	app := ui.NewApp(query)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("program error", "error", err)
	}
}
