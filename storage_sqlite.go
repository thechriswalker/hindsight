package hindsight

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

// schema version, will run the migrations up until that point
const currentSchemaVersion = 1

// current schema, table is different, as we will migrate data on
// startup
var schemaMigrations = []string{
	// 0 - the initial DB, moves us to
	`CREATE TABLE hindsight_events (
		id INTEGER PRIMARY KEY,
		time INTEGER NOT NULL,
		unique_visitor TEXT NOT NULL,
		req_host TEXT NOT NULL,
		req_path TEXT NOT NULL,
		req_method TEXT NOT NULL,
		res_status INTEGER NOT NULL,
		res_duration_ms INTEGER NOT NULL,
		res_bytes_written INTEGER NOT NULL,
		browser_kind TEXT NOT NULL,
		browser_name TEXT NOT NULL,
		browser_version TEXT NOT NULL,
		os_name TEXT NOT NULL,
		os_version TEXT NOT NULL,
		location_country_code TEXT NOT NULL,
		location_time_zone TEXT NOT NULL
	);`, // lets go the naive route and just list the fields
}

func NewSQLiteStorage(dsn string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not open db for storage: %s", err)
	}
	// initial connection stuff
	// turn on Write-Ahead-Log
	db.Exec(`PRAGMA journal_mode=WAL;`)
	// possibly create meta db
	db.Exec(`CREATE TABLE IF NOT EXISTS hindsight_schema (version INTEGER NOT NULL, time NUMERIC NOT NULL);`)
	// check schema version
	var schemaVersion int
	err = db.QueryRow(`SELECT MAX(version) FROM hindsight_schema;`).Scan(&schemaVersion)
	if err != nil {
		return nil, fmt.Errorf("could not read initial schema version: %w", err)
	}
	// while the schema version is less than target run a migration
	for ; schemaVersion < currentSchemaVersion; schemaVersion++ {
		_, err = db.Exec(schemaMigrations[schemaVersion])
		if err != nil {
			return nil, fmt.Errorf("failed to migrate from schema version %d: %w", schemaVersion, err)
		}
		db.Exec(`INSERT INTO hindsight_schema (version, time) VALUES (?, ?);`, schemaVersion+1, time.Now().Unix())
		if err != nil {
			return nil, fmt.Errorf("failed to update schema version table for version %d: %w", schemaVersion+1, err)
		}
	}

	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) Store(evts []*Event) error {
	// bulk insert is tricky, but SQLite is quick with single inserts.
	for i, ev := range evts {
		_, err := s.db.Exec(`INSERT INTO hindsight_events (
			time, unique_visitor,
			req_host, req_path, req_method,
			res_status, res_duration_ms, res_bytes_written,
			browser_kind, browser_name, browser_version,
			os_name, os_version,
			location_country_code, location_time_zone)
		VALUES (
			?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,?,
			?,?
		);`,
			ev.Time.Unix(), ev.Key,
			ev.Host, ev.Path, ev.Method,
			ev.StatusCode, ev.Duration, ev.BytesWritten,
			ev.Device, ev.Browser.Name, ev.Browser.Version,
			ev.OS.Name, ev.OS.Version,
			ev.CountryCode, ev.TimeZone,
		)
		if err != nil {
			return fmt.Errorf("failed to store event %d/%d: %w", i+1, len(evts), err)
		}
	}
	return nil
}

func (s *SQLiteStorage) Fetch(from, until time.Time, filter *Filter) ([]*Event, error) {
	query := `
		SELECT time, unique_visitor,
			req_host, req_path, req_method,
			res_status, res_duration_ms, res_bytes_written,
			browser_kind, browser_name, browser_version,
			os_name, os_version,
			location_country_code, location_time_zone
		FROM hindsight_events WHERE time BETWEEN ? AND ?
	`
	args := []interface{}{from.Unix(), until.Unix()}
	if filter != nil {
		if filter.HostList != nil && len(filter.HostList) > 0 {
			query += "AND (" + strings.Repeat("req_host = ? OR ", len(filter.HostList)-1) + "req_host = ?)"
			for _, host := range filter.HostList {
				args = append(args, host)
			}
		}
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying for events: %w", err)
	}
	events := []*Event{}
	for rows.Next() {
		next := &Event{}
		var unix int64
		err := rows.Scan(
			&unix,
			&(next.Host), &(next.Path), &(next.Method),
			&(next.StatusCode), &(next.Duration), &(next.BytesWritten),
			&(next.Device), &(next.Browser.Name), &(next.Browser.Version),
			&(next.OS.Name), &(next.OS.Version),
			&(next.CountryCode), &(next.TimeZone),
		)
		if err != nil {
			return events, fmt.Errorf("error scanning row: %w", err)
		}
		events = append(events, next)
	}
	if err = rows.Err(); err != nil {
		return events, fmt.Errorf("error while scanning rows: %w", err)
	}

	return events, nil
}
