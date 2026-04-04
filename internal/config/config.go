package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Interface     string `yaml:"interface"`
	LanCIDR       string `yaml:"lan_cidr"`
	GatewayIP     string `yaml:"gateway_ip"`
	DBPath        string `yaml:"db_path"`
	RetentionDays int    `yaml:"retention_days"`
	Listen        string `yaml:"listen"`
	LogLevel      string `yaml:"log_level"`
	LogDir        string `yaml:"log_dir"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		DBPath:        "./data/lanflow.db",
		RetentionDays: 90,
		Listen:        ":8080",
		LogLevel:      "info",
		LogDir:        "./logs",
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Interface == "" {
		return nil, fmt.Errorf("interface is required")
	}
	if cfg.LanCIDR == "" {
		return nil, fmt.Errorf("lan_cidr is required")
	}
	if cfg.GatewayIP == "" {
		return nil, fmt.Errorf("gateway_ip is required")
	}

	return cfg, nil
}
