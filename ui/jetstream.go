package ui

import (
	"context"
	"fmt"
	"log/slog"

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
		j.evt.Commit.Operation, j.evt.Commit.Collection, j.evt.Commit.RKey,
	)
}

func (j jetEventItem) Description() string {
	return fmt.Sprintf("%s - %s", j.evt.Did, j.evt.TimeUS)
}

type eventMsg struct {
	evt *models.Event
}

type jetStreamErrorMsg struct {
	err error
}

type JetStreamView struct {
	list   list.Model
	jc     *at.JetStreamClient
	ctx    context.Context
	cancel context.CancelFunc
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
		list: l,
		jc:   jc,
	}
}

func (m *JetStreamView) Listen() tea.Cmd {
	return func() tea.Msg {
		slog.Info("Listening for JetStream events...")
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
	item := jetEventItem{evt: evt}
	return m.list.InsertItem(0, item)
}

func (m *JetStreamView) Running() bool {
	return m.ctx != nil
}
func (m *JetStreamView) Clear() tea.Cmd {
	return func() tea.Msg {
		m.list.SetItems(nil)
		return nil
	}
}

func (m *JetStreamView) Start(cxs, dids []string, cursor *int64) tea.Cmd {
	if m.ctx != nil {
		slog.Warn("JetStream client already running")
		return nil
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
	m.list.SetSize(w, h)
}

func (m *JetStreamView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// case jetStreamStartMsg:
	// 	return m, m.Start(msg.cxs, msg.dids, msg.cur)
	//
	// case jetStreamStopMsg:
	// 	m.Stop()
	// 	return m, nil

	case jetStreamErrorMsg:
		slog.Error("JetStream client error", "error", msg.err)
		return m, nil

	case eventMsg:
		return m, tea.Batch(
			m.AddEvent(msg.evt),
			m.Listen(),
		)
	}
	l, cmd := m.list.Update(msg)
	m.list = l
	return m, cmd
}

func (m *JetStreamView) View() string {
	if m.ctx == nil {
		return dimStyle.Render("JetStream client not running")
	}
	return m.list.View()
}
