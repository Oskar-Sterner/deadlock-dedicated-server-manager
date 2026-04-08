package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
)

type createField int

const (
	fieldName createField = iota
	fieldPort
	fieldMap
	fieldPassword
	fieldSteamLogin
	fieldSteamPass
	fieldSteam2FA
	fieldCount
)

type serverCreatedMsg struct {
	server *ddsm.ServerRow
}

type serverCreateErrMsg struct {
	err error
}

type cancelCreateMsg struct{}

type CreateModel struct {
	inputs   []textinput.Model
	labels   []string
	focus    int
	width    int
	height   int
	creating bool
	errMsg   string
}

func NewCreateModel() CreateModel {
	inputs := make([]textinput.Model, fieldCount)
	labels := []string{
		"Server name",
		"Port",
		"Map",
		"Password",
		"Steam login",
		"Steam password",
		"Steam 2FA code",
	}

	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].CharLimit = 64
	}

	inputs[fieldName].Placeholder = "My Deadlock Server"
	inputs[fieldName].Focus()

	nextPort := ddsm.GetNextPort()
	inputs[fieldPort].Placeholder = fmt.Sprintf("%d (default)", nextPort)

	inputs[fieldMap].SetValue("dl_streets")

	inputs[fieldPassword].Placeholder = "optional"

	if ddsm.Cfg.SteamLogin != "" {
		inputs[fieldSteamLogin].SetValue(ddsm.Cfg.SteamLogin)
		inputs[fieldSteamLogin].Placeholder = "from config"
	} else {
		inputs[fieldSteamLogin].Placeholder = "required"
	}

	if ddsm.Cfg.SteamPassword != "" {
		inputs[fieldSteamPass].SetValue(ddsm.Cfg.SteamPassword)
		inputs[fieldSteamPass].Placeholder = "from config"
	} else {
		inputs[fieldSteamPass].Placeholder = "required"
	}
	inputs[fieldSteamPass].EchoMode = textinput.EchoPassword

	inputs[fieldSteam2FA].Placeholder = "optional"

	return CreateModel{
		inputs: inputs,
		labels: labels,
	}
}

func (m CreateModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CreateModel) Update(msg tea.Msg) (CreateModel, tea.Cmd) {
	if m.creating {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.errMsg = ""

		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return cancelCreateMsg{} }

		case "tab", "down":
			m.inputs[m.focus].Blur()
			m.focus = (m.focus + 1) % int(fieldCount)
			m.inputs[m.focus].Focus()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.inputs[m.focus].Blur()
			m.focus = (m.focus - 1 + int(fieldCount)) % int(fieldCount)
			m.inputs[m.focus].Focus()
			return m, textinput.Blink

		case "enter":
			if m.focus < int(fieldCount)-1 {
				m.inputs[m.focus].Blur()
				m.focus++
				m.inputs[m.focus].Focus()
				return m, textinput.Blink
			}
			return m.validateAndSubmit()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m CreateModel) validateAndSubmit() (CreateModel, tea.Cmd) {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" {
		m.errMsg = "Server name is required"
		m.inputs[m.focus].Blur()
		m.focus = int(fieldName)
		m.inputs[m.focus].Focus()
		return m, nil
	}

	steamLogin := strings.TrimSpace(m.inputs[fieldSteamLogin].Value())
	if steamLogin == "" {
		m.errMsg = "Steam login is required"
		m.inputs[m.focus].Blur()
		m.focus = int(fieldSteamLogin)
		m.inputs[m.focus].Focus()
		return m, nil
	}

	steamPass := strings.TrimSpace(m.inputs[fieldSteamPass].Value())
	if steamPass == "" {
		m.errMsg = "Steam password is required"
		m.inputs[m.focus].Blur()
		m.focus = int(fieldSteamPass)
		m.inputs[m.focus].Focus()
		return m, nil
	}

	portStr := strings.TrimSpace(m.inputs[fieldPort].Value())
	port := ddsm.GetNextPort()
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	mapName := strings.TrimSpace(m.inputs[fieldMap].Value())
	if mapName == "" {
		mapName = "dl_streets"
	}

	password := strings.TrimSpace(m.inputs[fieldPassword].Value())
	steam2FA := strings.TrimSpace(m.inputs[fieldSteam2FA].Value())

	m.creating = true
	return m, func() tea.Msg {
		server, err := ddsm.CreateServer(ddsm.ServerCreateOpts{
			Name:       name,
			Port:       port,
			Map:        mapName,
			Password:   password,
			SteamLogin: steamLogin,
			SteamPass:  steamPass,
			Steam2FA:   steam2FA,
		})
		if err != nil {
			return serverCreateErrMsg{err: err}
		}
		return serverCreatedMsg{server: server}
	}
}

func (m CreateModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("  Create Server"))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(Gray).Width(18)
	focusedLabelStyle := lipgloss.NewStyle().Foreground(White).Bold(true).Width(18)

	for i, input := range m.inputs {
		ls := labelStyle
		cursor := "  "
		if i == m.focus {
			ls = focusedLabelStyle
			cursor = "\u25b8 "
		}

		label := ls.Render(m.labels[i] + ":")
		b.WriteString(cursor + label + " " + input.View())

		if i == int(fieldMap) {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(Gray).Render("(dl_streets, dl_midtown, dl_hideout)"))
		}

		b.WriteString("\n")
	}

	if m.creating {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(Yellow).Bold(true).Render("Creating server...") + "\n")
	}

	if m.errMsg != "" {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true).Render(m.errMsg) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  tab/\u2191\u2193 navigate  enter next/submit  esc cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m *CreateModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := w / 3
	if inputWidth < 30 {
		inputWidth = 30
	}
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}
}
