package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fileEntry struct {
	name  string
	path  string
	isDir bool
	size  int64
}

type ExplorerModel struct {
	rootPath    string
	currentPath string
	entries     []fileEntry
	cursor      int
	width       int
	height      int
	message     string
}

func NewExplorerModel(rootPath string) ExplorerModel {
	m := ExplorerModel{
		rootPath:    rootPath,
		currentPath: rootPath,
	}
	m.loadDir()
	return m
}

func (m *ExplorerModel) loadDir() {
	m.entries = nil
	m.cursor = 0
	m.message = ""

	entries, err := os.ReadDir(m.currentPath)
	if err != nil {
		m.message = fmt.Sprintf("Error: %v", err)
		return
	}

	if m.currentPath != m.rootPath {
		m.entries = append(m.entries, fileEntry{
			name:  "..",
			path:  filepath.Dir(m.currentPath),
			isDir: true,
		})
	}

	var dirs, files []fileEntry
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		fe := fileEntry{
			name:  e.Name(),
			path:  filepath.Join(m.currentPath, e.Name()),
			isDir: e.IsDir(),
			size:  size,
		}
		if e.IsDir() {
			dirs = append(dirs, fe)
		} else {
			files = append(files, fe)
		}
	}
	m.entries = append(m.entries, dirs...)
	m.entries = append(m.entries, files...)
}

func (m ExplorerModel) Update(msg tea.Msg) (ExplorerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case configEditDoneMsg:
		m.loadDir()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				if entry.isDir {
					m.currentPath = entry.path
					m.loadDir()
				} else {
					return m, editFileCmd(entry.path)
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
			if m.currentPath != m.rootPath {
				m.currentPath = filepath.Dir(m.currentPath)
				m.loadDir()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
			return m, m.openInFileManager()
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loadDir()
		}
	}
	return m, nil
}

func (m ExplorerModel) openInFileManager() tea.Cmd {
	return func() tea.Msg {
		for _, cmd := range []string{"xdg-open", "open", "nautilus", "thunar", "dolphin"} {
			if _, err := exec.LookPath(cmd); err == nil {
				exec.Command(cmd, m.currentPath).Start()
				return serverActionDoneMsg{message: "Opened in file manager"}
			}
		}
		return serverActionDoneMsg{message: "No file manager found"}
	}
}

func (m ExplorerModel) View() string {
	var b strings.Builder

	relPath, _ := filepath.Rel(m.rootPath, m.currentPath)
	if relPath == "." {
		relPath = "/"
	} else {
		relPath = "/" + relPath
	}
	b.WriteString(lipgloss.NewStyle().Foreground(Gray).Render(fmt.Sprintf("  %s", relPath)))
	b.WriteString("\n\n")

	if len(m.entries) == 0 {
		b.WriteString("  (empty directory)\n")
	}

	visibleHeight := m.height - 6
	if visibleHeight < 5 {
		visibleHeight = 15
	}

	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(m.entries) {
		end = len(m.entries)
	}

	dirStyle := lipgloss.NewStyle().Foreground(Blue)
	fileStyle := lipgloss.NewStyle().Foreground(White)
	sizeStyle := lipgloss.NewStyle().Foreground(Gray)
	selectedDirStyle := lipgloss.NewStyle().Foreground(Blue).Bold(true)
	selectedFileStyle := lipgloss.NewStyle().Foreground(Red).Bold(true)

	for i := start; i < end; i++ {
		entry := m.entries[i]
		cursor := "  "
		var style lipgloss.Style

		if i == m.cursor {
			cursor = "\u25b8 "
			if entry.isDir {
				style = selectedDirStyle
			} else {
				style = selectedFileStyle
			}
		} else {
			if entry.isDir {
				style = dirStyle
			} else {
				style = fileStyle
			}
		}

		name := entry.name
		if entry.isDir && entry.name != ".." {
			name += "/"
		}

		line := cursor + style.Render(name)
		if !entry.isDir {
			line += "  " + sizeStyle.Render(formatSize(entry.size))
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n  " + m.message + "\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  [enter] open/edit  [backspace] up  [o] file manager  [r] refresh"))
	b.WriteString("\n")

	return b.String()
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(bytes)/1024/1024)
	}
	return fmt.Sprintf("%.1fGB", float64(bytes)/1024/1024/1024)
}

func (m *ExplorerModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
