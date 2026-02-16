package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/treethought/goatie/ui"

	"github.com/bluesky-social/indigo/atproto/identity"
	tea "github.com/charmbracelet/bubbletea"
)

type Client struct {
	dir identity.Directory
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:    false,
		DisableTimestamp: true,
	})
	f, err := os.Create("debug.log")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Warn("starting goatie")

	query := ""
	if len(os.Args) > 1 {
		query = os.Args[1]
	}

	app := ui.NewApp(query)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
