package ui

import (
	"context"
	"log/slog"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/treethought/attie/at"
)

type AppContext struct {
	identity   *identity.Identity
	repo       *comatproto.RepoDescribeRepo_Output
	collection string
	record     *at.Record
}

type App struct {
	client       *at.Client
	search       *CommandPallete
	repoView     *RepoView
	rlist        *RecordsList
	recordView   *RecordView
	jetEventView *JetStreamEventView
	active       tea.Model
	err          string
	w, h         int
	query        string
	spinner      spinner.Model
	loading      bool
	actx         *AppContext

	jetstream      *JetStreamView
	jetSreamActive bool

	// TODO better nav handling
	// this currently only used for going back from jetstreamevent
	lastView tea.Model
}

func NewApp(query string) *App {
	search := &CommandPallete{}
	repoView := NewRepoView()
	spin := spinner.New()
	spin.Spinner = spinner.Dot

	jc := at.NewJetstreamClient()
	jv := NewJetStreamView(jc)
	return &App{
		query:        query,
		client:       at.NewClient(""),
		search:       search,
		repoView:     repoView,
		rlist:        NewRecordsList(nil),
		recordView:   NewRecordView(false),
		jetEventView: NewJetEventView(false),
		active:       search,
		spinner:      spin,
		loading:      false,
		actx:         &AppContext{},
		jetstream:    jv,
	}
}

func (a *App) Init() tea.Cmd {
	a.loading = true
	if id, err := syntax.ParseAtIdentifier(a.query); err == nil {
		slog.Info("Starting with query", "id", id.String())
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

		slog.Info("Starting with query", "uri", uri.String())
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
	a.jetstream.SetSize(a.w, a.h)
	a.jetEventView.SetSize(a.w, a.h)
	return tea.Batch(cmds...)
}

func (a *App) resetToSearch() tea.Cmd {
	a.actx.identity = nil
	a.actx.repo = nil
	a.actx.collection = ""
	a.actx.record = nil
	a.active = a.search
	a.loading = false
	return a.search.Init()
}

func (a *App) setJetStreamActive(active bool) tea.Cmd {
	if active {
		a.jetEventView.SetEvent(nil)
		a.lastView = a.active
		a.jetSreamActive = true
		a.jetstream.SetSize(a.w, a.h)
		if a.jetstream.Running() {
			// pause but keep view and items visisble
			return a.jetstream.Stop()
		}

		cxs := []string{}
		dids := []string{}
		if a.actx.collection != "" {
			cxs = append(cxs, a.actx.collection)
		}
		if a.actx.identity != nil {
			dids = append(dids, a.actx.identity.DID.String())
		}
		return a.jetstream.Start(cxs, dids, nil)
	}

	a.jetSreamActive = false
	// clear event view
	a.jetEventView.SetEvent(nil)
	cmds := []tea.Cmd{
		a.jetstream.Stop(),
		a.jetstream.Clear(),
	}
	if a.lastView != nil {
		a.active = a.lastView
	} else {
		cmds = append(cmds, a.resetToSearch())
	}
	return tea.Sequence(
		cmds...,
	)
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
			// keep jetstream active, and stop on search submit
			a.jetSreamActive = false
			a.active = a.search
			a.search.loading = false
			return a, a.search.Init()
		case "ctrl+j":
			return a, a.setJetStreamActive(true)
		case "esc":
			if a.jetSreamActive {
				return a, a.setJetStreamActive(false)
			}
			switch a.active {
			case a.repoView:
				return a, a.resetToSearch()
			case a.rlist:
				if a.actx.identity == nil {
					return a, a.resetToSearch()
				}
				if a.actx.repo != nil {
					a.active = a.repoView
					return a, a.repoView.Init()
				}
				a.active = a.repoView
				return a, a.fetchRepo(a.actx.identity.DID.String())
			case a.recordView:
				if a.actx.identity == nil {
					return a, a.resetToSearch()
				}
				if a.actx.collection != "" {
					a.active = a.rlist
					return a, a.fetchRecords(a.actx.collection, a.actx.identity.DID.String())
				}
				a.active = a.rlist
				return a, nil
			case a.jetEventView:
				return a, a.setJetStreamActive(true)
			}
		}

	case searchSubmitMsg:
		// parse for handle/DID or a record URI
		id, err := syntax.ParseAtIdentifier(msg.identifier.String())
		if err != nil {
			slog.Error("Failed to parse identifier, should have caught during submission", "error", err)
			return a, nil
		}
		if id.IsDID() || id.IsHandle() {
			return a,
				tea.Sequence(
					a.setJetStreamActive(false),
					a.fetchRepo(id.String()),
				)
		}

	case repoLoadedMsg:
		a.loading = false
		a.actx.identity = msg.repo.Identity
		a.actx.repo = msg.repo.Repo
		a.actx.collection = ""
		a.actx.record = nil
		cmd := a.repoView.SetRepo(msg.repo)
		a.repoView.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.repoView
		a.search.loading = false
		return a, cmd

	case selectCollectionMsg:
		slog.Info("Collection selected", "collection", msg.collection)
		a.actx.collection = msg.collection
		return a, a.fetchRecords(msg.collection, a.repoView.repo.Handle)

	case recordsLoadedMsg:
		a.loading = false
		a.actx.identity = msg.records.Identity
		a.actx.collection = msg.records.Collection()
		a.actx.record = nil
		cmd := a.rlist.SetRecords(msg.records.Records)
		a.rlist.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.rlist
		a.search.loading = false
		return a, cmd

	case recordSelectedMsg:
		a.loading = false
		a.actx.identity = msg.record.Identity
		a.actx.collection = msg.record.Record.Collection()
		a.actx.record = msg.record.Record
		a.recordView.SetRecord(msg.record.Record)
		a.recordView.SetSize(a.w, a.h) // Set size before switching view
		a.active = a.recordView
		return a, nil

	case jetEventSelectedMsg:
		a.jetEventView.SetEvent(msg.evt)
		a.jetEventView.SetSize(a.w, a.h)
		a.active = a.jetEventView
		a.jetSreamActive = false
		return a, nil

	case repoErrorMsg:
		a.search.err = msg.err.Error()
		a.search.loading = false
		return a, nil
	}

	if a.jetSreamActive {
		_, cmd := a.jetstream.Update(msg)
		return a, cmd
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
		slog.Info("Fetching repo", "repoId", repoId)
		resp, err := a.client.GetRepo(context.Background(), repoId)
		if err != nil {
			slog.Error("Failed to get repo", "error", err)
			return repoErrorMsg{err: err}
		}
		slog.Info("Repo loaded", "repo", resp.Repo.Handle)
		return repoLoadedMsg{repo: resp}
	}
}

func (a *App) fetchRecords(collection, repo string) tea.Cmd {
	return func() tea.Msg {
		recs, err := a.client.ListRecords(context.Background(), collection, repo)
		if err != nil {
			slog.Error("Failed to list records", "error", err)
			return repoErrorMsg{err: err}
		}
		slog.Info("Records loaded", "repo", repo, "collection", collection, "numRecords", len(recs.Records))
		return recordsLoadedMsg{records: recs}
	}
}

func (a *App) fetchRecord(collection, repo, rkey string) tea.Cmd {
	return func() tea.Msg {
		rec, err := a.client.GetRecord(context.Background(), collection, repo, rkey)
		if err != nil {
			slog.Error("Failed to get record", "error", err)
			return repoErrorMsg{err: err}
		}
		slog.Info("Record loaded", "repo", repo, "collection", collection, "rkey", rkey)
		return recordSelectedMsg{
			record: rec,
		}
	}
}

func (a *App) View() string {
	if a.loading {
		return "Loading... " + a.spinner.View()
	}
	if a.jetSreamActive {
		return a.jetstream.View()
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
	records *at.RecordsWithIdentity
}

type recordSelectedMsg struct {
	record *at.RecordWithIdentity
}

type repoErrorMsg struct {
	err error
}
