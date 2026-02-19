package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/treethought/attie/at"
)

var (
	opStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	didStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

type jetEventItem struct {
	evt *models.Event
}

func (j jetEventItem) FilterValue() string {
	return ""
}
func (j jetEventItem) Title() string {
	return fmt.Sprintf("%s %s %s",
		opStyle.Render(j.evt.Commit.Operation), j.evt.Commit.Collection, dimStyle.Render(j.evt.Commit.RKey),
	)
}

func (j jetEventItem) Description() string {
	t := time.Unix(0, j.evt.TimeUS*int64(time.Microsecond))
	return fmt.Sprintf("%s - %s", didStyle.Render(j.evt.Did), t.Format("2006-01-02 15:04:05"))
}

type eventMsg struct {
	evt *models.Event
}

type jetStreamErrorMsg struct {
	err error
}

type session struct {
	lastCursor  *int64
	collections []string
	dids        []string
}

type JetStreamView struct {
	list    list.Model
	preview *JetStreamEventView
	jc      *at.JetStreamClient
	ctx     context.Context
	cancel  context.CancelFunc
	session session
	w, h    int
}

func NewJetStreamView(jc *at.JetStreamClient) *JetStreamView {
	del := list.DefaultDelegate{
		ShowDescription: true,
		Styles:          list.NewDefaultItemStyles(),
	}
	del.SetHeight(2)

	l := list.New(nil, del, 80, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return &JetStreamView{
		list:    l,
		preview: NewJetEventView(true),
		jc:      jc,
	}
}

func (m *JetStreamView) Listen() tea.Cmd {
	return func() tea.Msg {
		select {
		case err := <-m.jc.Err():
			slog.Error("JetStream client error", "error", err)
			return jetStreamErrorMsg{err: err}
		case evt := <-m.jc.Out():
			slog.Info("Received JetStream event", "did", evt.Did, "kind", evt.Kind)
			return eventMsg{evt: evt}
		}
	}
}

func (m *JetStreamView) AddEvent(evt *models.Event) tea.Cmd {
	m.session.lastCursor = &evt.TimeUS
	item := jetEventItem{evt: evt}
	return m.list.InsertItem(0, item)
}

func (m *JetStreamView) Running() bool {
	return m.ctx != nil
}
func (m *JetStreamView) Clear() tea.Cmd {
	return func() tea.Msg {
		m.session = session{}
		m.preview.SetEvent(nil)
		return m.list.SetItems(nil)
	}
}

func (m *JetStreamView) Start(cxs, dids []string, cursor *int64) tea.Cmd {
	if m.ctx != nil {
		slog.Warn("JetStream client already running")
		return nil
	}
	m.session = session{
		lastCursor:  cursor,
		collections: cxs,
		dids:        dids,
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	slog.Info("Starting JetStream client", "collections", cxs, "dids", dids, "cursor", cursor)
	go m.jc.Start(m.ctx, cxs, dids, cursor)
	return m.Listen()
}
func (m *JetStreamView) Stop() tea.Cmd {
	if m.cancel != nil {
		slog.Info("Stopping JetStream client")
		m.cancel()
		m.ctx = nil
	}
	return nil
}

func (m *JetStreamView) Init() tea.Cmd {
	return nil
}
func (m *JetStreamView) SetSize(w, h int) {
	m.w = w
	m.h = h
	hh := lipgloss.Height(m.header())
	if m.ctx == nil {
		hh += 1
	}
	if w > 100 {
		m.list.SetSize(w/2, h-hh)
		m.preview.SetSize(w/2, h-hh)
		return
	}
	m.list.SetSize(w, h-hh)
	m.preview.SetSize(0, 0)
}

func (m *JetStreamView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case jetStreamErrorMsg:
		slog.Error("JetStream client error", "error", msg.err)
		return m, nil

	case eventMsg:
		return m, tea.Batch(
			m.AddEvent(msg.evt),
			m.Listen(),
		)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if item, ok := m.list.SelectedItem().(jetEventItem); ok {
				return m, func() tea.Msg {
					return jetEventSelectedMsg{evt: item.evt}
				}
			}
		}
	}

	l, cmd := m.list.Update(msg)
	m.list = l
	if item, ok := m.list.SelectedItem().(jetEventItem); ok {
		m.preview.SetEvent(item.evt)
	}
	return m, cmd
}

var jetstreamTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205")).
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	BorderForeground(lipgloss.Color("62")).
	PaddingLeft(1)

func (m *JetStreamView) header() string {
	cxs := dimStyle.Render("all")
	if len(m.session.collections) > 0 {
		cxs = strings.Join(m.session.collections, ", ")
	}
	dids := dimStyle.Render("all")
	if len(m.session.dids) > 0 {
		dids = strings.Join(m.session.dids, ", ")
	}
	lastCursor := dimStyle.Render("live")
	if m.session.lastCursor != nil {
		lastCursor = fmt.Sprintf("%d", *m.session.lastCursor)
	}

	title := jetstreamTitleStyle.Render("📡  JetStream Events")

	dot := dimStyle.Render("  ·  ")
	filters := lipgloss.JoinHorizontal(lipgloss.Left,
		dimStyle.Render(" collections: "), cxs,
		dot, dimStyle.Render("dids: "), dids,
		dot, dimStyle.Render("cursor: "), lastCursor,
	)

	return lipgloss.JoinVertical(lipgloss.Left, title, filters)
}

func (m *JetStreamView) View() string {
	hdr := m.header()
	status := ""
	if m.ctx == nil {
		status = dimStyle.Render("  not connected  ·  press ctrl+j to start")
	}

	if m.w > 100 {
		left := lipgloss.JoinVertical(lipgloss.Left, m.list.View())
		right := m.preview.View()
		body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		if status != "" {
			return lipgloss.JoinVertical(lipgloss.Left, hdr, status, body)
		}
		return lipgloss.JoinVertical(lipgloss.Left, hdr, body)
	}

	if status != "" {
		return lipgloss.JoinVertical(lipgloss.Left, hdr, status, m.list.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, hdr, m.list.View())
}
