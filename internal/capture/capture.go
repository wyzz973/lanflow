package capture

import (
	"fmt"
	"log/slog"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"lanflow/internal/aggregator"
)

type Capture struct {
	handle *pcap.Handle
	agg    *aggregator.Aggregator
	logger *slog.Logger
}

func New(iface string, agg *aggregator.Aggregator, logger *slog.Logger) (*Capture, error) {
	handle, err := pcap.OpenLive(iface, 65535, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("open interface %s: %w", iface, err)
	}

	if err := handle.SetBPFFilter("ip"); err != nil {
		handle.Close()
		return nil, fmt.Errorf("set BPF filter: %w", err)
	}

	logger.Info("capture started", "interface", iface)

	return &Capture{
		handle: handle,
		agg:    agg,
		logger: logger,
	}, nil
}

func (c *Capture) Run() {
	source := gopacket.NewPacketSource(c.handle, c.handle.LinkType())
	source.NoCopy = true

	for packet := range source.Packets() {
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			continue
		}
		ip := ipLayer.(*layers.IPv4)
		c.agg.RecordPacket(ip.SrcIP.String(), ip.DstIP.String(), len(packet.Data()))
	}
}

func (c *Capture) Stop() {
	c.handle.Close()
	c.logger.Info("capture stopped")
}
