package ddsm

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/google/uuid"
)

const defaultStartScript = `#!/bin/bash

die() {
    echo "$0 failed, keeping container alive for debugging..."
    while true; do sleep 10; done
}

if [ "$(id -u)" != "1000" ]; then
    echo "ERROR: Script must run as the steam user (uid 1000)"
    die
fi

# --- Build launch arguments ---

ACTUAL_PORT="${PORT:-27015}"
ARGS="-port ${ACTUAL_PORT}"

[ -n "${SERVER_PASSWORD}" ] && ARGS="${ARGS} +sv_password ${SERVER_PASSWORD}"
[ -n "${MAP}" ]             && ARGS="${ARGS} +map ${MAP}"
ARGS="${ARGS} +rcon_password ddsm_rcon_secret"

ARGS="-dedicated -usercon -ip 0.0.0.0 -convars_visible_by_default -allow_no_lobby_connect -novid ${ARGS}"

# --- Validate game directory ---

DEADLOCK_DIR=/app/Deadlock
DEADLOCK_EXE="${DEADLOCK_DIR}/game/bin/win64/deadlock.exe"

mkdir -p "${DEADLOCK_DIR}"
DIR_PERM=$(stat -c "%u:%g:%a" "${DEADLOCK_DIR}")
if [ "${DIR_PERM}" != "1000:1000:755" ]; then
    echo "ERROR: ${DEADLOCK_DIR} has unexpected permissions ${DIR_PERM} (expected 1000:1000:755)"
    die
fi

# --- Download or update game files ---

if [ -f "${DEADLOCK_EXE}" ] && [ "${SKIP_UPDATE}" = "1" ]; then
    echo "Game installed and SKIP_UPDATE=1, skipping SteamCMD"
elif [ -n "${STEAM_LOGIN}" ]; then
    echo "Updating game files via SteamCMD..."
    STEAMCMD="${STEAM_HOME}/steamcmd/steamcmd.sh"
    ${STEAMCMD} \
        +@sSteamCmdForcePlatformType windows \
        +force_install_dir "${DEADLOCK_DIR}" \
        +login "${STEAM_LOGIN}" "${STEAM_PASSWORD}" "${STEAM_2FA_CODE}" \
        +app_update "${APPID}" validate \
        +quit || die
else
    echo "No STEAM_LOGIN set and game not installed"
    die
fi

if [ ! -f "${DEADLOCK_EXE}" ]; then
    echo "ERROR: ${DEADLOCK_EXE} not found after install"
    die
fi

# --- Launch server ---

CMD="${PROTON} run ${DEADLOCK_EXE} ${ARGS}"
echo "Starting Deadlock server: ${CMD}"
exec ${CMD}
`

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
	useOverlay := BaseInstalled()

	var volumePath string
	if useOverlay {
		if err := SetupOverlayDirs(id); err != nil {
			return nil, fmt.Errorf("failed to setup overlay: %w", err)
		}
		if err := MountOverlay(id); err != nil {
			return nil, fmt.Errorf("failed to mount overlay: %w", err)
		}
		volumePath = MergedPath(id)
	} else {
		volumePath = filepath.Join(Cfg.ServersDir, id)
		if err := os.MkdirAll(volumePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create server directory: %w", err)
		}
	}

	// Write embedded start.sh to the volume (owned by steam for overlay compat)
	dest := filepath.Join(volumePath, "start.sh")
	os.WriteFile(dest, []byte(defaultStartScript), 0755)
	os.Chown(dest, 1000, 1000)

	containerName := fmt.Sprintf("deadlock-%s", id[:8])
	skipUpdate := "0"
	if useOverlay {
		skipUpdate = "1"
	}
	env := map[string]string{
		"PORT":            fmt.Sprintf("%d", opts.Port),
		"MAP":             opts.Map,
		"SERVER_PASSWORD": opts.Password,
		"STEAM_LOGIN":     opts.SteamLogin,
		"STEAM_PASSWORD":  opts.SteamPass,
		"STEAM_2FA_CODE":  opts.Steam2FA,
		"SKIP_UPDATE":     skipUpdate,
	}

	containerID, err := CreateContainer(containerName, opts.Port, env, volumePath, useOverlay)
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

	if UsesOverlay(id) {
		UnmountOverlay(id)
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
	if UsesOverlay(id) {
		if err := MountOverlay(id); err != nil {
			return fmt.Errorf("failed to mount overlay: %w", err)
		}
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

// UpdateBase downloads or updates the shared base game install via a temporary Docker container.
func UpdateBase(steamLogin, steamPass, steam2FA string) error {
	if err := os.MkdirAll(Cfg.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Write start.sh to base dir
	os.WriteFile(filepath.Join(Cfg.BaseDir, "start.sh"), []byte(defaultStartScript), 0755)

	// Remove leftover update container
	RemoveContainer("ddsm-update-base")

	ctx := context.Background()
	envSlice := []string{
		"STEAM_LOGIN=" + steamLogin,
		"STEAM_PASSWORD=" + steamPass,
		"STEAM_2FA_CODE=" + steam2FA,
	}

	// Create container that only runs SteamCMD, then exits
	resp, err := DockerClient().ContainerCreate(ctx,
		&container.Config{
			Image:      Cfg.DockerImage,
			Env:        envSlice,
			Entrypoint: []string{"/bin/bash", "-c"},
			Cmd: []string{
				"mkdir -p /app/Deadlock && " +
					"chown -R steam:steam /app/Deadlock && " +
					"gosu steam ${STEAM_HOME}/steamcmd/steamcmd.sh " +
					"+@sSteamCmdForcePlatformType windows " +
					"+force_install_dir /app/Deadlock " +
					"+login \"${STEAM_LOGIN}\" \"${STEAM_PASSWORD}\" \"${STEAM_2FA_CODE}\" " +
					"+app_update 1422450 validate " +
					"+quit && " +
					"echo DDSM_UPDATE_COMPLETE",
			},
		},
		&container.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/app", Cfg.BaseDir),
			},
		},
		nil, nil, "ddsm-update-base",
	)
	if err != nil {
		return fmt.Errorf("failed to create update container: %w", err)
	}

	defer RemoveContainer(resp.ID)

	if err := StartContainer(resp.ID); err != nil {
		return fmt.Errorf("failed to start update container: %w", err)
	}

	// Stream logs to show download progress
	done := make(chan struct{})
	ch, err := StreamLogs(resp.ID, 0, done)
	if err != nil {
		return fmt.Errorf("failed to stream logs: %w", err)
	}

	success := false
	for line := range ch {
		fmt.Println(line)
		if strings.Contains(line, "DDSM_UPDATE_COMPLETE") {
			success = true
		}
	}

	// Fix ownership for overlayfs compatibility
	exec.Command("chown", "-R", "1000:1000", Cfg.BaseDir).Run()

	if !success {
		return fmt.Errorf("update did not complete successfully — check logs above")
	}
	return nil
}
