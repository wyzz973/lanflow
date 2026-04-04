package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := []byte(`
interface: "eth0"
lan_cidr: "192.168.1.0/24"
gateway_ip: "192.168.1.1"
db_path: "./data/lanflow.db"
retention_days: 90
listen: ":8080"
log_level: "info"
log_dir: "./logs"
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Interface != "eth0" {
		t.Errorf("Interface = %q, want %q", cfg.Interface, "eth0")
	}
	if cfg.LanCIDR != "192.168.1.0/24" {
		t.Errorf("LanCIDR = %q, want %q", cfg.LanCIDR, "192.168.1.0/24")
	}
	if cfg.GatewayIP != "192.168.1.1" {
		t.Errorf("GatewayIP = %q, want %q", cfg.GatewayIP, "192.168.1.1")
	}
	if cfg.RetentionDays != 90 {
		t.Errorf("RetentionDays = %d, want %d", cfg.RetentionDays, 90)
	}
	if cfg.Listen != ":8080" {
		t.Errorf("Listen = %q, want %q", cfg.Listen, ":8080")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	content := []byte(`
interface: "eth0"
lan_cidr: "10.0.0.0/8"
gateway_ip: "10.0.0.1"
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DBPath != "./data/lanflow.db" {
		t.Errorf("DBPath = %q, want default %q", cfg.DBPath, "./data/lanflow.db")
	}
	if cfg.RetentionDays != 90 {
		t.Errorf("RetentionDays = %d, want default %d", cfg.RetentionDays, 90)
	}
	if cfg.Listen != ":8080" {
		t.Errorf("Listen = %q, want default %q", cfg.Listen, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want default %q", cfg.LogLevel, "info")
	}
	if cfg.LogDir != "./logs" {
		t.Errorf("LogDir = %q, want default %q", cfg.LogDir, "./logs")
	}
}
