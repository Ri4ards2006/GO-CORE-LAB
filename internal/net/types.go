// ═══════════════════════════════════════════════════════════════════════════
// Package net implements raw packet dissection, layer parsing, and
// network telemetry capture for GO-CORE-LAB.
// ═══════════════════════════════════════════════════
package net

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// EtherType constants (Layer 2)
const (
	EtherTypeIPv4 uint16 = 0x0800
	EtherTypeARP  uint16 = 0x0806
	EtherTypeIPv6 uint16 = 0x86DD
)

// IP Protocol Numbers (Layer 3)
const (
	IPProtoICMP uint8 = 1
	IPProtoTCP  uint8 = 6
	IPProtoUDP  uint8 = 17
)

// TCP Flag bitmasks (Layer 4)
const (
	TCPFlagFIN uint8 = 0x01
	TCPFlagSYN uint8 = 0x02
	TCPFlagRST uint8 = 0x04
	TCPFlagPSH uint8 = 0x08
	TCPFlagACK uint8 = 0x10
	TCPFlagURG uint8 = 0x20
	TCPFlagECE uint8 = 0x40
	TCPFlagCWR uint8 = 0x80
)

// EthernetFrame represents a Layer 2 Ethernet II frame.
type EthernetFrame struct {
	DstMAC    net.HardwareAddr
	SrcMAC    net.HardwareAddr
	EtherType uint16
}

// IPv4Header represents a Layer 3 IPv4 packet header.
type IPv4Header struct {
	Version        uint8
	IHL            uint8 // Internet Header Length in 32-bit words
	DSCP           uint8
	ECN            uint8
	TotalLength    uint16
	Identification uint16
	Flags          uint8
	FragmentOffset uint16
	TTL            uint8
	Protocol       uint8
	Checksum       uint16
	SrcIP          net.IP
	DstIP          net.IP
}

// IPv6Header represents a Layer 3 IPv6 packet header.
type IPv6Header struct {
	Version      uint8
	TrafficClass uint8
	FlowLabel    uint32
	PayloadLen   uint16
	NextHeader   uint8
	HopLimit     uint8
	SrcIP        net.IP
	DstIP        net.IP
}

// ARPHeader represents an Address Resolution Protocol payload.
type ARPHeader struct {
	HardwareType uint16
	ProtocolType uint16
	HardwareSize uint8
	ProtocolSize uint8
	Opcode       uint16 // 1 = Request, 2 = Reply
	SenderMAC    net.HardwareAddr
	SenderIP     net.IP
	TargetMAC    net.HardwareAddr
	TargetIP     net.IP
}

// TCPHeader represents a Layer 4 TCP segment header.
type TCPHeader struct {
	SrcPort    uint16
	DstPort    uint16
	SeqNum     uint32
	AckNum     uint32
	DataOffset uint8 // Header length in 32-bit words
	Flags      uint8
	WindowSize uint16
	Checksum   uint16
	UrgentPtr  uint16
}

// FlagsString returns a human-readable list of active TCP flags (e.g. "[SYN, ACK]").
func (t TCPHeader) FlagsString() string {
	var f []string
	if t.Flags&TCPFlagSYN != 0 {
		f = append(f, "SYN")
	}
	if t.Flags&TCPFlagACK != 0 {
		f = append(f, "ACK")
	}
	if t.Flags&TCPFlagFIN != 0 {
		f = append(f, "FIN")
	}
	if t.Flags&TCPFlagRST != 0 {
		f = append(f, "RST")
	}
	if t.Flags&TCPFlagPSH != 0 {
		f = append(f, "PSH")
	}
	if t.Flags&TCPFlagURG != 0 {
		f = append(f, "URG")
	}
	if len(f) == 0 {
		return "none"
	}
	return strings.Join(f, "|")
}

// UDPHeader represents a Layer 4 UDP datagram header.
type UDPHeader struct {
	SrcPort  uint16
	DstPort  uint16
	Length   uint16
	Checksum uint16
}

// ICMPHeader represents an Internet Control Message Protocol header.
type ICMPHeader struct {
	Type     uint8
	Code     uint8
	Checksum uint16
}

// Packet represents a fully dissected network frame across all protocol layers.
type Packet struct {
	Timestamp time.Time
	Raw       []byte
	Ethernet  *EthernetFrame
	IPv4      *IPv4Header
	IPv6      *IPv6Header
	ARP       *ARPHeader
	TCP       *TCPHeader
	UDP       *UDPHeader
	ICMP      *ICMPHeader
	Payload   []byte
}

// Summary returns a single-line summary of the packet flow (e.g. "192.168.1.5:443 -> 192.168.1.100:52134 [TCP] SYN|ACK").
func (p *Packet) Summary() string {
	ts := p.Timestamp.Format("15:04:05.000000")
	if p.Timestamp.IsZero() {
		ts = time.Now().Format("15:04:05.000000")
	}

	if p.ARP != nil {
		op := "Request"
		if p.ARP.Opcode == 2 {
			op = "Reply"
		}
		return fmt.Sprintf("[%s] ARP %s: Who has %s? Tell %s (%s)",
			ts, op, p.ARP.TargetIP, p.ARP.SenderIP, p.ARP.SenderMAC)
	}

	src := "unknown"
	dst := "unknown"
	proto := "UNKNOWN"

	if p.IPv4 != nil {
		src = p.IPv4.SrcIP.String()
		dst = p.IPv4.DstIP.String()
	} else if p.IPv6 != nil {
		src = p.IPv6.SrcIP.String()
		dst = p.IPv6.DstIP.String()
	}

	if p.TCP != nil {
		return fmt.Sprintf("[%s] TCP  %s:%d -> %s:%d [%s] (seq=%d, ack=%d, len=%d)",
			ts, src, p.TCP.SrcPort, dst, p.TCP.DstPort, p.TCP.FlagsString(), p.TCP.SeqNum, p.TCP.AckNum, len(p.Payload))
	}

	if p.UDP != nil {
		return fmt.Sprintf("[%s] UDP  %s:%d -> %s:%d (len=%d)",
			ts, src, p.UDP.SrcPort, dst, p.UDP.DstPort, len(p.Payload))
	}

	if p.ICMP != nil {
		return fmt.Sprintf("[%s] ICMP %s -> %s (type=%d, code=%d)",
			ts, src, dst, p.ICMP.Type, p.ICMP.Code)
	}

	if p.IPv4 != nil {
		proto = fmt.Sprintf("IP proto=%d", p.IPv4.Protocol)
	}
	return fmt.Sprintf("[%s] %s %s -> %s (len=%d)", ts, proto, src, dst, len(p.Raw))
}

