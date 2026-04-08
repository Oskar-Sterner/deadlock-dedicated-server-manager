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
	ServerTabSettings
	ServerTabConfig
	ServerTabExplorer
)

var serverTabNames = []string{"Console", "Settings", "Config", "Explorer"}

type exitServerViewMsg struct{}

type ServerViewModel struct {
	activeTab  ServerTab
	serverID   string
	serverName string
	serverPort int
	console    ConsoleModel
	settings   SettingsModel
	config     ServerConfigModel
	explorer   ExplorerModel
	width      int
	height     int
}

func NewServerViewModel(serverID string) (ServerViewModel, tea.Cmd) {
	server, _ := ddsm.GetServer(serverID)
	if server == nil {
		return ServerViewModel{}, nil
	}

	volumePath := ddsm.ServerVolumePath(serverID)

	console := NewConsoleModel()
	console.inputFocus = false
	console.input.Blur()
	cmd := console.AttachServer(serverID)

	m := ServerViewModel{
		serverID:   serverID,
		serverName: server.Name,
		serverPort: server.Port,
		console:    console,
		settings:   NewSettingsModel(serverID),
		config:     NewServerConfigModel(volumePath),
		explorer:   NewExplorerModel(volumePath),
	}
	return m, cmd
}

func (m ServerViewModel) Init() tea.Cmd {
	return nil
}

func (m ServerViewModel) Update(msg tea.Msg) (ServerViewModel, tea.Cmd) {
	// Route settings saved messages regardless of active tab
	if saved, ok := msg.(settingsSavedMsg); ok {
		if saved.err == nil {
			if server, _ := ddsm.GetServer(m.serverID); server != nil {
				m.serverName = server.Name
				m.serverPort = server.Port
			}
			m.console.Detach()
			cmd := m.console.AttachServer(m.serverID)
			m.settings, _ = m.settings.Update(msg)
			return m, cmd
		}
		m.settings, _ = m.settings.Update(msg)
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Only block navigation when actively editing a field
		inputActive := (m.activeTab == ServerTabConsole && m.console.inputFocus) ||
			(m.activeTab == ServerTabSettings && m.settings.IsEditing())

		switch msg.String() {
		case "q":
			if inputActive {
				break
			}
			m.console.Detach()
			return m, func() tea.Msg { return exitServerViewMsg{} }

		case "esc":
			if m.activeTab == ServerTabConsole && m.console.inputFocus {
				m.console.inputFocus = false
				m.console.input.Blur()
				return m, nil
			}
			if m.activeTab == ServerTabSettings && m.settings.IsEditing() {
				break // let settings handle esc
			}
			m.console.Detach()
			return m, func() tea.Msg { return exitServerViewMsg{} }

		case "right":
			if inputActive {
				break
			}
			m.activeTab = ServerTab((int(m.activeTab) + 1) % len(serverTabNames))
			return m, nil

		case "left":
			if inputActive {
				break
			}
			m.activeTab = ServerTab((int(m.activeTab) - 1 + len(serverTabNames)) % len(serverTabNames))
			return m, nil

		case "tab":
			if inputActive {
				break
			}
			m.activeTab = ServerTab((int(m.activeTab) + 1) % len(serverTabNames))
			return m, nil

		case "shift+tab":
			if inputActive {
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
	case ServerTabSettings:
		m.settings, cmd = m.settings.Update(msg)
	case ServerTabConfig:
		m.config, cmd = m.config.Update(msg)
	case ServerTabExplorer:
		m.explorer, cmd = m.explorer.Update(msg)
	}

	return m, cmd
}

func (m ServerViewModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(fmt.Sprintf("  %s (port %d)", m.serverName, m.serverPort)))
	b.WriteString("\n")

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

	switch m.activeTab {
	case ServerTabConsole:
		b.WriteString(m.console.View())
	case ServerTabSettings:
		b.WriteString(m.settings.View())
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
	m.settings.SetSize(w, contentHeight)
	m.config.SetSize(w, contentHeight)
	m.explorer.SetSize(w, contentHeight)
}
