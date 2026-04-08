package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
)

type configEditDoneMsg struct{}

type ConfigModel struct {
	width  int
	height int
}

func NewConfigModel() ConfigModel {
	return ConfigModel{}
}

func (m ConfigModel) Init() tea.Cmd {
	return nil
}

func (m ConfigModel) Update(msg tea.Msg) (ConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			return m, openEditor()
		}
	}
	return m, nil
}

func openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}
	path := ddsm.ConfigPath()
	// Write current config to file if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		ddsm.EnsureConfigDir()
		ddsm.WriteConfigFile(path)
	}
	c := exec.Command(editor, path)
	// Fix for terminals like kitty that may not have terminfo on the server
	c.Env = append(os.Environ(), "TERM=xterm")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return configEditDoneMsg{}
	})
}

func (m ConfigModel) View() string {
	// Reload config to pick up changes made while the TUI is running
	ddsm.LoadConfig()

	var b strings.Builder

	b.WriteString(TitleStyle.Render("  Configuration"))
	b.WriteString("\n\n")

	cfg := ddsm.Cfg
	labelStyle := lipgloss.NewStyle().Foreground(Gray).Width(22)
	valStyle := lipgloss.NewStyle().Foreground(White)

	row := func(label, value string) {
		b.WriteString("  " + labelStyle.Render(label) + valStyle.Render(value) + "\n")
	}

	row("Config file:", ddsm.ConfigPath())
	row("Server IP:", cfg.ServerIP)
	row("RCON password:", strings.Repeat("*", len(cfg.RconPassword)))
	row("Servers dir:", cfg.ServersDir)
	row("Docker image:", cfg.DockerImage)
	row("Database:", cfg.DbPath)
	if cfg.SteamLogin != "" {
		row("Steam login:", cfg.SteamLogin)
		row("Steam password:", strings.Repeat("*", len(cfg.SteamPassword)))
	}
	b.WriteString("\n")
	row("Auto-sleep:", fmt.Sprintf("%v", cfg.AutoSleep.Enabled))
	row("Idle timeout:", fmt.Sprintf("%ds", cfg.AutoSleep.IdleTimeout))
	row("Poll interval:", fmt.Sprintf("%ds", cfg.AutoSleep.PollInterval))

	b.WriteString("\n")

	servers, _ := ddsm.ListServers()
	if len(servers) > 0 {
		b.WriteString(TitleStyle.Render("  Servers"))
		b.WriteString("\n\n")
		for _, s := range servers {
			id := s.ID
			if len(id) > 8 {
				id = id[:8]
			}
			b.WriteString(fmt.Sprintf("  %-8s %-20s port:%-6d map:%s\n", id, s.Name, s.Port, s.Map))
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  [e] edit config file"))
	b.WriteString("\n")

	return b.String()
}

func (m *ConfigModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
