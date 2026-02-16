package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RecordView struct {
	record *agnostic.RepoListRecords_Record
	vp     viewport.Model
}

func NewRecordView() *RecordView {
	vp := viewport.New(80, 20)
	return &RecordView{
		vp: vp,
	}
}

func (rv *RecordView) SetSize(w, h int) {
	rv.vp.Width = w
	rv.vp.Height = h
}

func (rv *RecordView) SetRecord(record *agnostic.RepoListRecords_Record) {
	rv.record = record
	if rv.record == nil || rv.record.Value == nil {
		rv.vp.SetContent("")
		return
	}
	data, err := json.MarshalIndent(rv.record.Value, "", "  ")
	if err != nil {
		data = fmt.Appendf([]byte{}, "error marshaling record: %v", err)
	}
	rv.vp.SetContent(string(data))
}

func (rv *RecordView) Init() tea.Cmd {
	return nil
}

func (rv *RecordView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	rv.vp, cmd = rv.vp.Update(msg)
	return rv, cmd
}

func (rv *RecordView) View() string {
	return rv.vp.View()
}

type RecordsList struct {
	rlist   list.Model
	preview RecordView
	header  string
	w, h    int
}

type RecordListItem struct {
	r      *agnostic.RepoListRecords_Record
	parsed syntax.ATURI
}

func NewRecordListItem(r *agnostic.RepoListRecords_Record) RecordListItem {
	uri, _ := syntax.ParseATURI(r.Uri)
	return RecordListItem{
		r:      r,
		parsed: uri,
	}
}

func (r RecordListItem) FilterValue() string {
	return r.parsed.RecordKey().String()
}
func (r RecordListItem) Title() string {
	return r.parsed.RecordKey().String()
}
func (r RecordListItem) Description() string {
	return truncMiddle(r.r.Cid, 24)
}

func truncMiddle(s string, max int) string {
	if len(s) <= max {
		return s
	}
	half := max / 2
	return s[:half] + "..." + s[len(s)-half:]
}

func NewRecordsList(records []*agnostic.RepoListRecords_Record) *RecordsList {
	del := list.DefaultDelegate{
		ShowDescription: true,
		Styles:          list.NewDefaultItemStyles(),
	}
	del.SetHeight(2)

	l := list.New(nil, del, 80, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	rl := &RecordsList{
		rlist:   l,
		preview: RecordView{},
	}
	rl.SetRecords(records)
	return rl
}

func (rl *RecordsList) SetRecords(records []*agnostic.RepoListRecords_Record) tea.Cmd {
	rl.preview.SetRecord(nil)
	rl.rlist.SetItems(nil)
	items := make([]list.Item, len(records))
	for i, rec := range records {
		ci := NewRecordListItem(rec)
		items[i] = list.Item(ci)
	}
	cmd := rl.rlist.SetItems(items)
	rl.header = rl.buildHeader()
	return cmd
}

func (rl *RecordsList) buildHeader() string {
	// TODO pass collection into model and fetch in init
	// for now just use the first record's collection for header
	if len(rl.rlist.Items()) == 0 {
		return "No Records"
	}
	rec, ok := rl.rlist.Items()[0].(RecordListItem)
	if !ok {
		return "Records"
	}
	uri, err := syntax.ParseATURI(rec.r.Uri)
	if err != nil {
		return "Records"
	}
	s := strings.Builder{}
	s.WriteString(uri.Collection().String())
	s.WriteString(" - ")
	s.WriteString(fmt.Sprintf("%d records", len(rl.rlist.Items())))
	return lipgloss.NewStyle().Bold(true).Render(s.String())
}

func (rl *RecordsList) SetSize(w, h int) {
	rl.w = w
	rl.h = h
	headerHeight := lipgloss.Height(rl.header)
	if rl.w > 100 {
		rl.rlist.SetSize(rl.w/2, rl.h-headerHeight)
		rl.preview.SetSize(rl.w/2, rl.h-headerHeight)
		return
	}
	rl.rlist.SetSize(rl.w, rl.h-headerHeight)
	rl.preview.SetSize(0, 0)
}

func (m *RecordsList) Init() tea.Cmd {
	return nil
}

func (rl *RecordsList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	rl.rlist, cmd = rl.rlist.Update(msg)
	if item, ok := rl.rlist.SelectedItem().(RecordListItem); ok {
		rl.preview.SetRecord(item.r)
	}
	return rl, cmd
}

func (rl *RecordsList) View() string {
	if rl.w > 100 {
		return lipgloss.JoinVertical(lipgloss.Left, rl.header, lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left, rl.header, rl.rlist.View()),
			rl.preview.View(),
		))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rl.header, rl.rlist.View())
}
