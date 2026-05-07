package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const dirName = ".hubsaude"
const dbName = "runner.db"

var DB *sql.DB

type Process struct {
	Name   string
	PID    int
	Port   int
	Status string
}

func HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
	}
	return filepath.Join(home, dirName), nil
}

func EnsureHomeDir() error {
	dir, err := HomeDir()
	if err != nil {
		return err
	}

	return os.MkdirAll(dir, 0755)
}

func InitDatabase() error {
	dir, err := HomeDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(dir, dbName)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir banco de dados: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("erro ao conectar ao banco de dados: %w", err)
	}

	DB = db

	return createSchema()
}

func createSchema() error {
	schema := `
	-- Armazena informações sobre o runtime Java
	CREATE TABLE IF NOT EXISTS java_runtime (
		id      INTEGER PRIMARY KEY,
		path    TEXT NOT NULL,
		version TEXT NOT NULL,
		source  TEXT NOT NULL, -- 'system' ou 'downloaded'
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Armazena informações sobre processos gerenciados pelo CLI
	CREATE TABLE IF NOT EXISTS processes (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL UNIQUE, -- 'assinador' ou 'simulador'
		pid        INTEGER,
		port       INTEGER,
		status     TEXT NOT NULL DEFAULT 'stopped', -- 'running' ou 'stopped'
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("erro ao criar schema: %w", err)
	}

	return nil
}

func SaveProcess(process Process) error {
	if DB == nil {
		return fmt.Errorf("banco de dados não inicializado")
	}

	_, err := DB.Exec(`
		INSERT INTO processes (name, pid, port, status, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(name) DO UPDATE SET
			pid = excluded.pid,
			port = excluded.port,
			status = excluded.status,
			updated_at = CURRENT_TIMESTAMP
	`, process.Name, process.PID, process.Port, process.Status)
	if err != nil {
		return fmt.Errorf("erro ao salvar processo %s: %w", process.Name, err)
	}

	return nil
}

// Close fecha a conexão com o banco de dados
func Close() {
	if DB != nil {
		DB.Close()
	}
}
