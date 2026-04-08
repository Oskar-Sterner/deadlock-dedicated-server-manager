package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
)

type ServerTab int

const (
	ServerTabConsole ServerTab = iota
	ServerTabConfig
	ServerTabExplorer
)

var serverTabNames = []string{"Console", "Config", "Explorer"}

type exitServerViewMsg struct{}

type ServerViewModel struct {
	activeTab  ServerTab
	serverID   string
	serverName string
	serverPort int
	console    ConsoleModel
	config     ServerConfigModel
	explorer   ExplorerModel
	width      int
	height     int
}

func NewServerViewModel(serverID string) ServerViewModel {
	server, _ := ddsm.GetServer(serverID)
	if server == nil {
		return ServerViewModel{}
	}

	volumePath := ddsm.ServerVolumePath(serverID)

	m := ServerViewModel{
		serverID:   serverID,
		serverName: server.Name,
		serverPort: server.Port,
		console:    NewConsoleModel(),
		config:     NewServerConfigModel(volumePath),
		explorer:   NewExplorerModel(volumePath),
	}
	// Start with console input unfocused so q/tab work for navigation.
	// User presses esc to focus the input for typing RCON commands.
	m.console.inputFocus = false
	m.console.input.Blur()
	return m
}

func (m ServerViewModel) Init() tea.Cmd {
	return m.console.AttachServer(m.serverID)
}

func (m ServerViewModel) Update(msg tea.Msg) (ServerViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			// If console input is focused, let it type 'q'
			if m.activeTab == ServerTabConsole && m.console.inputFocus {
				break
			}
			m.console.Detach()
			return m, func() tea.Msg { return exitServerViewMsg{} }

		case "esc":
			// Console tab: if input focused, unfocus first; otherwise exit
			if m.activeTab == ServerTabConsole && m.console.inputFocus {
				m.console.inputFocus = false
				m.console.input.Blur()
				return m, nil
			}
			m.console.Detach()
			return m, func() tea.Msg { return exitServerViewMsg{} }

		case "tab", "right":
			if m.activeTab == ServerTabConsole && m.console.inputFocus {
				break // let console handle it
			}
			m.activeTab = ServerTab((int(m.activeTab) + 1) % len(serverTabNames))
			return m, nil

		case "shift+tab", "left":
			if m.activeTab == ServerTabConsole && m.console.inputFocus {
				break
			}
			m.activeTab = ServerTab((int(m.activeTab) - 1 + len(serverTabNames)) % len(serverTabNames))
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case ServerTabConsole:
		m.console, cmd = m.console.Update(msg)
	case ServerTabConfig:
		m.config, cmd = m.config.Update(msg)
	case ServerTabExplorer:
		m.explorer, cmd = m.explorer.Update(msg)
	}

	return m, cmd
}

func (m ServerViewModel) View() string {
	var b strings.Builder

	// Server title
	b.WriteString(TitleStyle.Render(fmt.Sprintf("  %s (port %d)", m.serverName, m.serverPort)))
	b.WriteString("\n")

	// Sub-tab bar
	var tabs []string
	for i, name := range serverTabNames {
		if ServerTab(i) == m.activeTab {
			tabs = append(tabs, SubTabActiveStyle.Render(name))
		} else {
			tabs = append(tabs, SubTabInactiveStyle.Render(name))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	b.WriteString("  " + tabBar)
	b.WriteString("\n")

	// Active sub-tab content
	switch m.activeTab {
	case ServerTabConsole:
		b.WriteString(m.console.View())
	case ServerTabConfig:
		b.WriteString(m.config.View())
	case ServerTabExplorer:
		b.WriteString(m.explorer.View())
	}

	return b.String()
}

func (m *ServerViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentHeight := h - 4
	m.console.SetSize(w, contentHeight)
	m.config.SetSize(w, contentHeight)
	m.explorer.SetSize(w, contentHeight)
}
