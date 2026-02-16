package ui

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bluesky-social/indigo/api/agnostic"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/treethought/goatie/at"
)

type App struct {
	client     *at.Client
	search     *CommandPallete
	repoView   *RepoView
	rlist      *RecordsList
	recordView *RecordView
	active     tea.Model
	err        string
	w, h       int
}

func NewApp() *App {
	search := &CommandPallete{}
	repoView := NewRepoView()
	return &App{
		client:     at.NewClient(""),
		search:     search,
		repoView:   repoView,
		rlist:      NewRecordsList(nil),
		recordView: NewRecordView(false),
		active:     search,
	}
}

func (a *App) Init() tea.Cmd {
	return a.active.Init()
}

func (a *App) resizeChildren() tea.Cmd {
	cmds := []tea.Cmd{}
	a.search.SetSize(a.w, a.h)
	a.repoView.SetSize(a.w, a.h)
	a.rlist.SetSize(a.w, a.h)
	a.recordView.SetSize(a.w, a.h)
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// top level always handle ctrl-c
	case tea.WindowSizeMsg:
		a.w = msg.Width
		a.h = msg.Height
		return a, a.resizeChildren()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "esc":
			switch a.active {
			case a.repoView:
				a.active = a.search
				a.search.loading = false
				return a, nil
			case a.rlist:
				a.active = a.repoView
				return a, nil
				case a.recordView:
				a.active = a.rlist
				return a, nil
			}
		}

	case searchSubmitMsg:
		// parse for handle/DID or a record URI
		id, err := syntax.ParseAtIdentifier(msg.identifier.String())
		if err != nil {
			log.Fatalf("Failed to parse identifier, should have caught during submission: %s", err.Error())
			return a, nil
		}
		if id.IsDID() || id.IsHandle() {
			log.Printf("Repo identifier submitted: %s", id.String())
			return a, a.fetchRepo(id.String())
		}

	case repoLoadedMsg:
		cmd := a.repoView.SetRepo(msg.repo)
		a.repoView.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.repoView
		a.search.loading = false
		return a, cmd

	case selectCollectionMsg:
		log.Printf("Collection selected: %s", msg.collection)
		return a, a.fetchRecords(msg.collection, a.repoView.repo.Handle)

	case recordsLoadedMsg:
		cmd := a.rlist.SetRecords(msg.records)
		a.rlist.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.rlist
		a.search.loading = false
		return a, cmd

	case recordSelectedMsg:
		a.recordView.SetRecord(msg.record)
		a.recordView.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.recordView
		return a, nil

	case repoErrorMsg:
		a.search.err = msg.err.Error()
		a.search.loading = false
		return a, nil
	}

	var cmd tea.Cmd
	a.active, cmd = a.active.Update(msg)
	return a, cmd
}

func (a *App) fetchRepo(repoId string) tea.Cmd {
	return func() tea.Msg {
		resp, err := a.client.GetRepo(context.Background(), repoId)
		if err != nil {
			log.Printf("Failed to get repo: %s", err.Error())
			return repoErrorMsg{err: err}
		}
		log.WithFields(log.Fields{
			"repo": resp.Handle,
		}).Info("Repo loaded")
		return repoLoadedMsg{repo: resp}
	}
}

func (a *App) fetchRecords(collection, repo string) tea.Cmd {
	return func() tea.Msg {
		recs, err := a.client.ListRecords(context.Background(), collection, repo)
		if err != nil {
			log.Printf("Failed to list records: %s", err.Error())
			return repoErrorMsg{err: err}
		}
		log.WithFields(log.Fields{
			"repo":       repo,
			"collection": collection,
			"numRecords": len(recs),
		}).Info("Records loaded")
		return recordsLoadedMsg{records: recs}
	}
}

func (a *App) View() string {
	return a.active.View()
}

// Message types
type searchSubmitMsg struct {
	identifier syntax.AtIdentifier
}

type repoLoadedMsg struct {
	repo *comatproto.RepoDescribeRepo_Output
}

type selectCollectionMsg struct {
	collection string
}

type recordsLoadedMsg struct {
	records []*agnostic.RepoListRecords_Record
}

type recordSelectedMsg struct {
	record *agnostic.RepoListRecords_Record
}

type repoErrorMsg struct {
	err error
}

type CommandPallete struct {
	ti      textinput.Model
	err     string
	loading bool
	spinner spinner.Model
}

func (c *CommandPallete) Init() tea.Cmd {
	c.ti = textinput.New()
	c.ti.Placeholder = "Enter handle, DID, or AT URI"
	c.ti.Focus()
	c.spinner = spinner.New()
	c.spinner.Spinner = spinner.Dot
	return textinput.Blink
}

func (c *CommandPallete) SetSize(w, h int) {
	c.ti.Width = w - 2
}

func (c *CommandPallete) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := c.ti.Value()
			if val == "" {
				c.err = "Input cannot be empty"
				return c, nil
			}
			id, err := syntax.ParseAtIdentifier(val)
			if err != nil {
				c.err = fmt.Sprintf("Must use handle, DID or AT URI: %s", err.Error())
				return c, nil
			}
			c.err = ""
			c.loading = true
			return c, func() tea.Msg {
				log.Printf("Looking up identifier: %s", id.String())
				return searchSubmitMsg{identifier: id}
			}
		}

	}

	var cmds []tea.Cmd
	ti, tcmd := c.ti.Update(msg)
	c.ti = ti
	cmds = append(cmds, tcmd)

	sp, scmd := c.spinner.Update(msg)
	c.spinner = sp
	cmds = append(cmds, scmd)

	return c, tea.Batch(cmds...)
}

func (c *CommandPallete) View() string {
	s := fmt.Sprint("Search:\n", c.ti.View())
	if c.err != "" {
		s += fmt.Sprintf("\nError: %s", c.err)
	} else if c.loading {
		s += "\nLoading... " + c.spinner.View()
	}
	return s
}
