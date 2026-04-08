package ddsm

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type AutoSleepConfig struct {
	Enabled      bool `yaml:"enabled"`
	IdleTimeout  int  `yaml:"idle_timeout"`
	PollInterval int  `yaml:"poll_interval"`
}

type Config struct {
	ServerIP     string          `yaml:"server_ip"`
	RconPassword string          `yaml:"rcon_password"`
	ServersDir   string          `yaml:"servers_dir"`
	DockerImage  string          `yaml:"docker_image"`
	DbPath       string          `yaml:"db_path"`
	BaseDir      string          `yaml:"base_dir"`
	AutoSleep    AutoSleepConfig `yaml:"autosleep"`
}

var Cfg Config

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		ServerIP:     "0.0.0.0",
		RconPassword: "ddsm_rcon_secret",
		ServersDir:   "/opt/deadlock-servers",
		DockerImage:  "deadlock-server",
		DbPath:       filepath.Join(home, "deadlock-dedicated-server-manager", "data", "manager.db"),
		BaseDir:      "/opt/deadlock-base",
		AutoSleep: AutoSleepConfig{
			Enabled:      true,
			IdleTimeout:  300,
			PollInterval: 15,
		},
	}
}

func LoadConfig() error {
	Cfg = DefaultConfig()

	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, ".ddsm", "config.yaml"),
		"/etc/ddsm/config.yaml",
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, &Cfg); err != nil {
			return err
		}
		break
	}

	if v := os.Getenv("DDSM_SERVER_IP"); v != "" {
		Cfg.ServerIP = v
	}
	if v := os.Getenv("DDSM_RCON_PASSWORD"); v != "" {
		Cfg.RconPassword = v
	}
	if v := os.Getenv("DDSM_SERVERS_DIR"); v != "" {
		Cfg.ServersDir = v
	}
	if v := os.Getenv("DDSM_DOCKER_IMAGE"); v != "" {
		Cfg.DockerImage = v
	}
	if v := os.Getenv("DDSM_DB_PATH"); v != "" {
		Cfg.DbPath = v
	}
	if v := os.Getenv("DDSM_BASE_DIR"); v != "" {
		Cfg.BaseDir = v
	}
	if v := os.Getenv("DDSM_AUTOSLEEP_ENABLED"); v != "" {
		Cfg.AutoSleep.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("DDSM_AUTOSLEEP_IDLE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			Cfg.AutoSleep.IdleTimeout = n
		}
	}
	if v := os.Getenv("DDSM_AUTOSLEEP_POLL_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			Cfg.AutoSleep.PollInterval = n
		}
	}

	return nil
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ddsm", "config.yaml")
}

func EnsureConfigDir() error {
	home, _ := os.UserHomeDir()
	return os.MkdirAll(filepath.Join(home, ".ddsm"), 0755)
}

func WriteConfigFile(path string) error {
	data, err := yaml.Marshal(Cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
