package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
)

type serversRefreshMsg []*ddsm.ServerStatus
type tickMsg struct{}
type serverActionDoneMsg struct {
	message string
}
type switchToConsoleMsg struct {
	serverID string
}

type ServersModel struct {
	servers []*ddsm.ServerStatus
	cursor  int
	width   int
	height  int
	message string
}

func NewServersModel() ServersModel {
	return ServersModel{}
}

func (m ServersModel) Init() tea.Cmd {
	return refreshServers()
}

func refreshServers() tea.Cmd {
	return func() tea.Msg {
		statuses, _ := ddsm.ListServerStatuses()
		return serversRefreshMsg(statuses)
	}
}

func tickRefresh() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m ServersModel) Update(msg tea.Msg) (ServersModel, tea.Cmd) {
	switch msg := msg.(type) {
	case serversRefreshMsg:
		m.servers = msg
		if m.cursor >= len(m.servers) && len(m.servers) > 0 {
			m.cursor = len(m.servers) - 1
		}
		return m, tickRefresh()

	case tickMsg:
		return m, refreshServers()

	case serverActionDoneMsg:
		m.message = msg.message
		return m, refreshServers()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursor < len(m.servers)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			if m.cursor < len(m.servers) {
				return m, serverAction(m.servers[m.cursor].ID, "start")
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("x"))):
			if m.cursor < len(m.servers) {
				return m, serverAction(m.servers[m.cursor].ID, "stop")
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			if m.cursor < len(m.servers) {
				return m, serverAction(m.servers[m.cursor].ID, "restart")
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("w"))):
			if m.cursor < len(m.servers) {
				return m, serverAction(m.servers[m.cursor].ID, "wake")
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.cursor < len(m.servers) {
				return m, func() tea.Msg { return switchToConsoleMsg{serverID: m.servers[m.cursor].ID} }
			}
		}
	}

	return m, nil
}

func serverAction(id, action string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "start":
			err = ddsm.StartServer(id)
		case "stop":
			err = ddsm.StopServer(id)
		case "restart":
			err = ddsm.RestartServer(id)
		case "wake":
			err = ddsm.ManualWake(id)
		}
		if err != nil {
			return serverActionDoneMsg{message: fmt.Sprintf("Error: %v", err)}
		}
		return serverActionDoneMsg{message: fmt.Sprintf("Server %s: %s", id[:8], action)}
	}
}

func (m ServersModel) View() string {
	if len(m.servers) == 0 {
		return "\n  No servers configured. Run 'ddsm create' to add one.\n"
	}

	var b strings.Builder

	header := fmt.Sprintf("  %-10s %-24s %-12s %-8s %-12s %-10s %-12s", "ID", "NAME", "STATUS", "PORT", "PLAYERS", "CPU", "MEMORY")
	b.WriteString(lipgloss.NewStyle().Foreground(Gray).Width(m.width).Render(header))
	b.WriteString("\n")
	sepWidth := m.width - 4
	if sepWidth < 20 {
		sepWidth = 90
	}
	b.WriteString(lipgloss.NewStyle().Foreground(Gray).Render("  " + strings.Repeat("\u2500", sepWidth)))
	b.WriteString("\n")

	for i, s := range m.servers {
		id := s.ID
		if len(id) > 10 {
			id = id[:10]
		}
		name := s.Name
		if len(name) > 24 {
			name = name[:21] + "..."
		}

		players := "\u2014"
		if s.Status == "running" {
			players = fmt.Sprintf("%d/%d", s.Players, s.MaxPlayers)
		}

		cpu := "\u2014"
		mem := "\u2014"
		if s.Stats != nil {
			cpu = fmt.Sprintf("%.1f%%", s.Stats.CPUPercent)
			mem = fmt.Sprintf("%.0fMB", s.Stats.MemoryMB)
		}

		// Build line with plain text (no ANSI) so padding is correct
		line := fmt.Sprintf("  %-10s %-24s %-12s %-8d %-12s %-10s %-12s", id, name, s.Status, s.Port, players, cpu, mem)

		// Pad to full terminal width with spaces
		for len(line) < m.width {
			line += " "
		}

		// Apply cursor background to the full-width plain line
		if i == m.cursor {
			line = lipgloss.NewStyle().Background(lipgloss.Color("#2a2a2a")).Render(line)
		}

		// Insert status color last — replacing plain status text with styled version
		line = strings.Replace(line, s.Status, StatusStyle(s.Status).Render(s.Status), 1)

		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n  " + m.message + "\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  s start  x stop  r restart  w wake  enter console"))
	b.WriteString("\n")

	return b.String()
}

func (m *ServersModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ServersModel) SelectedServer() *ddsm.ServerStatus {
	if m.cursor < len(m.servers) {
		return m.servers[m.cursor]
	}
	return nil
}
