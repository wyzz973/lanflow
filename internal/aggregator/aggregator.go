package aggregator

import (
	"net"
	"sync"
	"time"

	"lanflow/internal/storage"
)

type Counter struct {
	TxBytes   int64
	RxBytes   int64
	TxPackets int64
	RxPackets int64
}

type DomainCounter struct {
	TxBytes int64
	RxBytes int64
}

// FlowKey identifies a connection by IP pair + port
type FlowKey struct {
	LocalIP    string
	RemoteIP   string
	RemotePort uint16
}

type Aggregator struct {
	mu         sync.Mutex
	lanNet     *net.IPNet
	excludeIPs map[string]bool
	counters   map[string]*Counter
	// Domain tracking
	domainCounters map[string]map[string]*DomainCounter // ip -> domain -> counter
	flowToDomain   map[FlowKey]string                   // connection -> domain mapping
}

func New(lanNet *net.IPNet, excludeIPs ...string) *Aggregator {
	exclude := make(map[string]bool, len(excludeIPs))
	for _, ip := range excludeIPs {
		exclude[ip] = true
	}
	return &Aggregator{
		lanNet:         lanNet,
		excludeIPs:     exclude,
		counters:       make(map[string]*Counter),
		domainCounters: make(map[string]map[string]*DomainCounter),
		flowToDomain:   make(map[FlowKey]string),
	}
}

func (a *Aggregator) RecordSNI(localIP, remoteIP string, remotePort uint16, domain string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := FlowKey{LocalIP: localIP, RemoteIP: remoteIP, RemotePort: remotePort}
	a.flowToDomain[key] = domain
}

func (a *Aggregator) RecordPacket(srcIP, dstIP string, srcPort, dstPort uint16, size int) {
	src := net.ParseIP(srcIP)
	dst := net.ParseIP(dstIP)
	if src == nil || dst == nil {
		return
	}

	srcInLAN := a.lanNet.Contains(src)
	dstInLAN := a.lanNet.Contains(dst)

	if srcInLAN == dstInLAN {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if srcInLAN {
		if a.excludeIPs[srcIP] {
			return
		}
		c := a.getOrCreate(srcIP)
		c.TxBytes += int64(size)
		c.TxPackets++
		// Track domain
		key := FlowKey{LocalIP: srcIP, RemoteIP: dstIP, RemotePort: dstPort}
		if domain, ok := a.flowToDomain[key]; ok {
			a.getOrCreateDomain(srcIP, domain).TxBytes += int64(size)
		}
	} else {
		if a.excludeIPs[dstIP] {
			return
		}
		c := a.getOrCreate(dstIP)
		c.RxBytes += int64(size)
		c.RxPackets++
		// Track domain
		key := FlowKey{LocalIP: dstIP, RemoteIP: srcIP, RemotePort: srcPort}
		if domain, ok := a.flowToDomain[key]; ok {
			a.getOrCreateDomain(dstIP, domain).RxBytes += int64(size)
		}
	}
}

func (a *Aggregator) getOrCreate(ip string) *Counter {
	c, ok := a.counters[ip]
	if !ok {
		c = &Counter{}
		a.counters[ip] = c
	}
	return c
}

func (a *Aggregator) getOrCreateDomain(ip, domain string) *DomainCounter {
	if _, ok := a.domainCounters[ip]; !ok {
		a.domainCounters[ip] = make(map[string]*DomainCounter)
	}
	dc, ok := a.domainCounters[ip][domain]
	if !ok {
		dc = &DomainCounter{}
		a.domainCounters[ip][domain] = dc
	}
	return dc
}

func (a *Aggregator) Snapshot() map[string]Counter {
	a.mu.Lock()
	defer a.mu.Unlock()

	snap := make(map[string]Counter, len(a.counters))
	for ip, c := range a.counters {
		snap[ip] = *c
	}
	return snap
}

func (a *Aggregator) FlushAndReset() []storage.TrafficRecord {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now().Truncate(time.Minute).Unix()
	records := make([]storage.TrafficRecord, 0, len(a.counters))

	for ip, c := range a.counters {
		records = append(records, storage.TrafficRecord{
			IP:        ip,
			Timestamp: now,
			TxBytes:   c.TxBytes,
			RxBytes:   c.RxBytes,
			TxPackets: c.TxPackets,
			RxPackets: c.RxPackets,
		})
	}

	a.counters = make(map[string]*Counter)
	return records
}

func (a *Aggregator) FlushDomains() []storage.DomainRecord {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now().Truncate(time.Minute).Unix()
	var records []storage.DomainRecord

	for ip, domains := range a.domainCounters {
		for domain, c := range domains {
			if c.TxBytes > 0 || c.RxBytes > 0 {
				records = append(records, storage.DomainRecord{
					IP:        ip,
					Domain:    domain,
					Timestamp: now,
					TxBytes:   c.TxBytes,
					RxBytes:   c.RxBytes,
				})
			}
		}
	}

	a.domainCounters = make(map[string]map[string]*DomainCounter)
	// Clean up old flow mappings to avoid memory leak
	if len(a.flowToDomain) > 10000 {
		a.flowToDomain = make(map[FlowKey]string)
	}

	return records
}
