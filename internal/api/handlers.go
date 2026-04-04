package api

import (
	"encoding/json"
	"net/http"
	"time"

	"lanflow/internal/storage"
)

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": data})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

type realtimeEntry struct {
	IP        string `json:"ip"`
	Name      string `json:"name"`
	TxBytes   int64  `json:"tx_bytes"`
	RxBytes   int64  `json:"rx_bytes"`
	TxPackets int64  `json:"tx_packets"`
	RxPackets int64  `json:"rx_packets"`
}

func (s *Server) handleRealtime(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, s.buildRealtimeEntries())
}

// buildRealtimeEntries merges today's DB totals with current in-memory snapshot.
func (s *Server) buildRealtimeEntries() []realtimeEntry {
	// Get today's start timestamp
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()

	// Query today's totals from DB
	dbTotals, _ := s.db.QueryStatsSummary(todayStart, now.Unix()+60)
	totals := make(map[string]*realtimeEntry)
	for _, r := range dbTotals {
		if s.excludeIPs[r.IP] {
			continue
		}
		totals[r.IP] = &realtimeEntry{
			IP:        r.IP,
			TxBytes:   r.TxBytes,
			RxBytes:   r.RxBytes,
			TxPackets: r.TxPackets,
			RxPackets: r.RxPackets,
		}
	}

	// Add current in-memory snapshot (not yet flushed to DB)
	snapshot := s.agg.Snapshot()
	for ip, c := range snapshot {
		if e, ok := totals[ip]; ok {
			e.TxBytes += c.TxBytes
			e.RxBytes += c.RxBytes
			e.TxPackets += c.TxPackets
			e.RxPackets += c.RxPackets
		} else {
			totals[ip] = &realtimeEntry{
				IP:        ip,
				TxBytes:   c.TxBytes,
				RxBytes:   c.RxBytes,
				TxPackets: c.TxPackets,
				RxPackets: c.RxPackets,
			}
		}
	}

	// Resolve device names
	deviceMap, _ := s.db.GetDeviceMap()
	entries := make([]realtimeEntry, 0, len(totals))
	for _, e := range totals {
		e.Name = e.IP
		if dev, ok := deviceMap[e.IP]; ok {
			e.Name = dev.Name
		}
		entries = append(entries, *e)
	}

	return entries
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	rangeStr := r.URL.Query().Get("range")
	dateStr := r.URL.Query().Get("date")

	from, to, err := parseTimeRange(rangeStr, dateStr)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	results, err := s.db.QueryStatsSummary(from, to)
	if err != nil {
		jsonError(w, "query failed", 500)
		return
	}

	deviceMap, _ := s.db.GetDeviceMap()
	type statsEntry struct {
		IP        string `json:"ip"`
		Name      string `json:"name"`
		TxBytes   int64  `json:"tx_bytes"`
		RxBytes   int64  `json:"rx_bytes"`
		TxPackets int64  `json:"tx_packets"`
		RxPackets int64  `json:"rx_packets"`
	}

	entries := make([]statsEntry, 0, len(results))
	for _, r := range results {
		name := r.IP
		if dev, ok := deviceMap[r.IP]; ok {
			name = dev.Name
		}
		entries = append(entries, statsEntry{
			IP:        r.IP,
			Name:      name,
			TxBytes:   r.TxBytes,
			RxBytes:   r.RxBytes,
			TxPackets: r.TxPackets,
			RxPackets: r.RxPackets,
		})
	}

	jsonResponse(w, entries)
}

func (s *Server) handleStatsIP(w http.ResponseWriter, r *http.Request) {
	ip := r.PathValue("ip")
	rangeStr := r.URL.Query().Get("range")
	dateStr := r.URL.Query().Get("date")

	from, to, err := parseTimeRange(rangeStr, dateStr)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	results, err := s.db.QueryStats(from, to, ip)
	if err != nil {
		jsonError(w, "query failed", 500)
		return
	}

	jsonResponse(w, results)
}

func (s *Server) handleDevicesList(w http.ResponseWriter, r *http.Request) {
	devices, err := s.db.ListDevices()
	if err != nil {
		jsonError(w, "query failed", 500)
		return
	}
	if devices == nil {
		devices = []storage.Device{}
	}
	jsonResponse(w, devices)
}

func (s *Server) handleDevicePut(w http.ResponseWriter, r *http.Request, ip string) {
	var body struct {
		Name string `json:"name"`
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", 400)
		return
	}
	if body.Name == "" {
		jsonError(w, "name is required", 400)
		return
	}

	if err := s.db.UpsertDevice(ip, body.Name, body.Note); err != nil {
		jsonError(w, "save failed", 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

type domainEntry struct {
	IP      string `json:"ip"`
	Name    string `json:"name"`
	Domain  string `json:"domain"`
	TxBytes int64  `json:"tx_bytes"`
	RxBytes int64  `json:"rx_bytes"`
	Total   int64  `json:"total"`
}

func (s *Server) handleDomainStats(w http.ResponseWriter, r *http.Request) {
	ip := r.PathValue("ip")
	rangeStr := r.URL.Query().Get("range")
	dateStr := r.URL.Query().Get("date")

	from, to, err := parseTimeRange(rangeStr, dateStr)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	results, err := s.db.QueryDomainStats(from, to, ip)
	if err != nil {
		jsonError(w, "query failed", 500)
		return
	}

	deviceMap, _ := s.db.GetDeviceMap()
	entries := make([]domainEntry, 0, len(results))
	for _, r := range results {
		name := r.IP
		if dev, ok := deviceMap[r.IP]; ok {
			name = dev.Name
		}
		entries = append(entries, domainEntry{
			IP:      r.IP,
			Name:    name,
			Domain:  r.Domain,
			TxBytes: r.TxBytes,
			RxBytes: r.RxBytes,
			Total:   r.TxBytes + r.RxBytes,
		})
	}

	jsonResponse(w, entries)
}

func parseTimeRange(rangeStr, dateStr string) (int64, int64, error) {
	if rangeStr == "" {
		rangeStr = "day"
	}

	var base time.Time
	if dateStr != "" {
		var err error
		base, err = time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			return 0, 0, err
		}
	} else {
		base = time.Now()
	}

	base = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.Local)

	var from, to time.Time
	switch rangeStr {
	case "day":
		from = base
		to = base.Add(24 * time.Hour)
	case "week":
		weekday := int(base.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from = base.AddDate(0, 0, -(weekday - 1))
		to = from.AddDate(0, 0, 7)
	case "month":
		from = time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, time.Local)
		to = from.AddDate(0, 1, 0)
	default:
		from = base
		to = base.Add(24 * time.Hour)
	}

	return from.Unix(), to.Unix(), nil
}
