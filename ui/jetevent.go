package ui

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/jetstream/pkg/models"
	tea "github.com/charmbracelet/bubbletea"
)

type jetEventSelectedMsg struct {
	evt *models.Event
}

type JetStreamEventView struct {
	ContentView
	evt *models.Event
}

func NewJetEventView(preview bool) *JetStreamEventView {
	return &JetStreamEventView{ContentView: newContentView(preview)}
}

func (v *JetStreamEventView) buildHeader() string {
	if v.evt == nil {
		return ""
	}
	if v.preview {
		return headerStyle.Render(fmt.Sprintf("%s  %s  %s",
			opStyle.Render(string(v.evt.Commit.Operation)),
			v.evt.Commit.Collection,
			dimStyle.Render(v.evt.Commit.RKey),
		))
	}
	t := time.Unix(0, v.evt.TimeUS*int64(time.Microsecond))
	return headerStyle.Render(fmt.Sprintf("%s  %s/%s  %s  %s",
		didStyle.Render(v.evt.Did),
		v.evt.Commit.Collection,
		v.evt.Commit.RKey,
		opStyle.Render(string(v.evt.Commit.Operation)),
		dimStyle.Render(t.Format("2006-01-02 15:04:05")),
	))
}

func (v *JetStreamEventView) SetEvent(evt *models.Event) {
	v.evt = evt
	if evt == nil {
		v.Set("", "")
		return
	}
	data, err := json.MarshalIndent(evt, "", "  ")
	if err != nil {
		data = fmt.Appendf([]byte{}, "error marshaling event: %v", err)
	}
	v.Set(v.buildHeader(), string(data))
}

func (v *JetStreamEventView) Init() tea.Cmd {
	return v.initVP()
}
func (v *JetStreamEventView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return v, v.updateVP(msg)
}
func (v *JetStreamEventView) View() string {
	return v.renderVP()
}
