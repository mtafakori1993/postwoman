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

const portableMarker = "portable.txt"

// DataDir returns the platform-appropriate directory for Postwoman's local
// data (SQLite database, WebView cache, cookies). The POSTWOMAN_DATA_DIR
// environment variable overrides detection (used by tests and portable
// launchers). A portable.txt file next to the executable switches to
// <exe dir>/data, so a portable copy keeps everything together on a USB drive.
func DataDir() string {
	if override := strings.TrimSpace(os.Getenv("POSTWOMAN_DATA_DIR")); override != "" {
		return override
	}
	if dir, ok := portableDataDirForExecutable(); ok {
		return dir
	}
	switch runtime.GOOS {
	case "windows", "darwin":
		// Windows: %APPDATA%\Postwoman
		// macOS:   ~/Library/Application Support/Postwoman
		if dir, err := os.UserConfigDir(); err == nil && dir != "" {
			dataDir := filepath.Join(dir, "Postwoman")
			if runtime.GOOS == "windows" {
				// Preserve source-build data from the pre-1.4 default path.
				if _, statErr := os.Stat(filepath.Join(dataDir, "postwoman.db")); os.IsNotExist(statErr) {
					if home, homeErr := os.UserHomeDir(); homeErr == nil {
						legacy := filepath.Join(home, ".postwoman")
						if _, legacyErr := os.Stat(filepath.Join(legacy, "postwoman.db")); legacyErr == nil {
							return legacy
						}
					}
				}
			}
			return dataDir
		}
	}
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "Postwoman")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "Postwoman")
}

func portableDataDirForExecutable() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	return portableDataDirFrom(exe)
}

func portableDataDirFrom(exe string) (string, bool) {
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, portableMarker)); err != nil {
		return "", false
	}
	return filepath.Join(dir, "data"), true
}

// IsPortable reports whether the app is running in portable mode either
// through the marker file or an explicit data-directory override.
func IsPortable() bool {
	if strings.TrimSpace(os.Getenv("POSTWOMAN_DATA_DIR")) != "" {
		return true
	}
	_, ok := portableDataDirForExecutable()
	return ok
}

// PortableWebviewDataPath returns the WebView2 user-data directory for a
// portable build, or "" to let Wails use its normal per-user location.
func PortableWebviewDataPath() string {
	if !IsPortable() {
		return ""
	}
	return filepath.Join(DataDir(), "webview2")
}

// GetDBPath returns the platform data directory path for the database.
func GetDBPath() string {
	return filepath.Join(DataDir(), "postwoman.db")
}

// SanitizeLike escapes LIKE wildcards for search queries.
func SanitizeLike(s string) string {
	r := strings.NewReplacer("%", "\\%", "_", "\\_")
	return r.Replace(s)
}
