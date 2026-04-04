package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type TrafficRecord struct {
	IP        string `json:"ip"`
	Timestamp int64  `json:"timestamp"`
	TxBytes   int64  `json:"tx_bytes"`
	RxBytes   int64  `json:"rx_bytes"`
	TxPackets int64  `json:"tx_packets"`
	RxPackets int64  `json:"rx_packets"`
}

type Device struct {
	IP   string `json:"ip"`
	Name string `json:"name"`
	Note string `json:"note"`
}

type DomainRecord struct {
	IP        string `json:"ip"`
	Domain    string `json:"domain"`
	Timestamp int64  `json:"timestamp"`
	TxBytes   int64  `json:"tx_bytes"`
	RxBytes   int64  `json:"rx_bytes"`
}

type DB struct {
	db *sql.DB
}

func New(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS devices (
		ip   TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		note TEXT DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS traffic_stats (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		ip         TEXT    NOT NULL,
		timestamp  INTEGER NOT NULL,
		tx_bytes   INTEGER NOT NULL,
		rx_bytes   INTEGER NOT NULL,
		tx_packets INTEGER NOT NULL,
		rx_packets INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_ip_time ON traffic_stats(ip, timestamp);
	CREATE INDEX IF NOT EXISTS idx_time ON traffic_stats(timestamp);
	CREATE TABLE IF NOT EXISTS domain_stats (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		ip         TEXT    NOT NULL,
		domain     TEXT    NOT NULL,
		timestamp  INTEGER NOT NULL,
		tx_bytes   INTEGER NOT NULL,
		rx_bytes   INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_domain_ip_time ON domain_stats(ip, timestamp);
	CREATE INDEX IF NOT EXISTS idx_domain_time ON domain_stats(timestamp);
	`
	_, err := db.Exec(schema)
	return err
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) InsertStats(records []TrafficRecord) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO traffic_stats (ip, timestamp, tx_bytes, rx_bytes, tx_packets, rx_packets) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.Exec(r.IP, r.Timestamp, r.TxBytes, r.RxBytes, r.TxPackets, r.RxPackets); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) QueryStats(from, to int64, ip string) ([]TrafficRecord, error) {
	var rows *sql.Rows
	var err error

	if ip != "" {
		rows, err = d.db.Query(`SELECT ip, timestamp, tx_bytes, rx_bytes, tx_packets, rx_packets FROM traffic_stats WHERE timestamp >= ? AND timestamp <= ? AND ip = ? ORDER BY timestamp`, from, to, ip)
	} else {
		rows, err = d.db.Query(`SELECT ip, timestamp, tx_bytes, rx_bytes, tx_packets, rx_packets FROM traffic_stats WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp`, from, to)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TrafficRecord
	for rows.Next() {
		var r TrafficRecord
		if err := rows.Scan(&r.IP, &r.Timestamp, &r.TxBytes, &r.RxBytes, &r.TxPackets, &r.RxPackets); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (d *DB) QueryStatsSummary(from, to int64) ([]TrafficRecord, error) {
	rows, err := d.db.Query(`SELECT ip, 0 as timestamp, SUM(tx_bytes), SUM(rx_bytes), SUM(tx_packets), SUM(rx_packets) FROM traffic_stats WHERE timestamp >= ? AND timestamp <= ? GROUP BY ip ORDER BY SUM(tx_bytes) + SUM(rx_bytes) DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TrafficRecord
	for rows.Next() {
		var r TrafficRecord
		if err := rows.Scan(&r.IP, &r.Timestamp, &r.TxBytes, &r.RxBytes, &r.TxPackets, &r.RxPackets); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (d *DB) UpsertDevice(ip, name, note string) error {
	_, err := d.db.Exec(`INSERT INTO devices (ip, name, note) VALUES (?, ?, ?) ON CONFLICT(ip) DO UPDATE SET name = excluded.name, note = excluded.note`, ip, name, note)
	return err
}

func (d *DB) ListDevices() ([]Device, error) {
	rows, err := d.db.Query(`SELECT ip, name, note FROM devices ORDER BY ip`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.IP, &dev.Name, &dev.Note); err != nil {
			return nil, err
		}
		devices = append(devices, dev)
	}
	return devices, rows.Err()
}

func (d *DB) GetDeviceMap() (map[string]Device, error) {
	devices, err := d.ListDevices()
	if err != nil {
		return nil, err
	}
	m := make(map[string]Device, len(devices))
	for _, dev := range devices {
		m[dev.IP] = dev
	}
	return m, nil
}

func (d *DB) InsertDomainStats(records []DomainRecord) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO domain_stats (ip, domain, timestamp, tx_bytes, rx_bytes) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.Exec(r.IP, r.Domain, r.Timestamp, r.TxBytes, r.RxBytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) QueryDomainStats(from, to int64, ip string) ([]DomainRecord, error) {
	query := `SELECT ip, domain, 0, SUM(tx_bytes), SUM(rx_bytes) FROM domain_stats WHERE timestamp >= ? AND timestamp <= ?`
	args := []interface{}{from, to}
	if ip != "" {
		query += ` AND ip = ?`
		args = append(args, ip)
	}
	query += ` GROUP BY ip, domain ORDER BY SUM(tx_bytes) + SUM(rx_bytes) DESC`

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DomainRecord
	for rows.Next() {
		var r DomainRecord
		if err := rows.Scan(&r.IP, &r.Domain, &r.Timestamp, &r.TxBytes, &r.RxBytes); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (d *DB) Cleanup(retentionDays int) (int64, error) {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
	result, err := d.db.Exec(`DELETE FROM traffic_stats WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	d.db.Exec(`DELETE FROM domain_stats WHERE timestamp < ?`, cutoff)
	return result.RowsAffected()
}
