package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewDB(t *testing.T) {
	dir := t.TempDir()
	db, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer db.Close()
}

func TestInsertAndQueryStats(t *testing.T) {
	dir := t.TempDir()
	db, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	records := []TrafficRecord{
		{IP: "192.168.1.10", Timestamp: now, TxBytes: 1000, RxBytes: 2000, TxPackets: 10, RxPackets: 20},
		{IP: "192.168.1.11", Timestamp: now, TxBytes: 500, RxBytes: 800, TxPackets: 5, RxPackets: 8},
	}

	if err := db.InsertStats(records); err != nil {
		t.Fatalf("InsertStats() error: %v", err)
	}

	results, err := db.QueryStats(now-60, now+60, "")
	if err != nil {
		t.Fatalf("QueryStats() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestQueryStatsByIP(t *testing.T) {
	dir := t.TempDir()
	db, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	records := []TrafficRecord{
		{IP: "192.168.1.10", Timestamp: now, TxBytes: 1000, RxBytes: 2000, TxPackets: 10, RxPackets: 20},
		{IP: "192.168.1.11", Timestamp: now, TxBytes: 500, RxBytes: 800, TxPackets: 5, RxPackets: 8},
	}
	_ = db.InsertStats(records)

	results, err := db.QueryStats(now-60, now+60, "192.168.1.10")
	if err != nil {
		t.Fatalf("QueryStats() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TxBytes != 1000 {
		t.Errorf("TxBytes = %d, want 1000", results[0].TxBytes)
	}
}

func TestDevices(t *testing.T) {
	dir := t.TempDir()
	db, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer db.Close()

	if err := db.UpsertDevice("192.168.1.10", "GPU-01", "lab room 301"); err != nil {
		t.Fatalf("UpsertDevice() error: %v", err)
	}

	devices, err := db.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices() error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].Name != "GPU-01" {
		t.Errorf("Name = %q, want %q", devices[0].Name, "GPU-01")
	}

	if err := db.UpsertDevice("192.168.1.10", "GPU-01-Updated", "new note"); err != nil {
		t.Fatalf("UpsertDevice() update error: %v", err)
	}
	devices, _ = db.ListDevices()
	if devices[0].Name != "GPU-01-Updated" {
		t.Errorf("Name after update = %q, want %q", devices[0].Name, "GPU-01-Updated")
	}
}

func TestCleanup(t *testing.T) {
	dir := t.TempDir()
	db, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer db.Close()

	old := time.Now().Add(-100 * 24 * time.Hour).Unix()
	recent := time.Now().Unix()
	records := []TrafficRecord{
		{IP: "192.168.1.10", Timestamp: old, TxBytes: 100, RxBytes: 200, TxPackets: 1, RxPackets: 2},
		{IP: "192.168.1.10", Timestamp: recent, TxBytes: 300, RxBytes: 400, TxPackets: 3, RxPackets: 4},
	}
	_ = db.InsertStats(records)

	deleted, err := db.Cleanup(90)
	if err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	results, _ := db.QueryStats(0, recent+60, "")
	if len(results) != 1 {
		t.Errorf("expected 1 remaining record, got %d", len(results))
	}
}
