package ui

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/treethought/goatie/at"
)

type App struct {
	client     *at.Client
	search     *CommandPallete
	identity   identity.Identity
	repoView   *RepoView
	rlist      *RecordsList
	recordView *RecordView
	active     tea.Model
	err        string
	w, h       int
	query      string
	spinner    spinner.Model
	loading    bool
}

func NewApp(query string) *App {
	search := &CommandPallete{}
	repoView := NewRepoView()
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	return &App{
		query:      query,
		client:     at.NewClient(""),
		search:     search,
		repoView:   repoView,
		rlist:      NewRecordsList(nil),
		recordView: NewRecordView(false),
		active:     search,
		spinner:    spin,
		loading:    false,
	}
}

func (a *App) Init() tea.Cmd {
	a.loading = true
	if id, err := syntax.ParseAtIdentifier(a.query); err == nil {
		log.Printf("Starting with query: %s", id.String())
		return a.fetchRepo(id.String())
	}
	if uri, err := syntax.ParseATURI(a.query); err == nil {

		if uri.Collection() == "" {
			return a.fetchRepo(uri.Authority().String())
		}
		if uri.RecordKey().String() == "" {
			id := uri.Authority().Handle().String()
			if uri.Authority().IsDID() {
				id = uri.Authority().DID().String()
			}
			return a.fetchRecords(uri.Collection().String(), id)
		}

		log.Printf("Starting with query: %s", uri.String())
		return a.fetchRecord(uri.Collection().String(), uri.Authority().String(), uri.RecordKey().String())
	}

	a.loading = false
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
		case "ctrl+k":
			a.active = a.search
			a.search.loading = false
			return a, a.search.Init()
		case "esc":
			switch a.active {
			case a.repoView:
				a.active = a.search
				a.search.loading = false
				return a, a.search.Init()
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
		a.loading = false
		cmd := a.repoView.SetRepo(msg.repo)
		a.repoView.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.repoView
		a.search.loading = false
		return a, cmd

	case selectCollectionMsg:
		log.Printf("Collection selected: %s", msg.collection)
		return a, a.fetchRecords(msg.collection, a.repoView.repo.Handle)

	case recordsLoadedMsg:
		a.loading = false
		cmd := a.rlist.SetRecords(msg.records)
		a.rlist.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.rlist
		a.search.loading = false
		return a, cmd

	case recordSelectedMsg:
		a.loading = false
		a.recordView.SetRecord(msg.record)
		a.recordView.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.recordView
		return a, nil

	case repoErrorMsg:
		a.search.err = msg.err.Error()
		a.search.loading = false
		return a, nil
	}

	var cmds []tea.Cmd
	if a.loading {
		sp, scmd := a.spinner.Update(msg)
		a.spinner = sp
		cmds = append(cmds, scmd)
	}
	var ac tea.Cmd
	a.active, ac = a.active.Update(msg)
	cmds = append(cmds, ac)
	return a, tea.Batch(cmds...)
}

func (a *App) fetchRepo(repoId string) tea.Cmd {
	return func() tea.Msg {
		resp, err := a.client.GetRepo(context.Background(), repoId)
		if err != nil {
			log.Printf("Failed to get repo: %s", err.Error())
			return repoErrorMsg{err: err}
		}
		log.WithFields(log.Fields{
			"repo": resp.Repo.Handle,
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

func (a *App) fetchRecord(collection, repo, rkey string) tea.Cmd {
	return func() tea.Msg {
		rec, err := a.client.GetRecord(context.Background(), collection, repo, rkey)
		if err != nil {
			log.Printf("Failed to get record: %s", err.Error())
			return repoErrorMsg{err: err}
		}
		log.WithFields(log.Fields{
			"repo":       repo,
			"collection": collection,
			"rkey":       rkey,
		}).Info("Record loaded")
		return recordSelectedMsg{
			record: &agnostic.RepoListRecords_Record{
				Uri:   rec.Uri,
				Value: rec.Value,
			}}
	}
}

func (a *App) View() string {
	if a.loading {
		return "Loading... " + a.spinner.View()
	}
	return a.active.View()
}

// Message types
type searchSubmitMsg struct {
	identifier syntax.AtIdentifier
}

type repoLoadedMsg struct {
	repo *at.RepoWithIdentity
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

