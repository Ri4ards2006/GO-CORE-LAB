//go:build !linux

// ═══════════════════════════════════════════════════════════════════════════
// Package hw provides cross-platform fallback stubs for hardware bus monitors.
// ═══════════════════════════════════════════════════
package hw

import "fmt"

// SPIBus fallback for non-Linux platforms.
type SPIBus struct {
	Device  string
	SpeedHz uint32
	Mode    uint8
}

// NewSPIBus returns an error on non-Linux platforms.
func NewSPIBus(device string, speedHz uint32, mode uint8) (*SPIBus, error) {
	return nil, fmt.Errorf("SPI bus monitoring is only supported on Linux")
}

func (s *SPIBus) Read(buf []byte) (int, error) {
	return 0, fmt.Errorf("unsupported on this OS")
}

func (s *SPIBus) Close() error {
	return nil
}

// I2CBus fallback for non-Linux platforms.
type I2CBus struct {
	Device    string
	SlaveAddr uint16
}

// NewI2CBus returns an error on non-Linux platforms.
func NewI2CBus(device string, slaveAddr uint16) (*I2CBus, error) {
	return nil, fmt.Errorf("I2C bus monitoring is only supported on Linux")
}

func (i *I2CBus) ReadBytes(length int) ([]byte, error) {
	return nil, fmt.Errorf("unsupported on this OS")
}

func (i *I2CBus) ReadBlock(reg byte, length int) ([]byte, error) {
	return nil, fmt.Errorf("unsupported on this OS")
}

func (i *I2CBus) WriteBlock(reg byte, data []byte) error {
	return fmt.Errorf("unsupported on this OS")
}

func (i *I2CBus) Close() error {
	return nil
}

