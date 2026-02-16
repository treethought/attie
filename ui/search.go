package ui

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CommandPallete struct {
	ti      textinput.Model
	err     string
	loading bool
	spinner spinner.Model
	width   int
	height  int
}

func (c *CommandPallete) Init() tea.Cmd {
	c.ti = textinput.New()
	c.ti.Placeholder = "Enter handle, DID, or AT URI"
	c.ti.Focus()
	c.ti.Width = 60
	c.spinner = spinner.New()
	c.spinner.Spinner = spinner.Dot
	return textinput.Blink
}

func (c *CommandPallete) SetSize(w, h int) {
	c.width = w
	c.height = h

	// Account for border (2 chars) + padding (4 chars) = 6 total
	maxWidth := 128
	availableWidth := w - 6

	if availableWidth < maxWidth {
		if availableWidth < 20 {
			availableWidth = 20 // Minimum width
		}
		c.ti.Width = availableWidth
	} else {
		c.ti.Width = maxWidth
	}
}

func (c *CommandPallete) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := c.ti.Value()
			if val == "" {
				c.err = "Input cannot be empty"
				return c, nil
			}
			id, err := syntax.ParseAtIdentifier(val)
			if err != nil {
				c.err = fmt.Sprintf("Must use handle, DID or AT URI: %s", err.Error())
				return c, nil
			}
			c.err = ""
			c.loading = true
			return c, func() tea.Msg {
				log.Printf("Looking up identifier: %s", id.String())
				return searchSubmitMsg{identifier: id}
			}
		}

	}

	var cmds []tea.Cmd
	ti, tcmd := c.ti.Update(msg)
	c.ti = ti
	cmds = append(cmds, tcmd)

	sp, scmd := c.spinner.Update(msg)
	c.spinner = sp
	cmds = append(cmds, scmd)

	return c, tea.Batch(cmds...)
}

var searchStyle = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))

func (c *CommandPallete) View() string {
	// make centered search box
	s := c.ti.View()
	if c.err != "" {
		s += fmt.Sprintf("\nError: %s", c.err)
	} else if c.loading {
		s += "\nLoading... " + c.spinner.View()
	}

	// Center the box in the full terminal window
	box := searchStyle.Render(s)
	return lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Center, box)
}
