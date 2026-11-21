package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := ensureDir(dir); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	dsn := fmt.Sprintf("file:%s?_fk=1&_journal_mode=WAL&_busy_timeout=10000&_synchronous=NORMAL", dbPath)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(0)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{sqlDB}

	if err := db.Migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info().Str("path", dbPath).Msg("Database initialized")
	return db, nil
}

func ensureDir(path string) error {
	if path == "" || path == "." {
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	// Verify it is actually writable.
	f, err := os.CreateTemp(path, ".postwoman-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

func (db *DB) Migrate() error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db.DB, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, err := goose.GetDBVersion(db.DB)
	if err != nil {
		return fmt.Errorf("failed to get db version: %w", err)
	}
	log.Info().Int64("version", version).Msg("Database migration completed")
	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func IsUniqueConstraintError(err error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
	}
	return false
}

func IsForeignKeyError(err error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey
	}
	return false
}

// GetDBPath returns the platform data directory path for the database.
func GetDBPath() string {
	if runtime.GOOS == "linux" {
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return filepath.Join(xdgData, "Postwoman", "postwoman.db")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "Postwoman", "postwoman.db")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".postwoman", "postwoman.db")
}

// SanitizeLike escapes LIKE wildcards for search queries.
func SanitizeLike(s string) string {
	r := strings.NewReplacer("%", "\\%", "_", "\\_")
	return r.Replace(s)
}
