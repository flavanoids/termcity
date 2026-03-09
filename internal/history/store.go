package history

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"termcity/internal/data"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists incidents to a local SQLite database for history viewing.
// It is safe for concurrent use from multiple goroutines.
type Store struct {
	db *sql.DB
}

// Open creates or opens a SQLite database at dbPath and runs migrations.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening history db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS incidents (
			dedup_key     TEXT PRIMARY KEY,
			title         TEXT NOT NULL,
			address       TEXT NOT NULL,
			lat           REAL NOT NULL,
			lng           REAL NOT NULL,
			incident_type INTEGER NOT NULL,
			source        TEXT NOT NULL,
			units         TEXT NOT NULL DEFAULT '',
			reported_at   DATETIME NOT NULL,
			logged_at     DATETIME NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_incidents_reported ON incidents(reported_at);
	`)
	if err != nil {
		return fmt.Errorf("migrating history db: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// LogIncidents stores new incidents, skipping any that already exist (dedup).
// Returns the number of newly inserted rows.
func (s *Store) LogIncidents(incidents []data.Incident) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO incidents
		(dedup_key, title, address, lat, lng, incident_type, source, units, reported_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, inc := range incidents {
		key := dedupKey(inc)
		unitsStr := strings.Join(inc.Units, ",")
		result, err := stmt.Exec(
			key, inc.Title, inc.Address, inc.Lat, inc.Lng,
			int(inc.Type), inc.Source, unitsStr, inc.Time.UTC(),
		)
		if err != nil {
			return inserted, fmt.Errorf("inserting incident: %w", err)
		}
		n, _ := result.RowsAffected()
		inserted += int(n)
	}

	return inserted, tx.Commit()
}

// QueryHistory returns incidents from the last N days, ordered newest first.
func (s *Store) QueryHistory(days int) ([]data.Incident, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	rows, err := s.db.Query(`
		SELECT title, address, lat, lng, incident_type, source, units, reported_at
		FROM incidents
		WHERE reported_at >= ?
		ORDER BY reported_at DESC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("querying history: %w", err)
	}
	defer rows.Close()

	var incidents []data.Incident
	for rows.Next() {
		var inc data.Incident
		var typeInt int
		var unitsStr string
		if err := rows.Scan(&inc.Title, &inc.Address, &inc.Lat, &inc.Lng,
			&typeInt, &inc.Source, &unitsStr, &inc.Time); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		inc.Type = data.IncidentType(typeInt)
		if unitsStr != "" {
			inc.Units = strings.Split(unitsStr, ",")
		}
		inc.Time = inc.Time.Local()
		incidents = append(incidents, inc)
	}
	return incidents, rows.Err()
}

// ClearHistory removes all stored incidents.
func (s *Store) ClearHistory() error {
	_, err := s.db.Exec("DELETE FROM incidents")
	if err != nil {
		return fmt.Errorf("clearing history: %w", err)
	}
	return nil
}

// Prune removes incidents older than 7 days.
func (s *Store) Prune() error {
	cutoff := time.Now().UTC().AddDate(0, 0, -7)
	_, err := s.db.Exec("DELETE FROM incidents WHERE reported_at < ?", cutoff)
	if err != nil {
		return fmt.Errorf("pruning history: %w", err)
	}
	return nil
}

// Stats returns counts of incidents in the 1-day, 3-day, and 7-day windows.
func (s *Store) Stats() (day1, day3, day7 int, err error) {
	now := time.Now().UTC()
	for _, pair := range []struct {
		dest *int
		days int
	}{
		{&day1, 1}, {&day3, 3}, {&day7, 7},
	} {
		cutoff := now.AddDate(0, 0, -pair.days)
		if err = s.db.QueryRow(
			"SELECT COUNT(*) FROM incidents WHERE reported_at >= ?", cutoff,
		).Scan(pair.dest); err != nil {
			return
		}
	}
	return
}

// dedupKey produces a stable hash for an incident so the same real-world event
// from different poll cycles is not stored twice. Uses type + normalized
// address + title + 5-minute time bucket.
func dedupKey(inc data.Incident) string {
	addr := strings.TrimSpace(strings.ToLower(inc.Address))
	title := strings.TrimSpace(strings.ToLower(inc.Title))
	trunc := inc.Time.Truncate(5 * time.Minute).UTC().Format(time.RFC3339)
	raw := fmt.Sprintf("%d|%s|%s|%s", inc.Type, addr, title, trunc)
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash[:16])
}
