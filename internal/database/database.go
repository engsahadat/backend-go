package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB is the global database connection pool.
var DB *sql.DB

// Init opens (or creates) the SQLite database and runs migrations.
func Init() error {
	// Store the DB file next to the binary / in project root.
	dbDir := "data"
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dbDir, "aiemployee.db")

	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	// Connection pool tuning for high performance.
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if err := migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	log.Printf("✅ Database ready (%s)\n", dbPath)
	return nil
}

// migrate creates tables if they don't exist.
func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id                 INTEGER PRIMARY KEY AUTOINCREMENT,
		email              TEXT    NOT NULL UNIQUE,
		name               TEXT    NOT NULL DEFAULT '',
		password_hash      TEXT    NOT NULL DEFAULT '',
		avatar_url         TEXT    NOT NULL DEFAULT '',
		provider           TEXT    NOT NULL DEFAULT 'email',
		provider_id        TEXT    NOT NULL DEFAULT '',
		is_verified        INTEGER NOT NULL DEFAULT 0,
		verification_token TEXT    NOT NULL DEFAULT '',
		created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider, provider_id);

	CREATE TABLE IF NOT EXISTS chat_history (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role       TEXT    NOT NULL,
		content    TEXT    NOT NULL,
		metadata   TEXT    NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_chat_user ON chat_history(user_id);
	`
	if _, err := DB.Exec(schema); err != nil {
		return err
	}

	// Safe migrations for existing tables
	_, _ = DB.Exec("ALTER TABLE users ADD COLUMN is_verified INTEGER NOT NULL DEFAULT 0")
	_, _ = DB.Exec("ALTER TABLE users ADD COLUMN verification_token TEXT NOT NULL DEFAULT ''")
	_, _ = DB.Exec("ALTER TABLE chat_history ADD COLUMN session_id TEXT NOT NULL DEFAULT 'default'")

	return nil
}

// Close gracefully closes the database.
func Close() {
	if DB != nil {
		DB.Close()
	}
}
