package ui

import (
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/treethought/attie/at"
)

type RecordView struct {
	ContentView
	record *at.Record
}

func NewRecordView(preview bool) *RecordView {
	return &RecordView{ContentView: newContentView(preview)}
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
	if record == nil || record.Value == nil {
		rv.Set("", "")
		return
	}
	data, err := json.MarshalIndent(record.Value, "", "  ")
	if err != nil {
		data = fmt.Appendf([]byte{}, "error marshaling record: %v", err)
	}
	rv.Set(rv.buildHeader(), string(data))
}

func (rv *RecordView) Init() tea.Cmd {
	return rv.initVP()
}
func (rv *RecordView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return rv, rv.updateVP(msg)
}
func (rv *RecordView) View() string {
	return rv.renderVP()
}
