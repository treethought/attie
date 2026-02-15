package ui

import (
	"fmt"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

// =============================================================================
// RepoView - Displays repository information and collections
// =============================================================================

var (
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	collectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dimStyle        = lipgloss.NewStyle().Faint(true)
)

type RepoView struct {
	repo *comatproto.RepoDescribeRepo_Output
}

func (r *RepoView) Init() tea.Cmd {
	return nil
}

func (r *RepoView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// RepoView doesn't handle messages directly
	// Parent App handles navigation
	return r, nil
}

func (r *RepoView) View() string {
	if r.repo == nil {
		return "No repository loaded"
	}

	var s strings.Builder

	// Header
	s.WriteString(headerStyle.Render("📦 Repository Information"))
	s.WriteString("\n\n")

	// Repository details
	s.WriteString(labelStyle.Render("Handle: "))
	s.WriteString(valueStyle.Render(r.repo.Handle))
	s.WriteString("\n")

	s.WriteString(labelStyle.Render("DID:    "))
	s.WriteString(valueStyle.Render(r.repo.Did))
	s.WriteString("\n")

	if r.repo.DidDoc != nil {
		s.WriteString(labelStyle.Render("DID Document: "))
		s.WriteString(dimStyle.Render("Available"))
		s.WriteString("\n")
	}

	s.WriteString("\n")

	// Collections section
	s.WriteString(headerStyle.Render("Collections"))
	s.WriteString(fmt.Sprintf(" (%d total)", len(r.repo.Collections)))
	s.WriteString("\n\n")

	if len(r.repo.Collections) == 0 {
		s.WriteString(dimStyle.Render("No collections found"))
	} else {
		for _, collection := range r.repo.Collections {
			s.WriteString("  • ")
			s.WriteString(collectionStyle.Render(collection))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n\n")

	// Footer with help
	s.WriteString(dimStyle.Render("Press Esc to go back • Ctrl+C to quit"))

	return s.String()
}

func (r *RepoView) SetRepo(repo *comatproto.RepoDescribeRepo_Output) {
	r.repo = repo
}
