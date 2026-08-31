package export

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPcapRoundTrip(t *testing.T) {
	buf := new(bytes.Buffer)

	writer, err := NewPcapWriter(buf)
	if err != nil {
		t.Fatalf("unexpected error creating pcap writer: %v", err)
	}

	p1 := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	p2 := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00}
	t1 := time.Unix(1700000000, 500000000)
	t2 := time.Unix(1700000001, 250000000)

	if err := writer.WritePacket(p1, t1); err != nil {
		t.Fatalf("unexpected error writing packet 1: %v", err)
	}
	if err := writer.WritePacket(p2, t2); err != nil {
		t.Fatalf("unexpected error writing packet 2: %v", err)
	}

	// Read back
	reader, err := NewPcapReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("unexpected error creating pcap reader: %v", err)
	}

	if reader.Header.Network != LinkTypeEthernet {
		t.Errorf("expected link type Ethernet (%d), got %d", LinkTypeEthernet, reader.Header.Network)
	}

	rP1, rT1, err := reader.NextPacket()
	if err != nil {
		t.Fatalf("unexpected error reading packet 1: %v", err)
	}
	if !bytes.Equal(rP1, p1) {
		t.Errorf("packet 1 data mismatch: expected %v, got %v", p1, rP1)
	}
	if rT1.Unix() != t1.Unix() {
		t.Errorf("packet 1 timestamp mismatch: expected %v, got %v", t1, rT1)
	}

	rP2, _, err := reader.NextPacket()
	if err != nil {
		t.Fatalf("unexpected error reading packet 2: %v", err)
	}
	if !bytes.Equal(rP2, p2) {
		t.Errorf("packet 2 data mismatch: expected %v, got %v", p2, rP2)
	}
}

func TestGenerateSamplePcapFixture(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata")
	_ = os.MkdirAll(dir, 0755)

	filePath := filepath.Join(dir, "sample.pcap")
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("could not create fixture file: %v", err)
	}
	defer f.Close()

	pw, err := NewPcapWriter(f)
	if err != nil {
		t.Fatalf("could not init pcap writer: %v", err)
	}

	// Packet 1: TCP SYN
	p1 := makeTestPacket(net.ParseIP("192.168.1.10"), net.ParseIP("192.168.1.1"), 49152, 443, 0x02, nil)
	// Packet 2: TCP SYN/ACK
	p2 := makeTestPacket(net.ParseIP("192.168.1.1"), net.ParseIP("192.168.1.10"), 443, 49152, 0x12, nil)
	// Packet 3: TCP PSH/ACK with payload
	p3 := makeTestPacket(net.ParseIP("192.168.1.10"), net.ParseIP("192.168.1.1"), 49152, 443, 0x18, []byte("GET /telemetry HTTP/1.1\r\n\r\n"))

	now := time.Now()
	_ = pw.WritePacket(p1, now)
	_ = pw.WritePacket(p2, now.Add(2*time.Millisecond))
	_ = pw.WritePacket(p3, now.Add(5*time.Millisecond))
}

func makeTestPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, tcpFlags uint8, payload []byte) []byte {
	buf := new(bytes.Buffer)
	// Ethernet
	buf.Write([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
	buf.Write([]byte{0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB})
	binary.Write(buf, binary.BigEndian, uint16(0x0800)) // IPv4

	// IPv4
	totalLen := uint16(20 + 20 + len(payload))
	buf.WriteByte(0x45)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.BigEndian, totalLen)
	binary.Write(buf, binary.BigEndian, uint16(1001))
	binary.Write(buf, binary.BigEndian, uint16(0))
	buf.WriteByte(64)
	buf.WriteByte(6) // TCP
	binary.Write(buf, binary.BigEndian, uint16(0))
	buf.Write(srcIP.To4())
	buf.Write(dstIP.To4())

	// TCP
	binary.Write(buf, binary.BigEndian, srcPort)
	binary.Write(buf, binary.BigEndian, dstPort)
	binary.Write(buf, binary.BigEndian, uint32(1000))
	binary.Write(buf, binary.BigEndian, uint32(2000))
	buf.WriteByte(0x50) // 20 bytes
	buf.WriteByte(tcpFlags)
	binary.Write(buf, binary.BigEndian, uint16(65535))
	binary.Write(buf, binary.BigEndian, uint16(0))
	binary.Write(buf, binary.BigEndian, uint16(0))

	buf.Write(payload)
	return buf.Bytes()
}
