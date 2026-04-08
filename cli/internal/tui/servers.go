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

// NotificationMsg is sent by background systems (autosleep, etc.) via p.Send()
// to display messages inside the TUI instead of writing to stdout.
type NotificationMsg struct {
	Text string
}

type serversRefreshMsg []*ddsm.ServerStatus
type tickMsg struct{}
type serverActionDoneMsg struct {
	message string
}
type ServersModel struct {
	servers    []*ddsm.ServerStatus
	cursor     int
	width      int
	height     int
	message    string
	creating   bool
	create     CreateModel
	viewing    bool
	serverView ServerViewModel
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
	// Handle create result messages regardless of mode
	switch msg.(type) {
	case cancelCreateMsg:
		m.creating = false
		return m, refreshServers()
	case serverCreatedMsg:
		created := msg.(serverCreatedMsg)
		m.creating = false
		m.message = fmt.Sprintf("Server '%s' created (port %d)", created.server.Name, created.server.Port)
		return m, refreshServers()
	case serverCreateErrMsg:
		errMsg := msg.(serverCreateErrMsg)
		m.create.errMsg = fmt.Sprintf("Error: %v", errMsg.err)
		m.create.creating = false
		return m, nil
	case exitServerViewMsg:
		m.viewing = false
		return m, refreshServers()
	case NotificationMsg:
		m.message = msg.(NotificationMsg).Text
		return m, nil
	}

	// Delegate to create model when in create mode
	if m.creating {
		var cmd tea.Cmd
		m.create, cmd = m.create.Update(msg)
		return m, cmd
	}

	// Delegate to server view when viewing a server
	if m.viewing {
		var cmd tea.Cmd
		m.serverView, cmd = m.serverView.Update(msg)
		return m, cmd
	}

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
		case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
			m.creating = true
			m.create = NewCreateModel()
			m.create.SetSize(m.width, m.height)
			return m, m.create.Init()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			if m.cursor < len(m.servers)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
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
				m.viewing = true
				var cmd tea.Cmd
				m.serverView, cmd = NewServerViewModel(m.servers[m.cursor].ID)
				m.serverView.SetSize(m.width, m.height)
				return m, cmd
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
	if m.creating {
		return m.create.View()
	}

	if m.viewing {
		return m.serverView.View()
	}

	if len(m.servers) == 0 {
		return "\n  No servers configured. Press 'c' to create one.\n"
	}

	var b strings.Builder

	header := fmt.Sprintf("  %-36s %-24s %-12s %-8s %-12s %-10s %-12s", "ID", "NAME", "STATUS", "PORT", "PLAYERS", "CPU", "MEMORY")
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
		if len(id) > 36 {
			id = id[:36]
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

		// Color status — reset only foreground (39) and bold (22), preserve background
		statusColored := statusAnsi(s.Status) + fmt.Sprintf("%-12s", s.Status) + "\033[39;22m"

		// Build line — statusColored has ANSI but fixed visible width of 12
		line := fmt.Sprintf("  %-36s %-24s %s%-8d %-12s %-10s %-12s", id, name, statusColored, s.Port, players, cpu, mem)

		// Pad to full width based on visible length (subtract ANSI overhead)
		visibleLen := 2 + 36 + 24 + 12 + 8 + 12 + 10 + 12
		pad := m.width - visibleLen
		if pad > 0 {
			line += strings.Repeat(" ", pad)
		}

		if i == m.cursor {
			// Use raw ANSI background to wrap entire line without reset issues
			line = "\033[48;2;42;42;42m" + line + "\033[0m"
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(Amber).Render(m.message) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  [c] create  [s] start  [x] stop  [r] restart  [w] wake  [enter] view server"))
	b.WriteString("\n")

	return b.String()
}

func statusAnsi(status string) string {
	switch status {
	case "running":
		return "\033[1;38;2;74;222;128m" // green bold
	case "exited", "stopped", "dead":
		return "\033[1;38;2;239;68;68m" // red bold
	case "sleeping":
		return "\033[1;38;2;250;204;21m" // yellow bold
	case "waking":
		return "\033[1;38;2;96;165;250m" // blue bold
	default:
		return "\033[38;2;163;163;163m" // gray
	}
}

func (m *ServersModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	if m.viewing {
		m.serverView.SetSize(w, h)
	}
}

func (m ServersModel) SelectedServer() *ddsm.ServerStatus {
	if m.cursor < len(m.servers) {
		return m.servers[m.cursor]
	}
	return nil
}
