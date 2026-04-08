package ddsm

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/glebarez/go-sqlite"
)

var (
	db     *sql.DB
	dbOnce sync.Once
)

type ServerRow struct {
	ID          string
	Name        string
	Port        int
	Map         string
	Password    string
	SteamLogin  string
	SteamPass   string
	Steam2FA    string
	SkipUpdate  int
	Deadworks   int
	ContainerID sql.NullString
	CreatedAt   string
}

func GetDB() *sql.DB {
	dbOnce.Do(func() {
		dir := filepath.Dir(Cfg.DbPath)
		os.MkdirAll(dir, 0755)

		var err error
		db, err = sql.Open("sqlite", Cfg.DbPath+"?_journal_mode=WAL&_foreign_keys=ON")
		if err != nil {
			panic("failed to open database: " + err.Error())
		}

		migrate()
	})
	return db
}

func migrate() {
	db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS servers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			port INTEGER NOT NULL,
			[map] TEXT NOT NULL DEFAULT 'dl_streets',
			password TEXT NOT NULL DEFAULT '',
			steam_login TEXT NOT NULL,
			steam_pass TEXT NOT NULL,
			steam_2fa TEXT NOT NULL DEFAULT '',
			skip_update INTEGER NOT NULL DEFAULT 1,
			deadworks INTEGER NOT NULL DEFAULT 0,
			container_id TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)

	// Add deadworks column to existing databases
	db.Exec(`ALTER TABLE servers ADD COLUMN deadworks INTEGER NOT NULL DEFAULT 0`)
}

func GetSetting(key string) string {
	var value string
	err := GetDB().QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

func SetSetting(key, value string) {
	GetDB().Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
}

func ListServers() ([]ServerRow, error) {
	rows, err := GetDB().Query("SELECT id, name, port, [map], password, steam_login, steam_pass, steam_2fa, skip_update, deadworks, container_id, created_at FROM servers ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []ServerRow
	for rows.Next() {
		var s ServerRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Port, &s.Map, &s.Password, &s.SteamLogin, &s.SteamPass, &s.Steam2FA, &s.SkipUpdate, &s.Deadworks, &s.ContainerID, &s.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func GetServer(id string) (*ServerRow, error) {
	var s ServerRow
	err := GetDB().QueryRow("SELECT id, name, port, [map], password, steam_login, steam_pass, steam_2fa, skip_update, deadworks, container_id, created_at FROM servers WHERE id = ?", id).
		Scan(&s.ID, &s.Name, &s.Port, &s.Map, &s.Password, &s.SteamLogin, &s.SteamPass, &s.Steam2FA, &s.SkipUpdate, &s.Deadworks, &s.ContainerID, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetNextPort() int {
	var maxPort sql.NullInt64
	GetDB().QueryRow("SELECT MAX(port) FROM servers").Scan(&maxPort)
	if !maxPort.Valid {
		return 27015
	}
	return int(maxPort.Int64) + 1
}

func InsertServer(s *ServerRow) error {
	_, err := GetDB().Exec(`
		INSERT INTO servers (id, name, port, [map], password, steam_login, steam_pass, steam_2fa, skip_update, deadworks, container_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Port, s.Map, s.Password, s.SteamLogin, s.SteamPass, s.Steam2FA, s.SkipUpdate, s.Deadworks, s.ContainerID)
	return err
}

func UpdateServerContainerID(id, containerID string) error {
	_, err := GetDB().Exec("UPDATE servers SET container_id = ? WHERE id = ?", containerID, id)
	return err
}

func UpdateServerFields(id, name, mapName, password string) error {
	_, err := GetDB().Exec("UPDATE servers SET name = ?, [map] = ?, password = ? WHERE id = ?", name, mapName, password, id)
	return err
}

func DeleteServerRow(id string) error {
	_, err := GetDB().Exec("DELETE FROM servers WHERE id = ?", id)
	return err
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}
