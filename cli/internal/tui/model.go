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
	TabConfig
	TabTools
)

var tabNames = []string{"Servers", "Config", "Tools"}

type Model struct {
	activeTab Tab
	servers   ServersModel
	config    ConfigModel
	tools     ToolsModel
	width     int
	height    int
	quitting  bool
}

func NewModel() Model {
	return Model{
		servers: NewServersModel(),
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
		m.config.SetSize(msg.Width, contentHeight)
		m.tools.SetSize(msg.Width, contentHeight)
		return m, nil

	case tea.KeyMsg:
		inputActive := m.activeTab == TabServers && (m.servers.creating || m.servers.viewing)

		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if inputActive {
				break
			}
			m.quitting = true
			return m, tea.Quit
		case "tab":
			if inputActive {
				break
			}
			m.activeTab = Tab((int(m.activeTab) + 1) % len(tabNames))
			return m, nil
		case "shift+tab":
			if inputActive {
				break
			}
			m.activeTab = Tab((int(m.activeTab) - 1 + len(tabNames)) % len(tabNames))
			return m, nil
		case "right":
			if inputActive {
				break
			}
			m.activeTab = Tab((int(m.activeTab) + 1) % len(tabNames))
			return m, nil
		case "left":
			if inputActive {
				break
			}
			m.activeTab = Tab((int(m.activeTab) - 1 + len(tabNames)) % len(tabNames))
			return m, nil
		}

	case configEditDoneMsg:
		// Fall through to let active tab handle it
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case TabServers:
		m.servers, cmd = m.servers.Update(msg)
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
	b.WriteString("\n\n")

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
	case TabConfig:
		b.WriteString(m.config.View())
	case TabTools:
		b.WriteString(m.tools.View())
	}

	// Status bar
	statusBar := StatusBarStyle.Width(m.width).Render("DDSM v0.2.0  |  [arrows/tab] switch tabs  |  [q] quit")
	b.WriteString("\n")
	b.WriteString(statusBar)

	return b.String()
}
