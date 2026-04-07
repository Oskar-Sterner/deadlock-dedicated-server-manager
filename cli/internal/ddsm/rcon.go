package ddsm

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/james4k/rcon"
)

type PlayerInfo struct {
	Players    int
	MaxPlayers int
}

func SendRconCommand(port int, command, password string) (string, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := rcon.Dial(addr, password)
	if err != nil {
		return "", fmt.Errorf("rcon connect failed: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(command)
	if err != nil {
		return "", fmt.Errorf("rcon send failed: %w", err)
	}

	response, _, err := conn.Read()
	if err != nil {
		return "", fmt.Errorf("rcon read failed: %w", err)
	}

	return response, nil
}

var statusRegex = regexp.MustCompile(`players\s*:\s*(\d+)\s*humans?,\s*(\d+)\s*bots?\s*\((\d+)\s*max\)`)

func QueryServerPlayers(port int) (*PlayerInfo, error) {
	response, err := SendRconCommand(port, "status", Cfg.RconPassword)
	if err != nil {
		return nil, err
	}

	matches := statusRegex.FindStringSubmatch(response)
	if matches == nil {
		return &PlayerInfo{Players: 0, MaxPlayers: 0}, nil
	}

	humans, _ := strconv.Atoi(matches[1])
	bots, _ := strconv.Atoi(matches[2])
	maxPlayers, _ := strconv.Atoi(matches[3])

	return &PlayerInfo{
		Players:    humans + bots,
		MaxPlayers: maxPlayers,
	}, nil
}
