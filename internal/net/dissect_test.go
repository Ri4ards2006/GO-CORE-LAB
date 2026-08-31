package net

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

// buildSyntheticTCPPacket constructs an in-memory Ethernet + IPv4 + TCP packet.
func buildSyntheticTCPPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, flags uint8, payload []byte) []byte {
	buf := new(bytes.Buffer)

	// 1. Ethernet Header (14 bytes)
	dstMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	srcMAC := []byte{0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB}
	buf.Write(dstMAC)
	buf.Write(srcMAC)
	binary.Write(buf, binary.BigEndian, EtherTypeIPv4)

	// 2. IPv4 Header (20 bytes)
	totalLen := uint16(20 + 20 + len(payload))
	buf.WriteByte(0x45) // Version 4, IHL 5 (20 bytes)
	buf.WriteByte(0x00) // DSCP/ECN
	binary.Write(buf, binary.BigEndian, totalLen)
	binary.Write(buf, binary.BigEndian, uint16(12345)) // ID
	binary.Write(buf, binary.BigEndian, uint16(0))     // Flags / Frag
	buf.WriteByte(64)                                  // TTL
	buf.WriteByte(IPProtoTCP)                          // Protocol 6
	binary.Write(buf, binary.BigEndian, uint16(0x1234)) // Checksum
	buf.Write(srcIP.To4())
	buf.Write(dstIP.To4())

	// 3. TCP Header (20 bytes)
	binary.Write(buf, binary.BigEndian, srcPort)
	binary.Write(buf, binary.BigEndian, dstPort)
	binary.Write(buf, binary.BigEndian, uint32(100000)) // Seq
	binary.Write(buf, binary.BigEndian, uint32(200000)) // Ack
	buf.WriteByte(0x50)                                 // DataOffset 5 (20 bytes)
	buf.WriteByte(flags)                                // Flags
	binary.Write(buf, binary.BigEndian, uint16(65535))  // Window
	binary.Write(buf, binary.BigEndian, uint16(0x5678))  // Checksum
	binary.Write(buf, binary.BigEndian, uint16(0))      // Urgent

	// 4. Payload
	buf.Write(payload)

	return buf.Bytes()
}

// buildSyntheticUDPPacket constructs an in-memory Ethernet + IPv4 + UDP packet.
func buildSyntheticUDPPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, payload []byte) []byte {
	buf := new(bytes.Buffer)

	// 1. Ethernet Header (14 bytes)
	buf.Write([]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})
	buf.Write([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66})
	binary.Write(buf, binary.BigEndian, EtherTypeIPv4)

	// 2. IPv4 Header (20 bytes)
	totalLen := uint16(20 + 8 + len(payload))
	buf.WriteByte(0x45)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.BigEndian, totalLen)
	binary.Write(buf, binary.BigEndian, uint16(54321))
	binary.Write(buf, binary.BigEndian, uint16(0))
	buf.WriteByte(128)
	buf.WriteByte(IPProtoUDP)
	binary.Write(buf, binary.BigEndian, uint16(0))
	buf.Write(srcIP.To4())
	buf.Write(dstIP.To4())

	// 3. UDP Header (8 bytes)
	binary.Write(buf, binary.BigEndian, srcPort)
	binary.Write(buf, binary.BigEndian, dstPort)
	binary.Write(buf, binary.BigEndian, uint16(8+len(payload)))
	binary.Write(buf, binary.BigEndian, uint16(0))

	// 4. Payload
	buf.Write(payload)

	return buf.Bytes()
}

func TestDissectTCP(t *testing.T) {
	src := net.ParseIP("192.168.1.50")
	dst := net.ParseIP("10.0.0.1")
	payload := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")

	raw := buildSyntheticTCPPacket(src, dst, 54321, 80, TCPFlagSYN|TCPFlagACK, payload)

	pkt, err := Dissect(raw)
	if err != nil {
		t.Fatalf("unexpected error dissecting TCP packet: %v", err)
	}

	if pkt.Ethernet == nil {
		t.Fatal("expected Ethernet frame, got nil")
	}
	if pkt.Ethernet.EtherType != EtherTypeIPv4 {
		t.Errorf("expected EtherType 0x0800, got 0x%04x", pkt.Ethernet.EtherType)
	}

	if pkt.IPv4 == nil {
		t.Fatal("expected IPv4 header, got nil")
	}
	if !pkt.IPv4.SrcIP.Equal(src) {
		t.Errorf("expected SrcIP %v, got %v", src, pkt.IPv4.SrcIP)
	}
	if !pkt.IPv4.DstIP.Equal(dst) {
		t.Errorf("expected DstIP %v, got %v", dst, pkt.IPv4.DstIP)
	}

	if pkt.TCP == nil {
		t.Fatal("expected TCP header, got nil")
	}
	if pkt.TCP.SrcPort != 54321 || pkt.TCP.DstPort != 80 {
		t.Errorf("expected ports 54321 -> 80, got %d -> %d", pkt.TCP.SrcPort, pkt.TCP.DstPort)
	}
	if pkt.TCP.FlagsString() != "SYN|ACK" {
		t.Errorf("expected flags 'SYN|ACK', got %q", pkt.TCP.FlagsString())
	}

	if !bytes.Equal(pkt.Payload, payload) {
		t.Errorf("expected payload %q, got %q", string(payload), string(pkt.Payload))
	}
}

func TestDissectUDP(t *testing.T) {
	src := net.ParseIP("172.16.0.2")
	dst := net.ParseIP("8.8.8.8")
	payload := []byte("DNS_QUERY_DATA")

	raw := buildSyntheticUDPPacket(src, dst, 60123, 53, payload)

	pkt, err := Dissect(raw)
	if err != nil {
		t.Fatalf("unexpected error dissecting UDP packet: %v", err)
	}

	if pkt.UDP == nil {
		t.Fatal("expected UDP header, got nil")
	}
	if pkt.UDP.SrcPort != 60123 || pkt.UDP.DstPort != 53 {
		t.Errorf("expected ports 60123 -> 53, got %d -> %d", pkt.UDP.SrcPort, pkt.UDP.DstPort)
	}
	if !bytes.Equal(pkt.Payload, payload) {
		t.Errorf("expected payload %q, got %q", string(payload), string(pkt.Payload))
	}
}

func TestDissectTruncatedPacket(t *testing.T) {
	raw := []byte{0x00, 0x01, 0x02}
	_, err := Dissect(raw)
	if err == nil {
		t.Fatal("expected error for truncated packet, got nil")
	}
}

