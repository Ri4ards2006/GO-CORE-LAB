// ═══════════════════════════════════════════════════════════════════════════
// Package net implements protocol layer dissectors for Ethernet, IP, TCP, UDP, ICMP.
// ═══════════════════════════════════════════════════
package net

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Dissect decodes a raw Ethernet frame slice into a structured Packet object.
func Dissect(raw []byte) (*Packet, error) {
	return DissectWithTimestamp(raw, time.Now())
}

// DissectWithTimestamp decodes a raw packet with an explicit capture timestamp.
func DissectWithTimestamp(raw []byte, ts time.Time) (*Packet, error) {
	if len(raw) < 14 {
		return nil, fmt.Errorf("packet too short for Ethernet header (%d bytes < 14)", len(raw))
	}

	pkt := &Packet{
		Timestamp: ts,
		Raw:       raw,
		Ethernet: &EthernetFrame{
			DstMAC:    net.HardwareAddr(raw[0:6]),
			SrcMAC:    net.HardwareAddr(raw[6:12]),
			EtherType: binary.BigEndian.Uint16(raw[12:14]),
		},
	}

	payload := raw[14:]

	switch pkt.Ethernet.EtherType {
	case EtherTypeIPv4:
		if err := dissectIPv4(pkt, payload); err != nil {
			return pkt, err
		}

	case EtherTypeIPv6:
		if err := dissectIPv6(pkt, payload); err != nil {
			return pkt, err
		}

	case EtherTypeARP:
		if err := dissectARP(pkt, payload); err != nil {
			return pkt, err
		}

	default:
		pkt.Payload = payload
	}

	return pkt, nil
}

func dissectIPv4(pkt *Packet, b []byte) error {
	if len(b) < 20 {
		pkt.Payload = b
		return fmt.Errorf("truncated IPv4 header (%d bytes < 20)", len(b))
	}

	version := b[0] >> 4
	if version != 4 {
		pkt.Payload = b
		return fmt.Errorf("invalid IPv4 version %d", version)
	}

	ihl := b[0] & 0x0F
	headerLen := int(ihl) * 4
	if len(b) < headerLen || headerLen < 20 {
		pkt.Payload = b
		return fmt.Errorf("invalid IPv4 header length %d", headerLen)
	}

	totalLen := binary.BigEndian.Uint16(b[2:4])
	if int(totalLen) > len(b) {
		totalLen = uint16(len(b))
	}

	pkt.IPv4 = &IPv4Header{
		Version:        version,
		IHL:            ihl,
		DSCP:           b[1] >> 2,
		ECN:            b[1] & 0x03,
		TotalLength:    totalLen,
		Identification: binary.BigEndian.Uint16(b[4:6]),
		Flags:          b[6] >> 5,
		FragmentOffset: binary.BigEndian.Uint16(b[6:8]) & 0x1FFF,
		TTL:            b[8],
		Protocol:       b[9],
		Checksum:       binary.BigEndian.Uint16(b[10:12]),
		SrcIP:          net.IPv4(b[12], b[13], b[14], b[15]),
		DstIP:          net.IPv4(b[16], b[17], b[18], b[19]),
	}

	l4Payload := b[headerLen:totalLen]
	dissectL4(pkt, pkt.IPv4.Protocol, l4Payload)
	return nil
}

func dissectIPv6(pkt *Packet, b []byte) error {
	if len(b) < 40 {
		pkt.Payload = b
		return fmt.Errorf("truncated IPv6 header (%d bytes < 40)", len(b))
	}

	version := b[0] >> 4
	if version != 6 {
		pkt.Payload = b
		return fmt.Errorf("invalid IPv6 version %d", version)
	}

	payloadLen := binary.BigEndian.Uint16(b[4:6])
	nextHeader := b[6]

	pkt.IPv6 = &IPv6Header{
		Version:      version,
		TrafficClass: (b[0]&0x0F)<<4 | (b[1] >> 4),
		FlowLabel:    uint32(b[1]&0x0F)<<16 | uint32(b[2])<<8 | uint32(b[3]),
		PayloadLen:   payloadLen,
		NextHeader:   nextHeader,
		HopLimit:     b[7],
		SrcIP:        net.IP(b[8:24]),
		DstIP:        net.IP(b[24:40]),
	}

	l4Payload := b[40:]
	if int(payloadLen) < len(l4Payload) {
		l4Payload = l4Payload[:payloadLen]
	}

	dissectL4(pkt, nextHeader, l4Payload)
	return nil
}

func dissectARP(pkt *Packet, b []byte) error {
	if len(b) < 28 {
		pkt.Payload = b
		return fmt.Errorf("truncated ARP packet (%d bytes < 28)", len(b))
	}

	pkt.ARP = &ARPHeader{
		HardwareType: binary.BigEndian.Uint16(b[0:2]),
		ProtocolType: binary.BigEndian.Uint16(b[2:4]),
		HardwareSize: b[4],
		ProtocolSize: b[5],
		Opcode:       binary.BigEndian.Uint16(b[6:8]),
		SenderMAC:    net.HardwareAddr(b[8:14]),
		SenderIP:     net.IPv4(b[14], b[15], b[16], b[17]),
		TargetMAC:    net.HardwareAddr(b[18:24]),
		TargetIP:     net.IPv4(b[24], b[25], b[26], b[27]),
	}

	pkt.Payload = b[28:]
	return nil
}

func dissectL4(pkt *Packet, proto uint8, b []byte) {
	switch proto {
	case IPProtoTCP:
		if len(b) < 20 {
			pkt.Payload = b
			return
		}
		dataOffset := (b[12] >> 4) * 4
		if int(dataOffset) > len(b) || dataOffset < 20 {
			dataOffset = 20
		}

		pkt.TCP = &TCPHeader{
			SrcPort:    binary.BigEndian.Uint16(b[0:2]),
			DstPort:    binary.BigEndian.Uint16(b[2:4]),
			SeqNum:     binary.BigEndian.Uint32(b[4:8]),
			AckNum:     binary.BigEndian.Uint32(b[8:12]),
			DataOffset: dataOffset / 4,
			Flags:      b[13],
			WindowSize: binary.BigEndian.Uint16(b[14:16]),
			Checksum:   binary.BigEndian.Uint16(b[16:18]),
			UrgentPtr:  binary.BigEndian.Uint16(b[18:20]),
		}

		if int(dataOffset) <= len(b) {
			pkt.Payload = b[dataOffset:]
		}

	case IPProtoUDP:
		if len(b) < 8 {
			pkt.Payload = b
			return
		}
		pkt.UDP = &UDPHeader{
			SrcPort:  binary.BigEndian.Uint16(b[0:2]),
			DstPort:  binary.BigEndian.Uint16(b[2:4]),
			Length:   binary.BigEndian.Uint16(b[4:6]),
			Checksum: binary.BigEndian.Uint16(b[6:8]),
		}
		if len(b) >= 8 {
			pkt.Payload = b[8:]
		}

	case IPProtoICMP:
		if len(b) < 4 {
			pkt.Payload = b
			return
		}
		pkt.ICMP = &ICMPHeader{
			Type:     b[0],
			Code:     b[1],
			Checksum: binary.BigEndian.Uint16(b[2:4]),
		}
		if len(b) >= 4 {
			pkt.Payload = b[4:]
		}

	default:
		pkt.Payload = b
	}
}

// PrintHexDump formats payload bytes in 16-byte canonical hex/ASCII format.
func FormatHexDump(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	result := ""
	for i := 0; i < len(data); i += 16 {
		chunk := data[i:]
		if len(chunk) > 16 {
			chunk = chunk[:16]
		}

		hexPart := ""
		for j := 0; j < 16; j++ {
			if j < len(chunk) {
				hexPart += fmt.Sprintf("%02x ", chunk[j])
			} else {
				hexPart += "   "
			}
			if j == 7 {
				hexPart += " "
			}
		}

		asciiPart := ""
		for _, b := range chunk {
			if b >= 32 && b <= 126 {
				asciiPart += string(b)
			} else {
				asciiPart += "."
			}
		}

		result += fmt.Sprintf("    %04x  %-49s |%s|\n", i, hexPart, asciiPart)
	}
	return result
}

