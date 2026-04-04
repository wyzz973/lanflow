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

type Aggregator struct {
	mu       sync.Mutex
	lanNet   *net.IPNet
	counters map[string]*Counter
}

func New(lanNet *net.IPNet) *Aggregator {
	return &Aggregator{
		lanNet:   lanNet,
		counters: make(map[string]*Counter),
	}
}

func (a *Aggregator) RecordPacket(srcIP, dstIP string, size int) {
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
		c := a.getOrCreate(srcIP)
		c.TxBytes += int64(size)
		c.TxPackets++
	} else {
		c := a.getOrCreate(dstIP)
		c.RxBytes += int64(size)
		c.RxPackets++
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
