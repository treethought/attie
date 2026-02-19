package ui

import (
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/treethought/attie/at"
)

type RecordsList struct {
	rlist      list.Model
	preview    *RecordView
	header     string
	w, h       int
	collection string
}

type RecordListItem struct {
	r      *at.Record
	parsed syntax.ATURI
}

func NewRecordListItem(r *at.Record) RecordListItem {
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

func NewRecordsList(records []*at.Record) *RecordsList {
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
		preview: NewRecordView(true),
	}
	rl.SetRecords(records)
	return rl
}

func (rl *RecordsList) SetRecords(records []*at.Record) tea.Cmd {
	if records == nil {
		return nil
	}
	rl.preview.SetRecord(nil)
	rl.rlist.SetItems(nil)
	items := make([]list.Item, len(records))
	for i, rec := range records {
		ci := NewRecordListItem(rec)
		items[i] = list.Item(ci)
	}
	cmd := rl.rlist.SetItems(items)
	if len(items) > 0 {
		rl.preview.SetRecord(items[0].(RecordListItem).r)
	}
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := rl.rlist.SelectedItem().(RecordListItem); ok {
				return rl, func() tea.Msg {
					return recordSelectedMsg{
						record: &at.RecordWithIdentity{
							Record: item.r,
						},
					}
				}
			}
		}
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
