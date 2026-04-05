package aggregator

import (
	"net"
	"testing"

	"lanflow/internal/storage"
)

func TestRecordPacket(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("192.168.1.10", "8.8.8.8", 12345, 443, 1500)
	agg.RecordPacket("8.8.8.8", "192.168.1.10", 443, 12345, 3000)

	snapshot := agg.Snapshot()
	counter, ok := snapshot["192.168.1.10"]
	if !ok {
		t.Fatal("expected counter for 192.168.1.10")
	}
	if counter.TxBytes != 1500 {
		t.Errorf("TxBytes = %d, want 1500", counter.TxBytes)
	}
	if counter.RxBytes != 3000 {
		t.Errorf("RxBytes = %d, want 3000", counter.RxBytes)
	}
	if counter.TxPackets != 1 {
		t.Errorf("TxPackets = %d, want 1", counter.TxPackets)
	}
	if counter.RxPackets != 1 {
		t.Errorf("RxPackets = %d, want 1", counter.RxPackets)
	}
}

func TestRecordPacketIgnoresLANToLAN(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("192.168.1.10", "192.168.1.11", 12345, 80, 1000)

	snapshot := agg.Snapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected empty snapshot for LAN-to-LAN traffic, got %d entries", len(snapshot))
	}
}

func TestRecordPacketIgnoresExternalToExternal(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("8.8.8.8", "1.1.1.1", 443, 12345, 1000)

	snapshot := agg.Snapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected empty snapshot, got %d entries", len(snapshot))
	}
}

func TestFlushAndReset(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("192.168.1.10", "8.8.8.8", 12345, 443, 1500)

	records := agg.FlushAndReset()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].IP != "192.168.1.10" {
		t.Errorf("IP = %q, want %q", records[0].IP, "192.168.1.10")
	}
	if records[0].TxBytes != 1500 {
		t.Errorf("TxBytes = %d, want 1500", records[0].TxBytes)
	}

	snapshot := agg.Snapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected empty snapshot after flush, got %d entries", len(snapshot))
	}
}

func TestExcludeIPs(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet, "192.168.1.1")

	agg.RecordPacket("192.168.1.1", "8.8.8.8", 12345, 443, 1000)
	agg.RecordPacket("192.168.1.10", "8.8.8.8", 12345, 443, 2000)
	agg.RecordPacket("8.8.8.8", "192.168.1.1", 443, 12345, 500)

	snapshot := agg.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 IP (gateway excluded), got %d", len(snapshot))
	}
	if _, ok := snapshot["192.168.1.1"]; ok {
		t.Error("gateway IP should be excluded")
	}
	if snapshot["192.168.1.10"].TxBytes != 2000 {
		t.Errorf("TxBytes = %d, want 2000", snapshot["192.168.1.10"].TxBytes)
	}
}

func TestMultipleIPs(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("192.168.1.10", "8.8.8.8", 12345, 443, 1000)
	agg.RecordPacket("192.168.1.11", "8.8.8.8", 12346, 443, 2000)
	agg.RecordPacket("8.8.8.8", "192.168.1.10", 443, 12345, 500)

	snapshot := agg.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("expected 2 IPs, got %d", len(snapshot))
	}
	if snapshot["192.168.1.10"].TxBytes != 1000 {
		t.Errorf("10 TxBytes = %d, want 1000", snapshot["192.168.1.10"].TxBytes)
	}
	if snapshot["192.168.1.10"].RxBytes != 500 {
		t.Errorf("10 RxBytes = %d, want 500", snapshot["192.168.1.10"].RxBytes)
	}
	if snapshot["192.168.1.11"].TxBytes != 2000 {
		t.Errorf("11 TxBytes = %d, want 2000", snapshot["192.168.1.11"].TxBytes)
	}
}

func TestDomainTracking(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	// Record SNI first
	agg.RecordSNI("192.168.1.10", "1.2.3.4", 443, "github.com")
	agg.RecordSNI("192.168.1.10", "5.6.7.8", 443, "bilibili.com")

	// Then record packets for those flows
	agg.RecordPacket("192.168.1.10", "1.2.3.4", 50000, 443, 1000)
	agg.RecordPacket("192.168.1.10", "1.2.3.4", 50000, 443, 2000)
	agg.RecordPacket("1.2.3.4", "192.168.1.10", 443, 50000, 5000) // download from github
	agg.RecordPacket("192.168.1.10", "5.6.7.8", 50001, 443, 500)

	records := agg.FlushDomains()
	if len(records) < 2 {
		t.Fatalf("expected at least 2 domain records, got %d", len(records))
	}

	domainMap := make(map[string]storage.DomainRecord)
	for _, r := range records {
		domainMap[r.Domain] = r
	}

	gh, ok := domainMap["github.com"]
	if !ok {
		t.Fatal("expected github.com in domain records")
	}
	if gh.TxBytes != 3000 {
		t.Errorf("github.com TxBytes = %d, want 3000", gh.TxBytes)
	}
	if gh.RxBytes != 5000 {
		t.Errorf("github.com RxBytes = %d, want 5000", gh.RxBytes)
	}

	bb, ok := domainMap["bilibili.com"]
	if !ok {
		t.Fatal("expected bilibili.com in domain records")
	}
	if bb.TxBytes != 500 {
		t.Errorf("bilibili.com TxBytes = %d, want 500", bb.TxBytes)
	}
}

func TestDNSMapping(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	// Record DNS: music.163.com resolves to 1.2.3.4
	agg.RecordDNS("music.163.com", []string{"1.2.3.4"})

	// Traffic to 1.2.3.4 without SNI should be attributed to music.163.com
	agg.RecordPacket("192.168.1.10", "1.2.3.4", 50000, 443, 5000)
	agg.RecordPacket("1.2.3.4", "192.168.1.10", 443, 50000, 10000)

	records := agg.FlushDomains()
	if len(records) != 1 {
		t.Fatalf("expected 1 domain record, got %d", len(records))
	}
	if records[0].Domain != "music.163.com" {
		t.Errorf("domain = %q, want music.163.com", records[0].Domain)
	}
	if records[0].TxBytes != 5000 {
		t.Errorf("TxBytes = %d, want 5000", records[0].TxBytes)
	}
	if records[0].RxBytes != 10000 {
		t.Errorf("RxBytes = %d, want 10000", records[0].RxBytes)
	}
}
