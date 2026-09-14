package net

import (
	"net"
	"testing"
)

func TestPacketFilterProtocols(t *testing.T) {
	tcpPkt := &Packet{
		IPv4: &IPv4Header{
			SrcIP:    net.ParseIP("192.168.1.10"),
			DstIP:    net.ParseIP("192.168.1.1"),
			Protocol: IPProtoTCP,
		},
		TCP: &TCPHeader{
			SrcPort: 54321,
			DstPort: 443,
		},
	}

	udpPkt := &Packet{
		IPv4: &IPv4Header{
			SrcIP:    net.ParseIP("192.168.1.10"),
			DstIP:    net.ParseIP("8.8.8.8"),
			Protocol: IPProtoUDP,
		},
		UDP: &UDPHeader{
			SrcPort: 12345,
			DstPort: 53,
		},
	}

	icmpPkt := &Packet{
		IPv4: &IPv4Header{
			SrcIP:    net.ParseIP("192.168.1.10"),
			DstIP:    net.ParseIP("1.1.1.1"),
			Protocol: IPProtoICMP,
		},
		ICMP: &ICMPHeader{
			Type: 8,
			Code: 0,
		},
	}

	// Filter: TCP only
	tcpFilter, err := NewPacketFilter("tcp", 0, "")
	if err != nil {
		t.Fatalf("unexpected filter error: %v", err)
	}
	if !tcpFilter.Matches(tcpPkt) {
		t.Error("expected TCP packet to match tcp filter")
	}
	if tcpFilter.Matches(udpPkt) {
		t.Error("expected UDP packet NOT to match tcp filter")
	}
	if tcpFilter.Matches(icmpPkt) {
		t.Error("expected ICMP packet NOT to match tcp filter")
	}

	// Filter: UDP only
	udpFilter, _ := NewPacketFilter("udp", 0, "")
	if !udpFilter.Matches(udpPkt) {
		t.Error("expected UDP packet to match udp filter")
	}
	if udpFilter.Matches(tcpPkt) {
		t.Error("expected TCP packet NOT to match udp filter")
	}
}

func TestPacketFilterPorts(t *testing.T) {
	pkt := &Packet{
		IPv4: &IPv4Header{
			SrcIP: net.ParseIP("10.0.0.1"),
			DstIP: net.ParseIP("10.0.0.2"),
		},
		TCP: &TCPHeader{
			SrcPort: 49152,
			DstPort: 80,
		},
	}

	// Match DstPort 80
	f80, _ := NewPacketFilter("", 80, "")
	if !f80.Matches(pkt) {
		t.Error("expected match for port 80")
	}

	// Match SrcPort 49152
	f49152, _ := NewPacketFilter("", 49152, "")
	if !f49152.Matches(pkt) {
		t.Error("expected match for port 49152")
	}

	// Non-matching port 443
	f443, _ := NewPacketFilter("", 443, "")
	if f443.Matches(pkt) {
		t.Error("expected port 443 NOT to match")
	}
}

func TestPacketFilterHostAndCIDR(t *testing.T) {
	pkt := &Packet{
		IPv4: &IPv4Header{
			SrcIP: net.ParseIP("192.168.1.100"),
			DstIP: net.ParseIP("172.16.0.1"),
		},
		TCP: &TCPHeader{
			SrcPort: 8080,
			DstPort: 80,
		},
	}

	// Exact Host match (SrcIP)
	fHost, err := NewPacketFilter("", 0, "192.168.1.100")
	if err != nil {
		t.Fatalf("unexpected filter error: %v", err)
	}
	if !fHost.Matches(pkt) {
		t.Error("expected match on exact host 192.168.1.100")
	}

	// Exact Host match (DstIP)
	fDst, _ := NewPacketFilter("", 0, "172.16.0.1")
	if !fDst.Matches(pkt) {
		t.Error("expected match on exact dst host 172.16.0.1")
	}

	// CIDR Subnet match
	fCIDR, err := NewPacketFilter("", 0, "192.168.1.0/24")
	if err != nil {
		t.Fatalf("unexpected CIDR error: %v", err)
	}
	if !fCIDR.Matches(pkt) {
		t.Error("expected match on CIDR 192.168.1.0/24")
	}

	// Non-matching CIDR
	fCIDRBad, _ := NewPacketFilter("", 0, "10.0.0.0/8")
	if fCIDRBad.Matches(pkt) {
		t.Error("expected 10.0.0.0/8 NOT to match")
	}

	// Invalid CIDR string
	_, err = NewPacketFilter("", 0, "invalid-ip-string")
	if err == nil {
		t.Error("expected error for invalid IP string, got nil")
	}
}

