package ddsm

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type ServerCreateOpts struct {
	Name       string
	Port       int
	Map        string
	Password   string
	SteamLogin string
	SteamPass  string
	Steam2FA   string
}

type ServerStatus struct {
	ServerRow
	Status     string
	StartedAt  string
	Stats      *ContainerStats
	Players    int
	MaxPlayers int
}

func CreateServer(opts ServerCreateOpts) (*ServerRow, error) {
	id := uuid.New().String()
	volumePath := filepath.Join(Cfg.ServersDir, id)

	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create server directory: %w", err)
	}

	startScript := filepath.Join(".", "start.sh")
	if data, err := os.ReadFile(startScript); err == nil {
		dest := filepath.Join(volumePath, "start.sh")
		os.WriteFile(dest, data, 0755)
	}

	containerName := fmt.Sprintf("deadlock-%s", id[:8])
	env := map[string]string{
		"PORT":            fmt.Sprintf("%d", opts.Port),
		"MAP":             opts.Map,
		"SERVER_PASSWORD": opts.Password,
		"STEAM_LOGIN":     opts.SteamLogin,
		"STEAM_PASSWORD":  opts.SteamPass,
		"STEAM_2FA_CODE":  opts.Steam2FA,
		"SKIP_UPDATE":     "0",
	}

	containerID, err := CreateContainer(containerName, opts.Port, env, volumePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	server := &ServerRow{
		ID:          id,
		Name:        opts.Name,
		Port:        opts.Port,
		Map:         opts.Map,
		Password:    opts.Password,
		SteamLogin:  opts.SteamLogin,
		SteamPass:   opts.SteamPass,
		Steam2FA:    opts.Steam2FA,
		SkipUpdate:  0,
		ContainerID: sql.NullString{String: containerID, Valid: true},
	}

	if err := InsertServer(server); err != nil {
		return nil, fmt.Errorf("failed to insert server: %w", err)
	}

	if err := StartContainer(containerID); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return server, nil
}

func DeleteServer(id string, deleteFiles bool) error {
	server, err := GetServer(id)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", id)
	}

	if server.ContainerID.Valid {
		RemoveContainer(server.ContainerID.String)
	}

	if deleteFiles {
		volumePath := filepath.Join(Cfg.ServersDir, id)
		os.RemoveAll(volumePath)
	}

	return DeleteServerRow(id)
}

func GetServerStatus(server *ServerRow) *ServerStatus {
	status := &ServerStatus{
		ServerRow: *server,
		Status:    "unknown",
	}

	if !server.ContainerID.Valid {
		return status
	}

	if IsSleeping(server.ID) {
		status.Status = "sleeping"
		return status
	}
	if IsWaking(server.ID) {
		status.Status = "waking"
		return status
	}

	info, err := GetContainerInfo(server.ContainerID.String)
	if err != nil {
		return status
	}

	status.Status = info.State
	status.StartedAt = info.StartedAt

	if info.State == "running" {
		stats, err := GetContainerStats(server.ContainerID.String)
		if err == nil {
			status.Stats = stats
		}

		players, err := QueryServerPlayers(server.Port)
		if err == nil {
			status.Players = players.Players
			status.MaxPlayers = players.MaxPlayers
		}
	}

	return status
}

func ListServerStatuses() ([]*ServerStatus, error) {
	servers, err := ListServers()
	if err != nil {
		return nil, err
	}

	statuses := make([]*ServerStatus, 0, len(servers))
	for i := range servers {
		statuses = append(statuses, GetServerStatus(&servers[i]))
	}
	return statuses, nil
}

func StartServer(id string) error {
	server, err := GetServer(id)
	if err != nil || server == nil {
		return fmt.Errorf("server not found: %s", id)
	}
	if !server.ContainerID.Valid {
		return fmt.Errorf("server has no container: %s", id)
	}
	ResetSleepState(id)
	return StartContainer(server.ContainerID.String)
}

func StopServer(id string) error {
	server, err := GetServer(id)
	if err != nil || server == nil {
		return fmt.Errorf("server not found: %s", id)
	}
	if !server.ContainerID.Valid {
		return fmt.Errorf("server has no container: %s", id)
	}
	ResetSleepState(id)
	return StopContainer(server.ContainerID.String)
}

func RestartServer(id string) error {
	server, err := GetServer(id)
	if err != nil || server == nil {
		return fmt.Errorf("server not found: %s", id)
	}
	if !server.ContainerID.Valid {
		return fmt.Errorf("server has no container: %s", id)
	}
	ResetSleepState(id)
	return RestartContainer(server.ContainerID.String)
}

func ForEachServer(action func(string) error) error {
	servers, err := ListServers()
	if err != nil {
		return err
	}
	for _, s := range servers {
		if err := action(s.ID); err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", s.Name, err)
		}
	}
	return nil
}
