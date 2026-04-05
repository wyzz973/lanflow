package api

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"lanflow/internal/aggregator"
	"lanflow/internal/classifier"
	"lanflow/internal/storage"
)

//go:embed all:static
var staticFS embed.FS

type Server struct {
	db         *storage.DB
	agg        *aggregator.Aggregator
	logger     *slog.Logger
	hub        *Hub
	excludeIPs map[string]bool // IPs to completely hide (e.g. gateway)
	selfIPs    map[string]bool // Server's own IPs (need traffic correction due to NAT)
	classifier *classifier.Classifier
}

func NewServer(db *storage.DB, agg *aggregator.Aggregator, logger *slog.Logger, excludeIPs []string, selfIPs []string, cls *classifier.Classifier) *Server {
	exclude := make(map[string]bool, len(excludeIPs))
	for _, ip := range excludeIPs {
		exclude[ip] = true
	}
	self := make(map[string]bool, len(selfIPs))
	for _, ip := range selfIPs {
		self[ip] = true
	}
	s := &Server{
		db:         db,
		agg:        agg,
		logger:     logger,
		hub:        newHub(),
		excludeIPs: exclude,
		selfIPs:    self,
		classifier: cls,
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
	mux.HandleFunc("GET /api/domains/{ip}", s.handleDomainStats)
	mux.HandleFunc("GET /api/domains", s.handleDomainStats)
	mux.HandleFunc("/ws/realtime", s.handleWebSocket)

	sub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	return mux
}
