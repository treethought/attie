package ui

import (
	"fmt"
	"strings"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/treethought/attie/at"
)

var (
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	collectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dimStyle        = lipgloss.NewStyle().Faint(true)
)

type CollectionList struct {
	list list.Model
}

type CollectionListItem struct {
	Name string
}

func (c CollectionListItem) FilterValue() string {
	return c.Name
}
func (c CollectionListItem) Title() string {
	return c.Name
	// return collectionStyle.Render(c.Name)
}
func (c CollectionListItem) Description() string {
	return ""
}

func NewCollectionList(collections []string) *CollectionList {
	items := make([]list.Item, len(collections))
	for i, col := range collections {
		ci := CollectionListItem{Name: col}
		items[i] = list.Item(ci)
	}
	del := list.DefaultDelegate{
		ShowDescription: false,
		Styles:          list.NewDefaultItemStyles(),
	}
	del.SetHeight(1)

	l := list.New(items, del, 80, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return &CollectionList{
		list: l,
	}
}

func (cl *CollectionList) Init() tea.Cmd {
	return nil
}

func (cl *CollectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	if !cl.list.SettingFilter() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if item, ok := cl.list.SelectedItem().(CollectionListItem); ok {
					return cl, func() tea.Msg {
						return selectCollectionMsg{collection: item.Name}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	cl.list, cmd = cl.list.Update(msg)
	return cl, cmd
}

func (cl *CollectionList) View() string {
	if len(cl.list.Items()) == 0 {
		return dimStyle.Render("No collections found")
	}
	return cl.list.View()
}

type RepoView struct {
	identity *identity.Identity
	repo     *comatproto.RepoDescribeRepo_Output
	clist    *CollectionList
	header   string
	width    int
	height   int
}

func NewRepoView() *RepoView {
	return &RepoView{
		clist:  NewCollectionList([]string{}),
		width:  80,
		height: 24,
	}
}

func (r *RepoView) buildHeader() string {
	if r.repo == nil {
		return ""
	}
	var s strings.Builder

	s.WriteString(headerStyle.Render("📦 Repository"))
	s.WriteString("\n\n")

	s.WriteString(labelStyle.Render("Handle: "))
	s.WriteString(valueStyle.Render(r.repo.Handle))
	s.WriteString("  ")
	s.WriteString(labelStyle.Render("Valid: "))
	if r.repo.HandleIsCorrect {
		s.WriteString(valueStyle.Render("✓"))
	} else {
		s.WriteString(dimStyle.Render("✗"))
	}
	s.WriteString("\n")

	s.WriteString(labelStyle.Render("DID:    "))
	s.WriteString(dimStyle.Render(r.repo.Did))
	s.WriteString("\n\n")

	s.WriteString(labelStyle.Render("PDS:  "))
	s.WriteString(valueStyle.Render(r.identity.PDSEndpoint()))
	s.WriteString("\n")

	// Collections section header
	s.WriteString(headerStyle.Render("Collections "))
	s.WriteString(dimStyle.Render(fmt.Sprintf("(%d)", len(r.repo.Collections))))
	s.WriteString("\n")

	// add bottom border
	return lipgloss.NewStyle().BorderBottom(true).Render(s.String())

}

func (r *RepoView) SetRepo(repo *at.RepoWithIdentity) tea.Cmd {
	r.identity = repo.Identity
	r.repo = repo.Repo
	r.header = r.buildHeader()
	r.clist = NewCollectionList(repo.Repo.Collections)
	return r.clist.Init()
}

func (r *RepoView) Init() tea.Cmd {
	return nil
}

func (r *RepoView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	clist, cmd := r.clist.Update(msg)
	r.clist = clist.(*CollectionList)
	return r, cmd
}

// updateListSize calculates and sets the list size to fill remaining space
func (r *RepoView) SetSize(w, h int) {
	r.width = w
	r.height = h
	if r.clist == nil {
		return
	}
	headerHeight := lipgloss.Height(r.header)
	footerHeight := 2 // "\n" + help text line

	// List gets all remaining space
	listHeight := r.height - headerHeight - footerHeight
	if listHeight < 5 {
		listHeight = 5
	}

	r.clist.list.SetSize(r.width, listHeight)
}

func (r *RepoView) View() string {
	if r.repo == nil {
		return "No repository loaded"
	}

	// Footer help text
	footer := dimStyle.Render("Press Esc to go back • ↑/↓ or j/k to navigate • Ctrl+C to quit")

	// Join header (fixed), list (scrollable), and footer
	return lipgloss.JoinVertical(
		lipgloss.Left,
		r.header,
		r.clist.View(),
		"\n"+footer,
	)
}
