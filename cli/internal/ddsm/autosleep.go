package ddsm

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type sleepState struct {
	emptyTimestamp *time.Time
	sleeping       bool
	waking         bool
	tcpListener    net.Listener
	udpConn        *net.UDPConn
	mu             sync.Mutex
}

var (
	sleepStates   = make(map[string]*sleepState)
	sleepStatesMu sync.Mutex
	sleepCancel   context.CancelFunc
)

func getSleepState(serverID string) *sleepState {
	sleepStatesMu.Lock()
	defer sleepStatesMu.Unlock()
	if s, ok := sleepStates[serverID]; ok {
		return s
	}
	s := &sleepState{}
	sleepStates[serverID] = s
	return s
}

func IsSleeping(serverID string) bool {
	s := getSleepState(serverID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sleeping
}

func IsWaking(serverID string) bool {
	s := getSleepState(serverID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waking
}

func StartAutoSleep() {
	if !Cfg.AutoSleep.Enabled {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sleepCancel = cancel

	interval := time.Duration(Cfg.AutoSleep.PollInterval) * time.Second
	idleTimeout := time.Duration(Cfg.AutoSleep.IdleTimeout) * time.Second

	fmt.Printf("[autosleep] Started — polling every %ds, idle timeout %ds\n", Cfg.AutoSleep.PollInterval, Cfg.AutoSleep.IdleTimeout)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pollServers(idleTimeout)
			}
		}
	}()
}

func StopAutoSleep() {
	if sleepCancel != nil {
		sleepCancel()
	}
	sleepStatesMu.Lock()
	defer sleepStatesMu.Unlock()
	for _, s := range sleepStates {
		s.mu.Lock()
		closeListeners(s)
		s.mu.Unlock()
	}
	sleepStates = make(map[string]*sleepState)
}

func pollServers(idleTimeout time.Duration) {
	servers, err := ListServers()
	if err != nil {
		return
	}

	for _, server := range servers {
		if !server.ContainerID.Valid {
			continue
		}

		s := getSleepState(server.ID)
		s.mu.Lock()

		if s.waking || s.sleeping {
			s.mu.Unlock()
			continue
		}

		info, err := GetContainerInfo(server.ContainerID.String)
		if err != nil || info.State != "running" {
			s.mu.Unlock()
			continue
		}

		players, err := QueryServerPlayers(server.Port)
		if err != nil || players == nil {
			s.mu.Unlock()
			continue
		}

		if players.Players > 0 {
			s.emptyTimestamp = nil
		} else {
			if s.emptyTimestamp == nil {
				now := time.Now()
				s.emptyTimestamp = &now
				fmt.Printf("[autosleep] %s (port %d): empty, starting idle timer\n", server.Name, server.Port)
			} else if time.Since(*s.emptyTimestamp) >= idleTimeout {
				fmt.Printf("[autosleep] %s (port %d): idle for %ds, sleeping\n", server.Name, server.Port, int(idleTimeout.Seconds()))
				s.mu.Unlock()
				sleepServer(server.ID, server.Port, server.ContainerID.String)
				continue
			}
		}

		s.mu.Unlock()
	}
}

func sleepServer(serverID string, port int, containerID string) {
	if err := StopContainer(containerID); err != nil {
		fmt.Printf("[autosleep] Failed to stop container: %v\n", err)
		return
	}

	s := getSleepState(serverID)
	s.mu.Lock()
	s.sleeping = true
	s.emptyTimestamp = nil
	s.mu.Unlock()

	time.Sleep(3 * time.Second)
	startWakeListener(serverID, port, containerID)
}

func startWakeListener(serverID string, port int, containerID string) {
	s := getSleepState(serverID)

	addr := fmt.Sprintf("0.0.0.0:%d", port)

	tcpListener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("[autosleep] Port %d TCP in use, retrying in 5s\n", port)
		time.AfterFunc(5*time.Second, func() {
			startWakeListener(serverID, port, containerID)
		})
		return
	}

	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		tcpListener.Close()
		fmt.Printf("[autosleep] Port %d UDP in use, retrying in 5s\n", port)
		time.AfterFunc(5*time.Second, func() {
			startWakeListener(serverID, port, containerID)
		})
		return
	}

	s.mu.Lock()
	s.tcpListener = tcpListener
	s.udpConn = udpConn
	s.mu.Unlock()

	fmt.Printf("[autosleep] Wake listener on port %d (TCP+UDP)\n", port)

	go func() {
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				return
			}
			conn.Close()
			fmt.Printf("[autosleep] TCP connection on port %d — waking server %s\n", port, serverID)
			wakeServer(serverID, port, containerID)
			return
		}
	}()

	go func() {
		buf := make([]byte, 1)
		for {
			_, _, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			fmt.Printf("[autosleep] UDP packet on port %d — waking server %s\n", port, serverID)
			wakeServer(serverID, port, containerID)
			return
		}
	}()
}

func wakeServer(serverID string, port int, containerID string) {
	s := getSleepState(serverID)
	s.mu.Lock()
	if s.waking {
		s.mu.Unlock()
		return
	}
	s.waking = true
	closeListeners(s)
	s.mu.Unlock()

	fmt.Printf("[autosleep] Waking server %s on port %d\n", serverID, port)

	time.Sleep(1 * time.Second)

	if err := StartContainer(containerID); err != nil {
		fmt.Printf("[autosleep] Failed to start container: %v\n", err)
	} else {
		fmt.Printf("[autosleep] Server %s started\n", serverID)
	}

	s.mu.Lock()
	s.sleeping = false
	s.waking = false
	s.emptyTimestamp = nil
	s.mu.Unlock()
}

func ManualWake(serverID string) error {
	server, err := GetServer(serverID)
	if err != nil || server == nil {
		return fmt.Errorf("server not found: %s", serverID)
	}
	if !server.ContainerID.Valid {
		return fmt.Errorf("server has no container")
	}

	s := getSleepState(serverID)
	s.mu.Lock()
	if !s.sleeping {
		s.mu.Unlock()
		return fmt.Errorf("server is not sleeping")
	}
	s.mu.Unlock()

	wakeServer(serverID, server.Port, server.ContainerID.String)
	return nil
}

func ResetSleepState(serverID string) {
	s := getSleepState(serverID)
	s.mu.Lock()
	defer s.mu.Unlock()
	closeListeners(s)
	s.sleeping = false
	s.waking = false
	s.emptyTimestamp = nil
}

func closeListeners(s *sleepState) {
	if s.tcpListener != nil {
		s.tcpListener.Close()
		s.tcpListener = nil
	}
	if s.udpConn != nil {
		s.udpConn.Close()
		s.udpConn = nil
	}
}
