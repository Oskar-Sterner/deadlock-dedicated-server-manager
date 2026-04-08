package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
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
	fieldDeadworks
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

type createProgressMsg string

type CreateModel struct {
	inputs       []textinput.Model
	labels       []string
	focus        int
	width        int
	height       int
	creating     bool
	errMsg       string
	deadworks    bool
	progressMsg  string
	progressChan <-chan string
	spinner      spinner.Model
}

func NewCreateModel() CreateModel {
	ddsm.LoadConfig()

	inputs := make([]textinput.Model, fieldCount)
	labels := []string{
		"Server name",
		"Port",
		"Map",
		"Password",
		"Deadworks",
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

	inputs[fieldMap].SetValue("dl_hideout")

	inputs[fieldPassword].Placeholder = "optional"

	inputs[fieldDeadworks].Placeholder = "press [enter] to toggle"

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

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(Yellow)

	return CreateModel{
		inputs:  inputs,
		labels:  labels,
		spinner: s,
	}
}

func (m CreateModel) Init() tea.Cmd {
	return textinput.Blink
}

func waitForProgress(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return createProgressMsg(msg)
	}
}

func (m CreateModel) Update(msg tea.Msg) (CreateModel, tea.Cmd) {
	switch msg := msg.(type) {
	case createProgressMsg:
		m.progressMsg = string(msg)
		return m, waitForProgress(m.progressChan)

	case spinner.TickMsg:
		if m.creating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

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
			if m.focus == int(fieldDeadworks) {
				m.deadworks = !m.deadworks
				return m, nil
			}

			if m.focus < int(fieldCount)-1 {
				m.inputs[m.focus].Blur()
				m.focus++
				m.inputs[m.focus].Focus()
				return m, textinput.Blink
			}
			return m.validateAndSubmit()

		case " ":
			if m.focus == int(fieldDeadworks) {
				m.deadworks = !m.deadworks
				return m, nil
			}
		}
	}

	if m.focus == int(fieldDeadworks) {
		return m, nil
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
		mapName = "dl_hideout"
	}

	password := strings.TrimSpace(m.inputs[fieldPassword].Value())
	steam2FA := strings.TrimSpace(m.inputs[fieldSteam2FA].Value())
	deadworks := m.deadworks

	m.creating = true
	m.progressMsg = "Initializing..."

	progressCh := make(chan string, 8)
	m.progressChan = progressCh

	createCmd := func() tea.Msg {
		server, err := ddsm.CreateServerWithProgress(ddsm.ServerCreateOpts{
			Name:       name,
			Port:       port,
			Map:        mapName,
			Password:   password,
			SteamLogin: steamLogin,
			SteamPass:  steamPass,
			Steam2FA:   steam2FA,
			Deadworks:  deadworks,
		}, progressCh)
		if err != nil {
			return serverCreateErrMsg{err: err}
		}
		return serverCreatedMsg{server: server}
	}

	return m, tea.Batch(createCmd, waitForProgress(progressCh), m.spinner.Tick)
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

		if i == int(fieldDeadworks) {
			checkbox := "[ ]"
			desc := lipgloss.NewStyle().Foreground(Gray).Render("  Install Deadworks mod framework")
			if m.deadworks {
				checkbox = lipgloss.NewStyle().Foreground(Green).Bold(true).Render("[x]")
				desc = lipgloss.NewStyle().Foreground(Green).Render("  Deadworks will be installed")
			}
			b.WriteString(cursor + label + " " + checkbox + desc)
			b.WriteString("\n")
			continue
		}

		b.WriteString(cursor + label + " " + input.View())

		if i == int(fieldMap) {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(Gray).Render("(dl_streets, dl_midtown, dl_hideout)"))
		}

		b.WriteString("\n")
	}

	if m.creating {
		b.WriteString("\n  " + m.spinner.View() + " " + lipgloss.NewStyle().Foreground(Yellow).Bold(true).Render(m.progressMsg) + "\n")
	}

	if m.errMsg != "" {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true).Render(m.errMsg) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  [tab/\u2191\u2193] navigate  [enter] next/submit  [space] toggle  [esc] cancel"))
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
