package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const banner = ` ____    ____    ____    __  __
|  _ \  |  _ \  / ___|  |  \/  |
| | | | | | | | \___ \  | |\/| |
| |_| | | |_| |  ___) | | |  | |
|____/  |____/  |____/  |_|  |_|`

type Tab int

const (
	TabServers Tab = iota
	TabConsole
	TabConfig
	TabTools
)

var tabNames = []string{"Servers", "Console", "Config", "Tools"}

type Model struct {
	activeTab Tab
	servers   ServersModel
	console   ConsoleModel
	config    ConfigModel
	tools     ToolsModel
	width     int
	height    int
	quitting  bool
}

func NewModel() Model {
	return Model{
		servers: NewServersModel(),
		console: NewConsoleModel(),
		config:  NewConfigModel(),
		tools:   NewToolsModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.servers.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentHeight := msg.Height - 12 // banner + tab bar + status bar
		m.servers.SetSize(msg.Width, contentHeight)
		m.console.SetSize(msg.Width, contentHeight)
		m.config.SetSize(msg.Width, contentHeight)
		m.tools.SetSize(msg.Width, contentHeight)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.console.Detach()
			return m, tea.Quit
		case "q":
			if m.activeTab == TabConsole && m.console.inputFocus {
				break
			}
			m.quitting = true
			m.console.Detach()
			return m, tea.Quit
		case "tab":
			m.activeTab = Tab((int(m.activeTab) + 1) % len(tabNames))
			return m, nil
		case "shift+tab":
			m.activeTab = Tab((int(m.activeTab) - 1 + len(tabNames)) % len(tabNames))
			return m, nil
		}

	case switchToConsoleMsg:
		m.activeTab = TabConsole
		cmd := m.console.AttachServer(msg.serverID)
		m.console.input.Focus()
		m.console.inputFocus = true
		return m, cmd

	case configEditDoneMsg:
		return m, nil
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case TabServers:
		m.servers, cmd = m.servers.Update(msg)
	case TabConsole:
		m.console, cmd = m.console.Update(msg)
	case TabConfig:
		m.config, cmd = m.config.Update(msg)
	case TabTools:
		m.tools, cmd = m.tools.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Banner top-left
	b.WriteString(BannerStyle.Render(banner))
	b.WriteString("\n")

	// Tab bar below banner
	var tabs []string
	for i, name := range tabNames {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, TabActiveStyle.Render(name))
		} else {
			tabs = append(tabs, TabInactiveStyle.Render(name))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	b.WriteString(TabBarStyle.Render(tabBar))
	b.WriteString("\n")

	// Active tab content
	switch m.activeTab {
	case TabServers:
		b.WriteString(m.servers.View())
	case TabConsole:
		b.WriteString(m.console.View())
	case TabConfig:
		b.WriteString(m.config.View())
	case TabTools:
		b.WriteString(m.tools.View())
	}

	// Status bar
	statusBar := StatusBarStyle.Width(m.width).Render("DDSM v0.1.0  |  Tab/Shift+Tab: switch tabs  |  q: quit")
	b.WriteString("\n")
	b.WriteString(statusBar)

	return b.String()
}
