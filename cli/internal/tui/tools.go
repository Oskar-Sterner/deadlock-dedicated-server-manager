package tui

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
)

type doctorResultMsg []ddsm.CheckResult
type publicIPMsg string

type ToolsModel struct {
	menuItems []string
	cursor    int
	output    viewport.Model
	outputStr string
	width     int
	height    int
}

func NewToolsModel() ToolsModel {
	vp := viewport.New(80, 15)
	return ToolsModel{
		menuItems: []string{"Doctor \u2014 Run health checks", "Show public IP", "About"},
		output:    vp,
	}
}

func (m ToolsModel) Init() tea.Cmd {
	return nil
}

func (m ToolsModel) Update(msg tea.Msg) (ToolsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case doctorResultMsg:
		var b strings.Builder
		for _, r := range msg {
			var icon string
			var style lipgloss.Style
			switch r.Status {
			case "pass":
				icon = "\u2713"
				style = lipgloss.NewStyle().Foreground(Green)
			case "fail":
				icon = "\u2717"
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
			case "warn":
				icon = "!"
				style = lipgloss.NewStyle().Foreground(Yellow)
			}
			b.WriteString(style.Render(fmt.Sprintf("  [%s] %s \u2014 %s", icon, r.Name, r.Detail)))
			b.WriteString("\n")
		}
		fails := 0
		for _, r := range msg {
			if r.Status == "fail" {
				fails++
			}
		}
		if fails > 0 {
			b.WriteString(fmt.Sprintf("\n  %d check(s) failed.\n", fails))
		} else {
			b.WriteString("\n  All checks passed.\n")
		}
		m.outputStr = b.String()
		m.output.SetContent(m.outputStr)

	case publicIPMsg:
		m.outputStr = fmt.Sprintf("  Public IP: %s\n", string(msg))
		m.output.SetContent(m.outputStr)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			switch m.cursor {
			case 0:
				m.outputStr = "  Running health checks...\n"
				m.output.SetContent(m.outputStr)
				return m, runDoctor()
			case 1:
				m.outputStr = "  Resolving public IP...\n"
				m.output.SetContent(m.outputStr)
				return m, resolvePublicIP()
			case 2:
				m.outputStr = "  DDSM \u2014 Deadlock Dedicated Server Manager v0.1.0\n  https://github.com/Oskar-Sterner/deadlock-dedicated-server-manager\n"
				m.output.SetContent(m.outputStr)
			}
		}
	}

	var cmd tea.Cmd
	m.output, cmd = m.output.Update(msg)
	return m, cmd
}

func runDoctor() tea.Cmd {
	return func() tea.Msg {
		results := ddsm.RunDoctor()
		return doctorResultMsg(results)
	}
}

func resolvePublicIP() tea.Cmd {
	return func() tea.Msg {
		conn, err := net.DialTimeout("udp", "8.8.8.8:80", 3*time.Second)
		if err != nil {
			return publicIPMsg(fmt.Sprintf("Error: %v", err))
		}
		defer conn.Close()
		addr := conn.LocalAddr().(*net.UDPAddr)
		return publicIPMsg(addr.IP.String())
	}
}

func (m ToolsModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("  Tools"))
	b.WriteString("\n\n")

	for i, item := range m.menuItems {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(White)
		if i == m.cursor {
			cursor = "\u25b8 "
			style = style.Foreground(Red).Bold(true)
		}
		b.WriteString(style.Render(cursor + item))
		b.WriteString("\n")
	}

	if m.outputStr != "" {
		b.WriteString("\n")
		b.WriteString(m.output.View())
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  enter select  j/k navigate"))
	b.WriteString("\n")

	return b.String()
}

func (m *ToolsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.output.Width = w - 4
	m.output.Height = h - 12
}
