// ═══════════════════════════════════════════════════════════════════════════
// Package pool implements sync.Pool buffer caches for zero-allocation packet
// capture, stream slicing, and binary parsing.
// ═══════════════════════════════════════════════════
package pool

import (
	"sync"
)

const (
	// PacketBufferSize is sized to hold maximum standard MTU / jumbo frames.
	PacketBufferSize = 65535

	// SmallBufferSize is sized for stream chunks, UART frames, and headers.
	SmallBufferSize = 4096
)

var packetPool = sync.Pool{
	New: func() any {
		b := make([]byte, PacketBufferSize)
		return &b
	},
}

var smallPool = sync.Pool{
	New: func() any {
		b := make([]byte, SmallBufferSize)
		return &b
	},
}

// GetPacketBuffer acquires a 65KB byte slice from the packet pool.
func GetPacketBuffer() []byte {
	return *packetPool.Get().(*[]byte)
}

// PutPacketBuffer returns a 65KB byte slice back to the pool.
func PutPacketBuffer(b []byte) {
	if cap(b) >= PacketBufferSize {
		b = b[:PacketBufferSize]
		packetPool.Put(&b)
	}
}

// GetSmallBuffer acquires a 4KB byte slice from the small buffer pool.
func GetSmallBuffer() []byte {
	return *smallPool.Get().(*[]byte)
}

// PutSmallBuffer returns a 4KB byte slice back to the pool.
func PutSmallBuffer(b []byte) {
	if cap(b) >= SmallBufferSize {
		b = b[:SmallBufferSize]
		smallPool.Put(&b)
	}
}

