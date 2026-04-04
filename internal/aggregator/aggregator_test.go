package aggregator

import (
	"net"
	"testing"
)

func TestRecordPacket(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("192.168.1.10", "8.8.8.8", 1500)
	agg.RecordPacket("8.8.8.8", "192.168.1.10", 3000)

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

	agg.RecordPacket("192.168.1.10", "192.168.1.11", 1000)

	snapshot := agg.Snapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected empty snapshot for LAN-to-LAN traffic, got %d entries", len(snapshot))
	}
}

func TestRecordPacketIgnoresExternalToExternal(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("8.8.8.8", "1.1.1.1", 1000)

	snapshot := agg.Snapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected empty snapshot, got %d entries", len(snapshot))
	}
}

func TestFlushAndReset(t *testing.T) {
	_, lanNet, _ := net.ParseCIDR("192.168.1.0/24")
	agg := New(lanNet)

	agg.RecordPacket("192.168.1.10", "8.8.8.8", 1500)

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

	agg.RecordPacket("192.168.1.1", "8.8.8.8", 1000)
	agg.RecordPacket("192.168.1.10", "8.8.8.8", 2000)
	agg.RecordPacket("8.8.8.8", "192.168.1.1", 500)

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

	agg.RecordPacket("192.168.1.10", "8.8.8.8", 1000)
	agg.RecordPacket("192.168.1.11", "8.8.8.8", 2000)
	agg.RecordPacket("8.8.8.8", "192.168.1.10", 500)

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
