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

		var srcPort, dstPort uint16
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp := tcpLayer.(*layers.TCP)
			srcPort = uint16(tcp.SrcPort)
			dstPort = uint16(tcp.DstPort)

			// Try to extract SNI from TLS ClientHello
			if dstPort == 443 && len(tcp.Payload) > 0 {
				if domain := extractSNI(tcp.Payload); domain != "" {
					c.agg.RecordSNI(ip.SrcIP.String(), ip.DstIP.String(), dstPort, domain)
				}
			}

			// Try to extract Host from HTTP
			if dstPort == 80 && len(tcp.Payload) > 0 {
				if domain := extractHTTPHost(tcp.Payload); domain != "" {
					c.agg.RecordSNI(ip.SrcIP.String(), ip.DstIP.String(), dstPort, domain)
				}
			}
		} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
			udp := udpLayer.(*layers.UDP)
			srcPort = uint16(udp.SrcPort)
			dstPort = uint16(udp.DstPort)

			// Parse DNS responses for IP-to-domain mapping
			if srcPort == 53 {
				if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
					dns := dnsLayer.(*layers.DNS)
					if dns.QR && len(dns.Questions) > 0 {
						qname := string(dns.Questions[0].Name)
						var ips []string
						for _, ans := range dns.Answers {
							if ans.Type == layers.DNSTypeA {
								ips = append(ips, ans.IP.String())
							}
						}
						if len(ips) > 0 && qname != "" {
							c.agg.RecordDNS(qname, ips)
						}
					}
				}
			}
		}

		c.agg.RecordPacket(ip.SrcIP.String(), ip.DstIP.String(), srcPort, dstPort, len(packet.Data()))
	}
}

func (c *Capture) Stop() {
	c.handle.Close()
	c.logger.Info("capture stopped")
}

func extractSNI(payload []byte) string {
	// Minimum TLS record: 5 bytes header + 1 byte content
	if len(payload) < 6 {
		return ""
	}
	// Check TLS record: ContentType=Handshake(22), Version
	if payload[0] != 22 {
		return ""
	}

	// Record length
	recordLen := int(payload[3])<<8 | int(payload[4])
	if len(payload) < 5+recordLen {
		return ""
	}
	data := payload[5 : 5+recordLen]

	// Handshake type: ClientHello = 1
	if len(data) < 1 || data[0] != 1 {
		return ""
	}

	// Skip handshake header (4 bytes: type + length)
	if len(data) < 4 {
		return ""
	}
	data = data[4:]

	// Skip client version (2) + random (32)
	if len(data) < 34 {
		return ""
	}
	data = data[34:]

	// Skip session ID
	if len(data) < 1 {
		return ""
	}
	sessIDLen := int(data[0])
	data = data[1:]
	if len(data) < sessIDLen {
		return ""
	}
	data = data[sessIDLen:]

	// Skip cipher suites
	if len(data) < 2 {
		return ""
	}
	csLen := int(data[0])<<8 | int(data[1])
	data = data[2:]
	if len(data) < csLen {
		return ""
	}
	data = data[csLen:]

	// Skip compression methods
	if len(data) < 1 {
		return ""
	}
	compLen := int(data[0])
	data = data[1:]
	if len(data) < compLen {
		return ""
	}
	data = data[compLen:]

	// Extensions
	if len(data) < 2 {
		return ""
	}
	extLen := int(data[0])<<8 | int(data[1])
	data = data[2:]
	if len(data) < extLen {
		return ""
	}
	data = data[:extLen]

	// Parse extensions looking for SNI (type 0)
	for len(data) >= 4 {
		extType := int(data[0])<<8 | int(data[1])
		extDataLen := int(data[2])<<8 | int(data[3])
		data = data[4:]
		if len(data) < extDataLen {
			return ""
		}

		if extType == 0 { // SNI extension
			sniData := data[:extDataLen]
			if len(sniData) < 2 {
				return ""
			}
			// SNI list length
			sniData = sniData[2:]
			if len(sniData) < 3 {
				return ""
			}
			// Type (0 = hostname) + length
			nameType := sniData[0]
			nameLen := int(sniData[1])<<8 | int(sniData[2])
			sniData = sniData[3:]
			if nameType == 0 && len(sniData) >= nameLen {
				return string(sniData[:nameLen])
			}
			return ""
		}

		data = data[extDataLen:]
	}

	return ""
}

func extractHTTPHost(payload []byte) string {
	// Quick check for HTTP request
	if len(payload) < 16 {
		return ""
	}
	// Check if it starts with an HTTP method
	s := string(payload)
	if len(s) > 0 && s[0] != 'G' && s[0] != 'P' && s[0] != 'H' && s[0] != 'D' && s[0] != 'O' && s[0] != 'C' {
		return ""
	}

	// Find Host header
	for i := 0; i < len(s)-6; i++ {
		if (s[i] == 'H' || s[i] == 'h') && (s[i+1] == 'o' || s[i+1] == 'O') && (s[i+2] == 's' || s[i+2] == 'S') && (s[i+3] == 't' || s[i+3] == 'T') && s[i+4] == ':' {
			// Found "Host:"
			j := i + 5
			// Skip whitespace
			for j < len(s) && s[j] == ' ' {
				j++
			}
			// Read until \r or \n
			k := j
			for k < len(s) && s[k] != '\r' && s[k] != '\n' {
				k++
			}
			host := s[j:k]
			// Remove port if present
			if idx := len(host) - 1; idx > 0 {
				for ci := 0; ci < len(host); ci++ {
					if host[ci] == ':' {
						host = host[:ci]
						break
					}
				}
			}
			return host
		}
	}
	return ""
}
