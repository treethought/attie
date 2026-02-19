package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContentView wraps a scrollable viewport with a header line.
type ContentView struct {
	vp      viewport.Model
	preview bool
	header  string
	empty   bool
}

func newContentView(preview bool) ContentView {
	return ContentView{
		vp:      viewport.New(80, 20),
		preview: preview,
		empty:   true,
	}
}

func (v *ContentView) Set(header, content string) {
	if header == "" && content == "" {
		v.empty = true
		v.header = ""
		v.vp.SetContent("")
		return
	}
	v.empty = false
	v.header = header
	v.vp.SetContent(content)
}

func (v *ContentView) SetSize(w, h int) {
	v.vp.Width = w
	v.vp.Height = h - lipgloss.Height(v.header)
}

func (v *ContentView) initVP() tea.Cmd {
	return v.vp.Init()
}

func (v *ContentView) updateVP(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.vp, cmd = v.vp.Update(msg)
	return cmd
}

func (v *ContentView) renderVP() string {
	if v.empty {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, v.header, v.vp.View())
}
