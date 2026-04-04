package api

import (
	"encoding/json"
	"net"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lanflow/internal/aggregator"
	"lanflow/internal/storage"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := aggregator.New(lanNet)

	srv := &Server{
		db:  db,
		agg: agg,
		hub: newHub(),
	}
	go srv.hub.run()
	return srv, func() { db.Close() }
}

func TestHandleRealtime(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	srv.agg.RecordPacket("192.168.1.10", "8.8.8.8", 1500)

	req := httptest.NewRequest("GET", "/api/realtime", nil)
	w := httptest.NewRecorder()
	srv.handleRealtime(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON decode: %v", err)
	}
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array")
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(data))
	}
}

func TestHandleDevices(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := strings.NewReader(`{"name":"GPU-01","note":"lab"}`)
	req := httptest.NewRequest("PUT", "/api/devices/192.168.1.10", body)
	w := httptest.NewRecorder()
	srv.handleDevicePut(w, req, "192.168.1.10")

	if w.Code != 200 {
		t.Fatalf("PUT status = %d, want 200", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/devices", nil)
	w = httptest.NewRecorder()
	srv.handleDevicesList(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 device, got %d", len(data))
	}
}

func TestHandleStats(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	now := time.Now()
	records := []storage.TrafficRecord{
		{IP: "192.168.1.10", Timestamp: now.Unix(), TxBytes: 1000, RxBytes: 2000, TxPackets: 10, RxPackets: 20},
	}
	srv.db.InsertStats(records)

	dateStr := now.Format("2006-01-02")
	req := httptest.NewRequest("GET", "/api/stats?range=day&date="+dateStr, nil)
	w := httptest.NewRecorder()
	srv.handleStats(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(data))
	}
}
