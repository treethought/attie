package ui

import (
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/treethought/goatie/at"
)

type RecordView struct {
	record  *at.Record
	vp      viewport.Model
	header  string
	preview bool
}

func NewRecordView(preview bool) *RecordView {
	vp := viewport.New(80, 20)
	return &RecordView{
		vp:      vp,
		preview: preview,
	}
}

func (rv *RecordView) SetSize(w, h int) {
	rv.vp.Width = w
	rv.vp.Height = h - lipgloss.Height(rv.header)
}

func (rv *RecordView) buildHeader() string {
	if rv.record == nil {
		return ""
	}
	uri, err := syntax.ParseATURI(rv.record.Uri)
	if err != nil {
		return headerStyle.Render(rv.record.Uri)
	}
	header := rv.record.Uri
	if rv.preview {
		header = fmt.Sprintf("%s/%s", uri.Collection(), uri.RecordKey().String())
	}
	return headerStyle.Render(header)
}

func (rv *RecordView) SetRecord(record *at.Record) {
	rv.record = record
	if rv.record == nil || rv.record.Value == nil {
		rv.vp.SetContent("")
		rv.header = ""
		return
	}
	data, err := json.MarshalIndent(rv.record.Value, "", "  ")
	if err != nil {
		data = fmt.Appendf([]byte{}, "error marshaling record: %v", err)
	}
	rv.vp.SetContent(string(data))
	rv.header = rv.buildHeader()
}

func (rv *RecordView) Init() tea.Cmd {
	return rv.vp.Init()
}

func (rv *RecordView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	rv.vp, cmd = rv.vp.Update(msg)
	return rv, cmd
}

func (rv *RecordView) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, rv.header, rv.vp.View())
}
