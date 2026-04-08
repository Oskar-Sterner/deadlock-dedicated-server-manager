package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
)

type logLineMsg string
type rconResponseMsg string

type ConsoleModel struct {
	serverID   string
	serverName string
	serverPort int
	viewport   viewport.Model
	input      textinput.Model
	logs       []string
	autoScroll bool
	logDone    chan struct{}
	logChan    <-chan string
	width      int
	height     int
	inputFocus bool
}

func NewConsoleModel() ConsoleModel {
	ti := textinput.New()
	ti.Placeholder = "RCON command..."
	ti.CharLimit = 256

	vp := viewport.New(80, 20)

	return ConsoleModel{
		viewport:   vp,
		input:      ti,
		autoScroll: true,
		inputFocus: true,
	}
}

func (m ConsoleModel) Init() tea.Cmd {
	return nil
}

func (m *ConsoleModel) AttachServer(serverID string) tea.Cmd {
	if m.logDone != nil {
		close(m.logDone)
	}

	server, err := ddsm.GetServer(serverID)
	if err != nil || server == nil {
		return nil
	}

	m.serverID = serverID
	m.serverName = server.Name
	m.serverPort = server.Port
	m.logs = nil
	m.logDone = make(chan struct{})
	m.logChan = nil

	if !server.ContainerID.Valid {
		m.logs = append(m.logs, "Server has no container")
		return nil
	}

	ch, err := ddsm.StreamLogs(server.ContainerID.String, 200, m.logDone)
	if err != nil {
		m.logs = append(m.logs, fmt.Sprintf("Failed to stream logs: %v", err))
		return nil
	}

	m.logChan = ch
	return waitForLogLine(ch)
}

func waitForLogLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logLineMsg("[stream ended]")
		}
		return logLineMsg(line)
	}
}

func (m *ConsoleModel) Detach() {
	if m.logDone != nil {
		close(m.logDone)
		m.logDone = nil
		m.logChan = nil
	}
}

func (m ConsoleModel) Update(msg tea.Msg) (ConsoleModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case logLineMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 500 {
			m.logs = m.logs[len(m.logs)-500:]
		}
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		if m.logChan != nil {
			return m, waitForLogLine(m.logChan)
		}
		return m, nil

	case rconResponseMsg:
		resp := string(msg)
		if strings.HasPrefix(resp, "Error:") {
			m.logs = append(m.logs, lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(resp))
		} else {
			m.logs = append(m.logs, resp)
		}
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.inputFocus = !m.inputFocus
			if m.inputFocus {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil
		case "a":
			if !m.inputFocus {
				m.autoScroll = !m.autoScroll
				return m, nil
			}
		case "enter":
			if m.inputFocus {
				cmd := strings.TrimSpace(m.input.Value())
				if cmd != "" {
					m.input.Reset()
					m.logs = append(m.logs, lipgloss.NewStyle().Foreground(Amber).Render("> "+cmd))
					m.viewport.SetContent(strings.Join(m.logs, "\n"))
					if m.autoScroll {
						m.viewport.GotoBottom()
					}
					return m, sendRconCmd(m.serverPort, cmd)
				}
			}
		}

		// Don't forward left/right to children — model uses those for tab switching
		if msg.String() != "left" && msg.String() != "right" {
			if m.inputFocus {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func sendRconCmd(port int, command string) tea.Cmd {
	return func() tea.Msg {
		resp, err := ddsm.SendRconCommand(port, command, ddsm.Cfg.RconPassword)
		if err != nil {
			return rconResponseMsg(fmt.Sprintf("Error: %v", err))
		}
		return rconResponseMsg(resp)
	}
}

func (m ConsoleModel) View() string {
	if m.serverID == "" {
		return "\n  No server selected. Press Enter on a server in the Servers tab.\n"
	}

	var b strings.Builder

	b.WriteString("\n")

	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	inputStyle := lipgloss.NewStyle().BorderTop(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(Gray)
	b.WriteString(inputStyle.Render(m.input.View()))
	b.WriteString("\n")

	scrollIndicator := "ON"
	if !m.autoScroll {
		scrollIndicator = "OFF"
	}
	b.WriteString(HelpStyle.Render(fmt.Sprintf("  [esc] toggle focus  [a] auto-scroll (%s)  [enter] send command", scrollIndicator)))

	return b.String()
}

func (m *ConsoleModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w - 4
	m.viewport.Height = h - 6
	m.input.Width = w - 4
}
