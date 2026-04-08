package ddsm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BaseInstalled checks if the shared base game install exists.
func BaseInstalled() bool {
	exe := filepath.Join(Cfg.BaseDir, "Deadlock", "game", "bin", "win64", "deadlock.exe")
	_, err := os.Stat(exe)
	return err == nil
}

// UsesOverlay checks if a server uses an overlay mount.
func UsesOverlay(serverID string) bool {
	upper := filepath.Join(Cfg.ServersDir, serverID, "upper")
	_, err := os.Stat(upper)
	return err == nil
}

// IsOverlayMounted checks if the overlay for a server is currently mounted.
func IsOverlayMounted(serverID string) bool {
	merged := MergedPath(serverID)
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), merged)
}

// MergedPath returns the overlay merged directory path for a server.
func MergedPath(serverID string) string {
	return filepath.Join(Cfg.ServersDir, serverID, "merged")
}

// SetupOverlayDirs creates the overlay directory structure for a server.
func SetupOverlayDirs(serverID string) error {
	base := filepath.Join(Cfg.ServersDir, serverID)
	for _, dir := range []string{"upper", "work", "merged"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	return nil
}

// MountOverlay mounts the overlayfs for a server if not already mounted.
func MountOverlay(serverID string) error {
	if IsOverlayMounted(serverID) {
		return nil
	}

	upper := filepath.Join(Cfg.ServersDir, serverID, "upper")
	work := filepath.Join(Cfg.ServersDir, serverID, "work")
	merged := MergedPath(serverID)

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", Cfg.BaseDir, upper, work)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", opts, merged)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount overlay: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// UnmountOverlay unmounts the overlayfs for a server.
func UnmountOverlay(serverID string) error {
	if !IsOverlayMounted(serverID) {
		return nil
	}
	merged := MergedPath(serverID)
	cmd := exec.Command("umount", merged)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("umount overlay: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// MountAllOverlays mounts overlays for all servers that use them.
func MountAllOverlays() {
	if !BaseInstalled() {
		return
	}
	servers, err := ListServers()
	if err != nil {
		return
	}
	for _, s := range servers {
		if UsesOverlay(s.ID) {
			MountOverlay(s.ID)
		}
	}
}

// ServerVolumePath returns the path to mount into Docker for a server.
func ServerVolumePath(serverID string) string {
	if UsesOverlay(serverID) {
		return MergedPath(serverID)
	}
	return filepath.Join(Cfg.ServersDir, serverID)
}
