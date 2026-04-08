package ddsm

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var dockerClient *client.Client

func DockerClient() *client.Client {
	if dockerClient == nil {
		var err error
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			panic("failed to create Docker client: " + err.Error())
		}
	}
	return dockerClient
}

type ContainerInfo struct {
	ID        string
	Status    string
	State     string
	StartedAt string
}

type ContainerStats struct {
	CPUPercent    float64
	MemoryMB      float64
	MemoryLimitMB float64
}

func GetContainerInfo(containerID string) (*ContainerInfo, error) {
	ctx := context.Background()
	info, err := DockerClient().ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}
	return &ContainerInfo{
		ID:        info.ID[:12],
		Status:    info.State.Status,
		State:     info.State.Status,
		StartedAt: info.State.StartedAt,
	}, nil
}

func GetContainerStats(containerID string) (*ContainerStats, error) {
	ctx := context.Background()
	resp, err := DockerClient().ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs     uint64 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)
	numCPUs := stats.CPUStats.OnlineCPUs
	if numCPUs == 0 {
		numCPUs = 1
	}

	var cpuPercent float64
	if systemDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(numCPUs) * 100
	}

	return &ContainerStats{
		CPUPercent:    float64(int(cpuPercent*10)) / 10,
		MemoryMB:      float64(stats.MemoryStats.Usage) / 1024 / 1024,
		MemoryLimitMB: float64(stats.MemoryStats.Limit) / 1024 / 1024,
	}, nil
}

func CreateContainer(name string, port int, env map[string]string, volumePath string, useOverlay bool) (string, error) {
	ctx := context.Background()

	envSlice := make([]string, 0, len(env)+4)
	for k, v := range env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	envSlice = append(envSlice,
		"PROTON_LOG=0",
		"PROTON_NO_WRITE_WATCH=1",
		"WINEDLLOVERRIDES=winedbg.exe=d",
		"DISPLAY=:99",
	)

	portStr := fmt.Sprintf("%d", port)
	portTCP := nat.Port(portStr + "/tcp")
	portUDP := nat.Port(portStr + "/udp")

	cfg := &container.Config{
		Image: Cfg.DockerImage,
		Env:   envSlice,
		ExposedPorts: nat.PortSet{
			portTCP: struct{}{},
			portUDP: struct{}{},
		},
	}

	// For overlay servers, skip the destructive chown -R in the entrypoint.
	// The base files are already owned by steam:steam and chown -R triggers
	// a full copy-up on overlayfs, duplicating the entire 34GB game install.
	if useOverlay {
		cfg.Entrypoint = []string{"/bin/bash", "-c",
			"rm -f /tmp/.X99-lock /tmp/.X11-unix/X99 && " +
				"Xvfb :99 -screen 0 640x480x8 -nolisten tcp -nolisten unix +extension GLX & sleep 1 && " +
				"chmod a+x /app/start.sh && " +
				"exec gosu steam /app/start.sh"}
	}

	resp, err := DockerClient().ContainerCreate(ctx,
		cfg,
		&container.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/app", volumePath),
				"/etc/localtime:/etc/localtime:ro",
				"/etc/machine-id:/etc/machine-id:ro",
			},
			PortBindings: nat.PortMap{
				portTCP: []nat.PortBinding{{HostPort: portStr}},
				portUDP: []nat.PortBinding{{HostPort: portStr}},
			},
			RestartPolicy: container.RestartPolicy{Name: "always"},
		},
		nil, nil, name,
	)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func StartContainer(containerID string) error {
	return DockerClient().ContainerStart(context.Background(), containerID, container.StartOptions{})
}

func StopContainer(containerID string) error {
	timeout := 10
	return DockerClient().ContainerStop(context.Background(), containerID, container.StopOptions{Timeout: &timeout})
}

func RestartContainer(containerID string) error {
	timeout := 10
	return DockerClient().ContainerRestart(context.Background(), containerID, container.StopOptions{Timeout: &timeout})
}

func RemoveContainer(containerID string) error {
	StopContainer(containerID)
	return DockerClient().ContainerRemove(context.Background(), containerID, container.RemoveOptions{})
}

func StreamLogs(containerID string, tail int, done <-chan struct{}) (<-chan string, error) {
	ctx, cancel := context.WithCancel(context.Background())

	reader, err := DockerClient().ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       fmt.Sprintf("%d", tail),
		Timestamps: false,
	})
	if err != nil {
		cancel()
		return nil, err
	}

	ch := make(chan string, 64)

	go func() {
		defer close(ch)
		defer reader.Close()
		defer cancel()

		header := make([]byte, 8)
		for {
			select {
			case <-done:
				return
			default:
			}

			_, err := io.ReadFull(reader, header)
			if err != nil {
				return
			}

			size := binary.BigEndian.Uint32(header[4:8])
			payload := make([]byte, size)
			_, err = io.ReadFull(reader, payload)
			if err != nil {
				return
			}

			line := string(payload)
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}

			select {
			case ch <- line:
			case <-done:
				return
			}
		}
	}()

	go func() {
		<-done
		cancel()
	}()

	return ch, nil
}

func WaitForContainerRunning(containerID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		info, err := GetContainerInfo(containerID)
		if err == nil && info.State == "running" {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("container %s did not start within %v", containerID, timeout)
}
