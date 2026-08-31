// ═══════════════════════════════════════════════════════════════════════════
// Package export implements PCAP binary format writer and reader.
// ═══════════════════════════════════════════════════
package export

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// PCAP Magic numbers
const (
	PcapMagicMicroseconds uint32 = 0xa1b2c3d4
	PcapMagicNanoseconds  uint32 = 0xa1b23c4d
	LinkTypeEthernet      uint32 = 1
)

// PcapGlobalHeader represents the standard 24-byte libpcap file header.
type PcapGlobalHeader struct {
	MagicNumber  uint32 // 0xa1b2c3d4 (native byte order)
	VersionMajor uint16 // 2
	VersionMinor uint16 // 4
	ThisZone     int32  // GMT to local correction (usually 0)
	SigFigs      uint32 // Accuracy of timestamps (usually 0)
	SnapLen      uint32 // Max length of captured packets in bytes
	Network      uint32 // Data link type (1 = LINKTYPE_ETHERNET)
}

// PcapPacketHeader represents the 16-byte record header preceding each packet.
type PcapPacketHeader struct {
	TsSec   uint32 // Timestamp seconds
	TsUsec  uint32 // Timestamp microseconds (or nanoseconds)
	InclLen uint32 // Number of octets of packet saved in file
	OrigLen uint32 // Actual length of packet when transmitted
}

// PcapWriter serializes captured network frames into standard .pcap format.
type PcapWriter struct {
	w io.Writer
}

// NewPcapWriter creates and writes the global header to the output stream.
func NewPcapWriter(w io.Writer) (*PcapWriter, error) {
	hdr := PcapGlobalHeader{
		MagicNumber:  PcapMagicMicroseconds,
		VersionMajor: 2,
		VersionMinor: 4,
		ThisZone:     0,
		SigFigs:      0,
		SnapLen:      65535,
		Network:      LinkTypeEthernet,
	}

	if err := binary.Write(w, binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("write pcap global header: %w", err)
	}

	return &PcapWriter{w: w}, nil
}

// WritePacket appends a packet record to the PCAP file.
func (pw *PcapWriter) WritePacket(pkt []byte, ts time.Time) error {
	if ts.IsZero() {
		ts = time.Now()
	}

	pktHdr := PcapPacketHeader{
		TsSec:   uint32(ts.Unix()),
		TsUsec:  uint32(ts.Nanosecond() / 1000), // Convert to microseconds
		InclLen: uint32(len(pkt)),
		OrigLen: uint32(len(pkt)),
	}

	if err := binary.Write(pw.w, binary.LittleEndian, &pktHdr); err != nil {
		return fmt.Errorf("write pcap packet header: %w", err)
	}

	if _, err := pw.w.Write(pkt); err != nil {
		return fmt.Errorf("write pcap packet data: %w", err)
	}

	return nil
}

// PcapReader parses standard .pcap files and iterates over individual packets.
type PcapReader struct {
	r            io.Reader
	Header       PcapGlobalHeader
	isBigEndian  bool
	isNanosecond bool
}

// NewPcapReader reads the global header from a pcap stream.
func NewPcapReader(r io.Reader) (*PcapReader, error) {
	var magic uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("read pcap magic: %w", err)
	}

	var bo binary.ByteOrder = binary.LittleEndian
	isBE := false
	isNano := false

	switch magic {
	case PcapMagicMicroseconds:
		bo = binary.LittleEndian
	case PcapMagicNanoseconds:
		bo = binary.LittleEndian
		isNano = true
	case 0xd4c3b2a1: // Big Endian micro
		bo = binary.BigEndian
		isBE = true
	case 0x4d3cb2a1: // Big Endian nano
		bo = binary.BigEndian
		isBE = true
		isNano = true
	default:
		return nil, fmt.Errorf("unrecognized pcap magic number: 0x%08x", magic)
	}

	hdr := PcapGlobalHeader{MagicNumber: magic}
	if err := binary.Read(r, bo, &hdr.VersionMajor); err != nil {
		return nil, err
	}
	if err := binary.Read(r, bo, &hdr.VersionMinor); err != nil {
		return nil, err
	}
	if err := binary.Read(r, bo, &hdr.ThisZone); err != nil {
		return nil, err
	}
	if err := binary.Read(r, bo, &hdr.SigFigs); err != nil {
		return nil, err
	}
	if err := binary.Read(r, bo, &hdr.SnapLen); err != nil {
		return nil, err
	}
	if err := binary.Read(r, bo, &hdr.Network); err != nil {
		return nil, err
	}

	return &PcapReader{
		r:            r,
		Header:       hdr,
		isBigEndian:  isBE,
		isNanosecond: isNano,
	}, nil
}

// OpenPcap opens a file from disk and initializes a PcapReader.
func OpenPcap(path string) (*PcapReader, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open pcap file: %w", err)
	}
	pr, err := NewPcapReader(f)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return pr, f, nil
}

// NextPacket reads the next packet from the pcap stream.
func (pr *PcapReader) NextPacket() ([]byte, time.Time, error) {
	var bo binary.ByteOrder = binary.LittleEndian
	if pr.isBigEndian {
		bo = binary.BigEndian
	}

	var pktHdr PcapPacketHeader
	if err := binary.Read(pr.r, bo, &pktHdr); err != nil {
		return nil, time.Time{}, err
	}

	var ts time.Time
	if pr.isNanosecond {
		ts = time.Unix(int64(pktHdr.TsSec), int64(pktHdr.TsUsec))
	} else {
		ts = time.Unix(int64(pktHdr.TsSec), int64(pktHdr.TsUsec)*1000)
	}

	data := make([]byte, pktHdr.InclLen)
	if _, err := io.ReadFull(pr.r, data); err != nil {
		return nil, ts, fmt.Errorf("read packet data: %w", err)
	}

	return data, ts, nil
}
