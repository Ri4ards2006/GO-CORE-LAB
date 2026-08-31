// ═══════════════════════════════════════════════════════════════════════════
// Package net implements Linux raw socket packet capture (AF_PACKET).
// ═══════════════════════════════════════════════════
package net

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"
)

// htons converts host byte order to network byte order uint16.
func htons(v uint16) uint16 {
	return (v << 8) | (v >> 8)
}

// CaptureConfig defines configuration options for network interface capture.
type CaptureConfig struct {
	Interface string
	SnapLen   int
	Promisc   bool
}

// RawSocketCapture implements low-level Linux raw socket packet sniffing.
type RawSocketCapture struct {
	Config CaptureConfig
	fd     int
}

// NewRawSocketCapture initializes a raw socket sniffer bound to an interface.
func NewRawSocketCapture(cfg CaptureConfig) *RawSocketCapture {
	if cfg.SnapLen <= 0 {
		cfg.SnapLen = 65535
	}
	return &RawSocketCapture{
		Config: cfg,
		fd:     -1,
	}
}

// Start opens the raw socket and streams captured dissected packets across a channel.
func (c *RawSocketCapture) Start(ctx context.Context) (<-chan *Packet, error) {
	// ETH_P_ALL = 0x0003 (Capture all protocols)
	const ethPAll = 0x0003

	// Open raw socket: AF_PACKET, SOCK_RAW, htons(ETH_P_ALL)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(ethPAll)))
	if err != nil {
		return nil, fmt.Errorf("open AF_PACKET raw socket (requires root / CAP_NET_RAW): %w", err)
	}
	c.fd = fd

	// If interface specified, bind socket to network interface
	if c.Config.Interface != "" {
		iface, err := net.InterfaceByName(c.Config.Interface)
		if err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("interface %q not found: %w", c.Config.Interface, err)
		}

		sll := syscall.SockaddrLinklayer{
			Protocol: htons(ethPAll),
			Ifindex:  iface.Index,
		}
		if err := syscall.Bind(fd, &sll); err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("bind raw socket to %q: %w", c.Config.Interface, err)
		}
	}

	packetChan := make(chan *Packet, 128)

	go func() {
		defer close(packetChan)
		defer syscall.Close(fd)

		buf := make([]byte, c.Config.SnapLen)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, _, err := syscall.Recvfrom(fd, buf, 0)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					continue
				}

				rawCopy := make([]byte, n)
				copy(rawCopy, buf[:n])

				pkt, err := DissectWithTimestamp(rawCopy, time.Now())
				if err == nil && pkt != nil {
					select {
					case packetChan <- pkt:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return packetChan, nil
}

// Close terminates the raw socket handle.
func (c *RawSocketCapture) Close() error {
	if c.fd >= 0 {
		err := syscall.Close(c.fd)
		c.fd = -1
		return err
	}
	return nil
}

