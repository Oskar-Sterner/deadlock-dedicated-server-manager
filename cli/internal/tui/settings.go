package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
)

type settingsSavedMsg struct{ err error }

type SettingsModel struct {
	serverID string
	fields   []textinput.Model
	labels   []string
	cursor   int
	editing  bool
	saving   bool
	message  string
	width    int
	height   int
}

const (
	settingsFieldName = iota
	settingsFieldMap
	settingsFieldPassword
	settingsFieldCount
)

func NewSettingsModel(serverID string) SettingsModel {
	server, _ := ddsm.GetServer(serverID)
	if server == nil {
		return SettingsModel{serverID: serverID}
	}

	fields := make([]textinput.Model, settingsFieldCount)
	labels := []string{"Name", "Map", "Password"}

	for i := range fields {
		fields[i] = textinput.New()
		fields[i].CharLimit = 64
		fields[i].Width = 40
	}

	fields[settingsFieldName].SetValue(server.Name)
	fields[settingsFieldMap].SetValue(server.Map)
	fields[settingsFieldPassword].SetValue(server.Password)

	// All fields start blurred — user must press Enter to edit
	for i := range fields {
		fields[i].Blur()
	}

	return SettingsModel{
		serverID: serverID,
		fields:   fields,
		labels:   labels,
		cursor:   0,
		editing:  false,
	}
}

func (m SettingsModel) IsEditing() bool {
	return m.editing
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	if len(m.fields) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case settingsSavedMsg:
		m.saving = false
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.message = "Settings saved. Server is restarting..."
		}
		return m, nil

	case tea.KeyMsg:
		if m.saving {
			return m, nil
		}

		if m.editing {
			// In edit mode: typing goes to the field
			switch msg.String() {
			case "enter":
				// Confirm and exit edit mode
				m.fields[m.cursor].Blur()
				m.editing = false
				return m, nil
			case "esc":
				// Cancel edit mode
				m.fields[m.cursor].Blur()
				m.editing = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.fields[m.cursor], cmd = m.fields[m.cursor].Update(msg)
				return m, cmd
			}
		}

		// Browse mode: navigate fields, Enter to edit
		switch msg.String() {
		case "down":
			m.cursor = (m.cursor + 1) % len(m.fields)
			return m, nil
		case "up":
			m.cursor = (m.cursor - 1 + len(m.fields)) % len(m.fields)
			return m, nil
		case "enter":
			m.editing = true
			m.fields[m.cursor].Focus()
			return m, nil
		case "ctrl+s":
			m.saving = true
			m.message = "Saving and restarting server..."
			name := strings.TrimSpace(m.fields[settingsFieldName].Value())
			mapName := strings.TrimSpace(m.fields[settingsFieldMap].Value())
			password := strings.TrimSpace(m.fields[settingsFieldPassword].Value())
			id := m.serverID
			return m, func() tea.Msg {
				err := ddsm.UpdateServerAndRecreate(id, name, mapName, password)
				return settingsSavedMsg{err: err}
			}
		}
	}

	return m, nil
}

func (m SettingsModel) View() string {
	if len(m.fields) == 0 {
		return "\n  No server data available.\n"
	}

	var b strings.Builder
	b.WriteString("\n")

	labelStyle := lipgloss.NewStyle().Width(12).Foreground(Gray)
	activeLabel := lipgloss.NewStyle().Width(12).Foreground(Red).Bold(true)

	for i, field := range m.fields {
		ls := labelStyle
		prefix := "  "
		if i == m.cursor {
			ls = activeLabel
			prefix = "\u25b8 "
		}
		b.WriteString(prefix + ls.Render(m.labels[i]) + " " + field.View())
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n")
		msgStyle := lipgloss.NewStyle().Foreground(Amber)
		if strings.HasPrefix(m.message, "Error:") {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
		}
		b.WriteString("  " + msgStyle.Render(m.message))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.editing {
		b.WriteString(HelpStyle.Render("  [enter] confirm  [esc] cancel  [ctrl+s] save & restart"))
	} else {
		b.WriteString(HelpStyle.Render("  [arrows] navigate  [enter] edit field  [ctrl+s] save & restart  [q] back"))
	}
	b.WriteString("\n")

	return b.String()
}

func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	for i := range m.fields {
		m.fields[i].Width = w - 20
	}
}
