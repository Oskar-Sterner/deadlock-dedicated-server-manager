package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/ddsm"
	"github.com/Oskar-Sterner/deadlock-dedicated-server-manager/cli/internal/tui"
	"gopkg.in/yaml.v3"
)

const version = "0.1.0"

const usage = `DDSM — Deadlock Dedicated Server Manager v%s

Usage:
  ddsm                     Launch interactive TUI
  ddsm --help              Show this help message
  ddsm status              List all servers with status
  ddsm create              Create a new server (interactive)
  ddsm start [id|all]      Start server(s)
  ddsm stop [id|all]       Stop server(s)
  ddsm restart [id|all]    Restart server(s)
  ddsm delete [id]         Delete a server
  ddsm logs [id]           Tail live server logs
  ddsm rcon [id] [cmd]     Execute RCON command
  ddsm attach [id]         Attach to server container
  ddsm config              Show current configuration
  ddsm config edit         Open config in $EDITOR
  ddsm update-base         Download/update shared game files
  ddsm doctor              Run health diagnostics
`

func main() {
	if err := ddsm.LoadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	ddsm.EnsureConfigDir()

	if len(os.Args) < 2 {
		ddsm.MountAllOverlays()
		ddsm.StartAutoSleep()
		defer ddsm.StopAutoSleep()
		defer ddsm.CloseDB()

		p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	switch os.Args[1] {
	case "--help", "-h", "help":
		fmt.Printf(usage, version)

	case "status":
		cmdStatus()

	case "create":
		cmdCreate()

	case "start":
		cmdStartStopRestart("start", ddsm.StartServer)

	case "stop":
		cmdStartStopRestart("stop", ddsm.StopServer)

	case "restart":
		cmdStartStopRestart("restart", ddsm.RestartServer)

	case "delete":
		cmdDelete()

	case "logs":
		cmdLogs()

	case "rcon":
		cmdRcon()

	case "attach":
		cmdAttach()

	case "config":
		cmdConfig()

	case "update-base":
		cmdUpdateBase()

	case "doctor":
		cmdDoctor()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'ddsm --help' for usage.\n", os.Args[1])
		os.Exit(1)
	}
}

func cmdStatus() {
	statuses, err := ddsm.ListServerStatuses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(statuses) == 0 {
		fmt.Println("No servers configured. Run 'ddsm create' to add one.")
		return
	}

	fmt.Printf("%-36s %-24s %-12s %-8s %-12s %-10s %-12s\n", "ID", "NAME", "STATUS", "PORT", "PLAYERS", "CPU", "MEMORY")
	fmt.Println(strings.Repeat("─", 118))
	for _, s := range statuses {
		id := s.ID
		if len(id) > 36 {
			id = id[:36]
		}
		name := s.Name
		if len(name) > 24 {
			name = name[:21] + "..."
		}

		players := "—"
		if s.Status == "running" {
			players = fmt.Sprintf("%d/%d", s.Players, s.MaxPlayers)
		}

		cpu := "—"
		mem := "—"
		if s.Stats != nil {
			cpu = fmt.Sprintf("%.1f%%", s.Stats.CPUPercent)
			mem = fmt.Sprintf("%.0fMB", s.Stats.MemoryMB)
		}

		fmt.Printf("%-36s %-24s %-12s %-8d %-12s %-10s %-12s\n", id, name, s.Status, s.Port, players, cpu, mem)
	}
}

func cmdCreate() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Server name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Fprintln(os.Stderr, "Name is required")
		os.Exit(1)
	}

	nextPort := ddsm.GetNextPort()
	fmt.Printf("Port [%d]: ", nextPort)
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	port := nextPort
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	fmt.Print("Map [dl_streets]: ")
	mapName, _ := reader.ReadString('\n')
	mapName = strings.TrimSpace(mapName)
	if mapName == "" {
		mapName = "dl_streets"
	}

	fmt.Print("Server password (optional): ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	defaultLogin := ddsm.Cfg.SteamLogin
	if defaultLogin != "" {
		fmt.Printf("Steam login [%s]: ", defaultLogin)
	} else {
		fmt.Print("Steam login: ")
	}
	steamLogin, _ := reader.ReadString('\n')
	steamLogin = strings.TrimSpace(steamLogin)
	if steamLogin == "" {
		steamLogin = defaultLogin
	}
	if steamLogin == "" {
		fmt.Fprintln(os.Stderr, "Steam login is required")
		os.Exit(1)
	}

	defaultPass := ddsm.Cfg.SteamPassword
	if defaultPass != "" {
		fmt.Printf("Steam password [%s]: ", strings.Repeat("*", len(defaultPass)))
	} else {
		fmt.Print("Steam password: ")
	}
	steamPass, _ := reader.ReadString('\n')
	steamPass = strings.TrimSpace(steamPass)
	if steamPass == "" {
		steamPass = defaultPass
	}
	if steamPass == "" {
		fmt.Fprintln(os.Stderr, "Steam password is required")
		os.Exit(1)
	}

	fmt.Print("Steam 2FA code (optional): ")
	steam2FA, _ := reader.ReadString('\n')
	steam2FA = strings.TrimSpace(steam2FA)

	fmt.Println("\nCreating server...")
	server, err := ddsm.CreateServer(ddsm.ServerCreateOpts{
		Name:       name,
		Port:       port,
		Map:        mapName,
		Password:   password,
		SteamLogin: steamLogin,
		SteamPass:  steamPass,
		Steam2FA:   steam2FA,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Server '%s' created (ID: %s, port: %d)\n", server.Name, server.ID[:8], server.Port)
}

func cmdStartStopRestart(action string, fn func(string) error) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: ddsm %s [id|all]\n", action)
		os.Exit(1)
	}
	target := os.Args[2]

	if target == "all" {
		fmt.Printf("%sing all servers...\n", capitalize(action))
		ddsm.ForEachServer(func(id string) error {
			server, _ := ddsm.GetServer(id)
			if server != nil {
				fmt.Printf("  %sing %s...\n", capitalize(action), server.Name)
			}
			return fn(id)
		})
		fmt.Println("Done.")
		return
	}

	server := resolveServer(target)
	fmt.Printf("%sing %s...\n", capitalize(action), server.Name)
	if err := fn(server.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done.")
}

func cmdDelete() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ddsm delete [id]")
		os.Exit(1)
	}
	server := resolveServer(os.Args[2])

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Delete server '%s' (%s)? This cannot be undone.\n", server.Name, server.ID[:8])
	fmt.Print("Also delete server files? [y/N]: ")
	answer, _ := reader.ReadString('\n')
	deleteFiles := strings.TrimSpace(strings.ToLower(answer)) == "y"

	fmt.Print("Type the server name to confirm: ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(confirm) != server.Name {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	if err := ddsm.DeleteServer(server.ID, deleteFiles); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server deleted.")
}

func cmdLogs() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ddsm logs [id]")
		os.Exit(1)
	}
	server := resolveServer(os.Args[2])
	if !server.ContainerID.Valid {
		fmt.Fprintln(os.Stderr, "Server has no container")
		os.Exit(1)
	}

	done := make(chan struct{})
	ch, err := ddsm.StreamLogs(server.ContainerID.String, 200, done)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("--- Logs for %s (port %d) — Ctrl+C to stop ---\n", server.Name, server.Port)
	for line := range ch {
		fmt.Println(line)
	}
}

func cmdRcon() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: ddsm rcon [id] [command...]")
		os.Exit(1)
	}
	server := resolveServer(os.Args[2])
	command := strings.Join(os.Args[3:], " ")

	response, err := ddsm.SendRconCommand(server.Port, command, ddsm.Cfg.RconPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RCON error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(response)
}

func cmdAttach() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ddsm attach [id]")
		os.Exit(1)
	}
	server := resolveServer(os.Args[2])
	if !server.ContainerID.Valid {
		fmt.Fprintln(os.Stderr, "Server has no container")
		os.Exit(1)
	}

	cmd := exec.Command("docker", "exec", "-it", server.ContainerID.String, "/bin/bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Attach failed: %v\n", err)
		os.Exit(1)
	}
}

func cmdConfig() {
	if len(os.Args) > 2 && os.Args[2] == "edit" {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}
		path := ddsm.ConfigPath()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			data, _ := yaml.Marshal(ddsm.Cfg)
			os.WriteFile(path, data, 0644)
		}
		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Editor failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Config file: %s\n\n", ddsm.ConfigPath())
		data, _ := yaml.Marshal(ddsm.Cfg)
		fmt.Print(string(data))
	}
}

func cmdUpdateBase() {
	if ddsm.BaseInstalled() {
		fmt.Printf("Base install found at %s\n", ddsm.Cfg.BaseDir)
		fmt.Print("Re-download/update? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)

	defaultLogin := ddsm.Cfg.SteamLogin
	if defaultLogin != "" {
		fmt.Printf("Steam login [%s]: ", defaultLogin)
	} else {
		fmt.Print("Steam login: ")
	}
	steamLogin, _ := reader.ReadString('\n')
	steamLogin = strings.TrimSpace(steamLogin)
	if steamLogin == "" {
		steamLogin = defaultLogin
	}
	if steamLogin == "" {
		fmt.Fprintln(os.Stderr, "Steam login is required")
		os.Exit(1)
	}

	defaultPass := ddsm.Cfg.SteamPassword
	if defaultPass != "" {
		fmt.Printf("Steam password [%s]: ", strings.Repeat("*", len(defaultPass)))
	} else {
		fmt.Print("Steam password: ")
	}
	steamPass, _ := reader.ReadString('\n')
	steamPass = strings.TrimSpace(steamPass)
	if steamPass == "" {
		steamPass = defaultPass
	}
	if steamPass == "" {
		fmt.Fprintln(os.Stderr, "Steam password is required")
		os.Exit(1)
	}

	fmt.Print("Steam 2FA code (optional): ")
	steam2FA, _ := reader.ReadString('\n')
	steam2FA = strings.TrimSpace(steam2FA)

	fmt.Printf("\nDownloading game files to %s...\n", ddsm.Cfg.BaseDir)
	fmt.Println("This will take a while (~34GB download).")
	fmt.Println()

	if err := ddsm.UpdateBase(steamLogin, steamPass, steam2FA); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\nBase install updated. New servers will use shared game files via overlay.")
}

func cmdDoctor() {
	fmt.Println("Running health checks...")
	results := ddsm.RunDoctor()
	ddsm.PrintDoctorResults(results)
}

func resolveServer(input string) *ddsm.ServerRow {
	server, err := ddsm.GetServer(input)
	if err == nil && server != nil {
		return server
	}

	servers, err := ddsm.ListServers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var matches []ddsm.ServerRow
	for _, s := range servers {
		if strings.HasPrefix(s.ID, input) {
			matches = append(matches, s)
		}
	}

	if len(matches) == 0 {
		fmt.Fprintf(os.Stderr, "No server found matching '%s'\n", input)
		os.Exit(1)
	}
	if len(matches) > 1 {
		fmt.Fprintf(os.Stderr, "Ambiguous ID '%s' — matches %d servers. Use more characters.\n", input, len(matches))
		os.Exit(1)
	}

	return &matches[0]
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
