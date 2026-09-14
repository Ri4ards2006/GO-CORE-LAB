//go:build linux

// ═══════════════════════════════════════════════════════════════════════════
// Package hw provides Linux hardware bus monitors for passive SPI and I2C buses.
// ═══════════════════════════════════════════════════
package hw

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Linux SPI ioctl constants (from <linux/spi/spidev.h>)
const (
	spiIocMagic         = 'k'
	spiIocWrMode        = 0x40016b01 // _IOW(SPI_IOC_MAGIC, 1, __u8)
	spiIocWrBitsPerWord = 0x40016b03 // _IOW(SPI_IOC_MAGIC, 3, __u8)
	spiIocWrMaxSpeedHz  = 0x40046b04 // _IOW(SPI_IOC_MAGIC, 4, __u32)
	spiIocRdMaxSpeedHz  = 0x80046b04 // _IOR(SPI_IOC_MAGIC, 4, __u32)
)

// Linux I2C ioctl constants (from <linux/i2c-dev.h>)
const (
	i2cSlave = 0x0703 // Use this slave address
)

// SPIBus wraps a Linux spidev interface (e.g. /dev/spidev0.0).
type SPIBus struct {
	Device   string
	SpeedHz  uint32
	Mode     uint8
	file     *os.File
}

// NewSPIBus opens a Linux spidev character device and configures speed and mode.
func NewSPIBus(device string, speedHz uint32, mode uint8) (*SPIBus, error) {
	if speedHz == 0 {
		speedHz = 1000000 // Default 1 MHz
	}

	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open SPI device %q: %w", device, err)
	}

	// 1. Set SPI Mode
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(spiIocWrMode), uintptr(unsafe.Pointer(&mode))); errno != 0 {
		f.Close()
		return nil, fmt.Errorf("ioctl SPI_IOC_WR_MODE on %q: %w", device, errno)
	}

	// 2. Set Max Speed (Hz)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(spiIocWrMaxSpeedHz), uintptr(unsafe.Pointer(&speedHz))); errno != 0 {
		f.Close()
		return nil, fmt.Errorf("ioctl SPI_IOC_WR_MAX_SPEED_HZ on %q: %w", device, errno)
	}

	return &SPIBus{
		Device:  device,
		SpeedHz: speedHz,
		Mode:    mode,
		file:    f,
	}, nil
}

// Read reads raw streaming bytes from the SPI bus.
func (s *SPIBus) Read(buf []byte) (int, error) {
	if s.file == nil {
		return 0, fmt.Errorf("SPI device is closed")
	}
	return s.file.Read(buf)
}

// Close releases the SPI file descriptor.
func (s *SPIBus) Close() error {
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// I2CBus wraps a Linux i2c-dev interface (e.g. /dev/i2c-1).
type I2CBus struct {
	Device    string
	SlaveAddr uint16
	file      *os.File
}

// NewI2CBus opens a Linux I2C device and sets the target slave address.
func NewI2CBus(device string, slaveAddr uint16) (*I2CBus, error) {
	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open I2C device %q: %w", device, err)
	}

	// Set I2C Slave Address via ioctl
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(i2cSlave), uintptr(slaveAddr)); errno != 0 {
		f.Close()
		return nil, fmt.Errorf("ioctl I2C_SLAVE (0x%02x) on %q: %w", slaveAddr, device, errno)
	}

	return &I2CBus{
		Device:    device,
		SlaveAddr: slaveAddr,
		file:      f,
	}, nil
}

// ReadBytes reads a block of N bytes from the selected I2C slave.
func (i *I2CBus) ReadBytes(length int) ([]byte, error) {
	if i.file == nil {
		return nil, fmt.Errorf("I2C device is closed")
	}
	buf := make([]byte, length)
	n, err := i.file.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read I2C: %w", err)
	}
	return buf[:n], nil
}

// ReadBlock writes a register address and reads back length bytes.
func (i *I2CBus) ReadBlock(reg byte, length int) ([]byte, error) {
	if i.file == nil {
		return nil, fmt.Errorf("I2C device is closed")
	}
	// Write register pointer
	if _, err := i.file.Write([]byte{reg}); err != nil {
		return nil, fmt.Errorf("write register pointer 0x%02x: %w", reg, err)
	}
	return i.ReadBytes(length)
}

// WriteBlock writes a register address followed by data bytes.
func (i *I2CBus) WriteBlock(reg byte, data []byte) error {
	if i.file == nil {
		return fmt.Errorf("I2C device is closed")
	}
	payload := append([]byte{reg}, data...)
	_, err := i.file.Write(payload)
	return err
}

// Close closes the I2C device file descriptor.
func (i *I2CBus) Close() error {
	if i.file != nil {
		err := i.file.Close()
		i.file = nil
		return err
	}
	return nil
}

