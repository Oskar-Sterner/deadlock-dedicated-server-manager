package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ServerConfigModel struct {
	volumePath string
	cfgFiles   []string
	cursor     int
	width      int
	height     int
}

func NewServerConfigModel(volumePath string) ServerConfigModel {
	m := ServerConfigModel{
		volumePath: volumePath,
	}
	m.scanCfgFiles()
	return m
}

func (m *ServerConfigModel) scanCfgFiles() {
	m.cfgFiles = nil

	cfgDir := filepath.Join(m.volumePath, "Deadlock", "game", "citadel", "cfg")
	entries, err := os.ReadDir(cfgDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			m.cfgFiles = append(m.cfgFiles, filepath.Join(cfgDir, e.Name()))
		}
	}
}

func (m ServerConfigModel) Update(msg tea.Msg) (ServerConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case configEditDoneMsg:
		m.scanCfgFiles()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			if m.cursor < len(m.cfgFiles)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.cursor < len(m.cfgFiles) {
				return m, editFileCmd(m.cfgFiles[m.cursor])
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.scanCfgFiles()
		}
	}
	return m, nil
}

func editFileCmd(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}
	c := exec.Command(editor, path)
	c.Env = append(os.Environ(), "TERM=xterm")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return configEditDoneMsg{}
	})
}

func (m ServerConfigModel) View() string {
	var b strings.Builder

	if len(m.cfgFiles) == 0 {
		b.WriteString("\n  No config files found.\n")
		b.WriteString(HelpStyle.Render("\n  Config files appear after the server runs for the first time."))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("  [r] refresh"))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString("\n")
	for i, f := range m.cfgFiles {
		name := filepath.Base(f)
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(White)
		if i == m.cursor {
			cursor = "\u25b8 "
			style = style.Foreground(Red).Bold(true)
		}
		b.WriteString(style.Render(cursor + name))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  [enter] edit  [r] refresh"))
	b.WriteString("\n")

	return b.String()
}

func (m *ServerConfigModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
