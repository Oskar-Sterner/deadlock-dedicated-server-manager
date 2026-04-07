package ddsm

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/docker/docker/api/types/image"
)

type CheckResult struct {
	Name   string
	Status string // "pass", "fail", "warn"
	Detail string
}

func RunDoctor() []CheckResult {
	var results []CheckResult

	results = append(results, checkDocker())
	results = append(results, checkImage())
	results = append(results, checkDiskSpace())
	results = append(results, checkServers()...)

	return results
}

func checkDocker() CheckResult {
	_, err := DockerClient().Ping(context.Background())
	if err != nil {
		return CheckResult{"Docker daemon", "fail", fmt.Sprintf("Cannot connect: %v. Is Docker running?", err)}
	}
	return CheckResult{"Docker daemon", "pass", "Connected via /var/run/docker.sock"}
}

func checkImage() CheckResult {
	ctx := context.Background()
	images, err := DockerClient().ImageList(ctx, image.ListOptions{})
	if err != nil {
		return CheckResult{"Docker image", "fail", fmt.Sprintf("Cannot list images: %v", err)}
	}

	target := Cfg.DockerImage
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == target+":latest" || tag == target {
				return CheckResult{"Docker image", "pass", fmt.Sprintf("Image '%s' found", target)}
			}
		}
	}
	return CheckResult{"Docker image", "fail", fmt.Sprintf("Image '%s' not found. Build it first.", target)}
}

func checkDiskSpace() CheckResult {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(Cfg.ServersDir, &stat); err != nil {
		if os.IsNotExist(err) {
			return CheckResult{"Disk space", "warn", fmt.Sprintf("Servers dir '%s' does not exist yet", Cfg.ServersDir)}
		}
		return CheckResult{"Disk space", "fail", fmt.Sprintf("Cannot stat: %v", err)}
	}
	freeGB := float64(stat.Bavail*uint64(stat.Bsize)) / 1024 / 1024 / 1024
	if freeGB < 5 {
		return CheckResult{"Disk space", "warn", fmt.Sprintf("%.1f GB free at %s (recommend 5+ GB)", freeGB, Cfg.ServersDir)}
	}
	return CheckResult{"Disk space", "pass", fmt.Sprintf("%.1f GB free at %s", freeGB, Cfg.ServersDir)}
}

func checkServers() []CheckResult {
	servers, err := ListServers()
	if err != nil {
		return []CheckResult{{"Server database", "fail", fmt.Sprintf("Cannot read: %v", err)}}
	}

	if len(servers) == 0 {
		return []CheckResult{{"Servers", "pass", "No servers configured"}}
	}

	var results []CheckResult
	for _, s := range servers {
		name := fmt.Sprintf("Server '%s' (port %d)", s.Name, s.Port)
		if !s.ContainerID.Valid {
			results = append(results, CheckResult{name, "warn", "No container ID in database"})
			continue
		}

		info, err := GetContainerInfo(s.ContainerID.String)
		if err != nil {
			results = append(results, CheckResult{name, "fail", fmt.Sprintf("Container %s not found", s.ContainerID.String[:12])})
			continue
		}

		if info.State == "running" {
			_, err := QueryServerPlayers(s.Port)
			if err != nil {
				results = append(results, CheckResult{name, "warn", fmt.Sprintf("Running but RCON failed: %v", err)})
			} else {
				results = append(results, CheckResult{name, "pass", "Running, RCON OK"})
			}
		} else {
			results = append(results, CheckResult{name, "pass", fmt.Sprintf("Container state: %s", info.State)})
		}
	}
	return results
}

func PrintDoctorResults(results []CheckResult) {
	for _, r := range results {
		var icon string
		switch r.Status {
		case "pass":
			icon = "✓"
		case "fail":
			icon = "✗"
		case "warn":
			icon = "!"
		}
		fmt.Printf("  [%s] %s — %s\n", icon, r.Name, r.Detail)
	}

	fails := 0
	for _, r := range results {
		if r.Status == "fail" {
			fails++
		}
	}
	if fails > 0 {
		fmt.Printf("\n  %d check(s) failed.\n", fails)
	} else {
		fmt.Println("\n  All checks passed.")
	}
}
