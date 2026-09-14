// ═══════════════════════════════════════════════════════════════════════════
// Package net provides userspace packet filtering by protocol, ports,
// and host / CIDR subnets.
// ═══════════════════════════════════════════════════
package net

import (
	"fmt"
	"net"
	"strings"
)

// PacketFilter defines matching criteria for dissected network packets.
type PacketFilter struct {
	Protocol string     // "tcp", "udp", "icmp", "arp", "ipv4", "ipv6", "ip"
	Port     uint16     // Matches either SrcPort or DstPort (0 = any)
	SrcPort  uint16     // Matches specific SrcPort (0 = any)
	DstPort  uint16     // Matches specific DstPort (0 = any)
	Host     net.IP     // Matches either SrcIP or DstIP
	CIDR     *net.IPNet // Matches either SrcIP or DstIP within subnet
}

// NewPacketFilter constructs a filter with parsed protocol, port, and host/CIDR rules.
func NewPacketFilter(protocol string, port uint16, hostOrCIDR string) (*PacketFilter, error) {
	filter := &PacketFilter{
		Protocol: strings.ToLower(strings.TrimSpace(protocol)),
		Port:     port,
	}

	if hostOrCIDR != "" {
		hostOrCIDR = strings.TrimSpace(hostOrCIDR)
		if strings.Contains(hostOrCIDR, "/") {
			_, ipNet, err := net.ParseCIDR(hostOrCIDR)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", hostOrCIDR, err)
			}
			filter.CIDR = ipNet
		} else {
			ip := net.ParseIP(hostOrCIDR)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP address %q", hostOrCIDR)
			}
			filter.Host = ip
		}
	}

	return filter, nil
}

// IsEmpty returns true if no filtering rules have been specified.
func (f *PacketFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return f.Protocol == "" && f.Port == 0 && f.SrcPort == 0 && f.DstPort == 0 && f.Host == nil && f.CIDR == nil
}

// Matches evaluates a dissected Packet against all configured criteria.
func (f *PacketFilter) Matches(pkt *Packet) bool {
	if f == nil || f.IsEmpty() {
		return true
	}
	if pkt == nil {
		return false
	}

	// 1. Protocol Filter
	if f.Protocol != "" {
		switch f.Protocol {
		case "tcp":
			if pkt.TCP == nil {
				return false
			}
		case "udp":
			if pkt.UDP == nil {
				return false
			}
		case "icmp":
			if pkt.ICMP == nil {
				return false
			}
		case "arp":
			if pkt.ARP == nil {
				return false
			}
		case "ip", "ipv4":
			if pkt.IPv4 == nil {
				return false
			}
		case "ipv6":
			if pkt.IPv6 == nil {
				return false
			}
		default:
			return false
		}
	}

	// 2. Port Filters (applies to TCP or UDP)
	if f.Port != 0 {
		matched := false
		if pkt.TCP != nil {
			if pkt.TCP.SrcPort == f.Port || pkt.TCP.DstPort == f.Port {
				matched = true
			}
		} else if pkt.UDP != nil {
			if pkt.UDP.SrcPort == f.Port || pkt.UDP.DstPort == f.Port {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}

	if f.SrcPort != 0 {
		if (pkt.TCP != nil && pkt.TCP.SrcPort != f.SrcPort) ||
			(pkt.UDP != nil && pkt.UDP.SrcPort != f.SrcPort) ||
			(pkt.TCP == nil && pkt.UDP == nil) {
			return false
		}
	}

	if f.DstPort != 0 {
		if (pkt.TCP != nil && pkt.TCP.DstPort != f.DstPort) ||
			(pkt.UDP != nil && pkt.UDP.DstPort != f.DstPort) ||
			(pkt.TCP == nil && pkt.UDP == nil) {
			return false
		}
	}

	// 3. Host Filter (Exact IP)
	if f.Host != nil {
		matched := false
		if pkt.IPv4 != nil {
			if pkt.IPv4.SrcIP.Equal(f.Host) || pkt.IPv4.DstIP.Equal(f.Host) {
				matched = true
			}
		} else if pkt.IPv6 != nil {
			if pkt.IPv6.SrcIP.Equal(f.Host) || pkt.IPv6.DstIP.Equal(f.Host) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}

	// 4. CIDR Subnet Filter
	if f.CIDR != nil {
		matched := false
		if pkt.IPv4 != nil {
			if f.CIDR.Contains(pkt.IPv4.SrcIP) || f.CIDR.Contains(pkt.IPv4.DstIP) {
				matched = true
			}
		} else if pkt.IPv6 != nil {
			if f.CIDR.Contains(pkt.IPv6.SrcIP) || f.CIDR.Contains(pkt.IPv6.DstIP) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

