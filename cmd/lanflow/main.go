package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lanflow/internal/aggregator"
	"lanflow/internal/api"
	"lanflow/internal/capture"
	"lanflow/internal/config"
	"lanflow/internal/logger"
	"lanflow/internal/storage"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	logLevel := flag.String("log-level", "", "override log level (debug/info/warn/error)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	log, err := logger.Setup(cfg.LogLevel, cfg.LogDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}

	log.Info("lanflow starting",
		"interface", cfg.Interface,
		"lan_cidr", cfg.LanCIDR,
		"listen", cfg.Listen,
	)

	_, lanNet, err := net.ParseCIDR(cfg.LanCIDR)
	if err != nil {
		log.Error("invalid lan_cidr", "error", err)
		os.Exit(1)
	}

	db, err := storage.New(cfg.DBPath)
	if err != nil {
		log.Error("database error", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Exclude gateway and local IPs from traffic stats
	excludeIPs := []string{cfg.GatewayIP}
	if iface, err := net.InterfaceByName(cfg.Interface); err == nil {
		if addrs, err := iface.Addrs(); err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					excludeIPs = append(excludeIPs, ipnet.IP.String())
				}
			}
		}
	}
	log.Info("excluding IPs from stats", "ips", excludeIPs)
	agg := aggregator.New(lanNet, excludeIPs...)

	cap, err := capture.New(cfg.Interface, agg, log)
	if err != nil {
		log.Error("capture error", "error", err)
		os.Exit(1)
	}

	go cap.Run()

	srv := api.NewServer(db, agg, log)

	go flushLoop(agg, db, log, cfg.RetentionDays)
	go broadcastLoop(srv)

	httpServer := &http.Server{
		Addr:    cfg.Listen,
		Handler: srv.Handler(),
	}

	go func() {
		log.Info("web server started", "listen", cfg.Listen)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Info("shutting down", "signal", sig.String())

	cap.Stop()

	records := agg.FlushAndReset()
	if len(records) > 0 {
		if err := db.InsertStats(records); err != nil {
			log.Error("final flush failed", "error", err)
		}
		log.Info("final flush completed", "records", len(records))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)

	log.Info("lanflow stopped")
}

func flushLoop(agg *aggregator.Aggregator, db *storage.DB, log *slog.Logger, retentionDays int) {
	flushTicker := time.NewTicker(60 * time.Second)
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer flushTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			records := agg.FlushAndReset()
			if len(records) == 0 {
				continue
			}
			if err := db.InsertStats(records); err != nil {
				log.Error("flush to db failed", "error", err)
				continue
			}
			total := int64(0)
			for _, r := range records {
				total += r.TxBytes + r.RxBytes
			}
			log.Info("flush completed", "ips", len(records), "total_bytes", total)

		case <-cleanupTicker.C:
			deleted, err := db.Cleanup(retentionDays)
			if err != nil {
				log.Error("cleanup failed", "error", err)
			} else if deleted > 0 {
				log.Info("cleanup completed", "deleted_rows", deleted)
			}
		}
	}
}

func broadcastLoop(srv *api.Server) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		srv.BroadcastRealtime()
	}
}
