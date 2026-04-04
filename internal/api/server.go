package api

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"lanflow/internal/aggregator"
	"lanflow/internal/storage"
)

//go:embed all:static
var staticFS embed.FS

type Server struct {
	db         *storage.DB
	agg        *aggregator.Aggregator
	logger     *slog.Logger
	hub        *Hub
	excludeIPs map[string]bool
}

func NewServer(db *storage.DB, agg *aggregator.Aggregator, logger *slog.Logger, excludeIPs []string) *Server {
	exclude := make(map[string]bool, len(excludeIPs))
	for _, ip := range excludeIPs {
		exclude[ip] = true
	}
	s := &Server{
		db:         db,
		agg:        agg,
		logger:     logger,
		hub:        newHub(),
		excludeIPs: exclude,
	}
	go s.hub.run()
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/realtime", s.handleRealtime)
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("GET /api/stats/{ip}", s.handleStatsIP)
	mux.HandleFunc("GET /api/devices", s.handleDevicesList)
	mux.HandleFunc("PUT /api/devices/{ip}", func(w http.ResponseWriter, r *http.Request) {
		s.handleDevicePut(w, r, r.PathValue("ip"))
	})
	mux.HandleFunc("/ws/realtime", s.handleWebSocket)

	sub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	return mux
}
