package binary

import (
    "encoding/binary"
    "fmt"
    "os"
)

// ELF Magic bytes — erste 4 bytes jeder ELF Datei
var elfMagic = [4]byte{0x7f, 'E', 'L', 'F'}

// Konstanten statt magic numbers
const (
    Class32    = 1
    Class64    = 2
    DataLE     = 1 // Little Endian
    DataBE     = 2 // Big Endian
)

// Maschinen-Architektur Map — direkt lesbar statt hex
var machineNames = map[uint16]string{
    0x03: "x86",
    0x3E: "x86_64",
    0x28: "ARM",
    0xB7: "ARM64",
    0xF3: "RISC-V",
}

// ELFHeader — die ersten 64 bytes jeder ELF Datei
type ELFHeader struct {
    Class   uint8  // 32 oder 64 bit
    Data    uint8  // endianness
    Machine uint16 // CPU Architektur
    Entry   uint64 // wo fängt der code an
    ShOff   uint64 // offset zur section header table
    ShNum   uint16 // anzahl sections
    ShStrIdx uint16 // index des section name strings
}

// Section — ein Segment der ELF Datei
type Section struct {
    Name   string
    Type   uint32
    Offset uint64
    Size   uint64
}

// ELFFile — das komplette geparste File
type ELFFile struct {
    Header   ELFHeader
    Sections []Section
    Path     string
}

// ParseELF — liest eine ELF Datei und gibt ein ELFFile zurück
func ParseELF(path string) (*ELFFile, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("open: %w", err)
    }
    defer f.Close()

    // erste 64 bytes lesen
    raw := make([]byte, 64)
    if _, err := f.Read(raw); err != nil {
        return nil, fmt.Errorf("read header: %w", err)
    }

    // Magic bytes checken
    if raw[0] != elfMagic[0] || raw[1] != elfMagic[1] ||
        raw[2] != elfMagic[2] || raw[3] != elfMagic[3] {
        return nil, fmt.Errorf("not an ELF file")
    }

    elf := &ELFFile{Path: path}
    elf.Header.Class = raw[4]
    elf.Header.Data = raw[5]

    // Little Endian oder Big Endian bestimmen
    var bo binary.ByteOrder
    if elf.Header.Data == DataLE {
        bo = binary.LittleEndian
    } else {
        bo = binary.BigEndian
    }

    elf.Header.Machine = bo.Uint16(raw[18:20])

    // 32bit vs 64bit — Entry Point und Section Offset sind unterschiedlich groß
    if elf.Header.Class == Class64 {
        elf.Header.Entry = bo.Uint64(raw[24:32])
        elf.Header.ShOff = bo.Uint64(raw[40:48])
        elf.Header.ShNum = bo.Uint16(raw[60:62])
        elf.Header.ShStrIdx = bo.Uint16(raw[62:64])
    } else {
        elf.Header.Entry = uint64(bo.Uint32(raw[24:28]))
        elf.Header.ShOff = uint64(bo.Uint32(raw[32:36]))
        elf.Header.ShNum = bo.Uint16(raw[48:50])
        elf.Header.ShStrIdx = bo.Uint16(raw[50:52])
    }

    return elf, nil
}

// Print — gibt den Header lesbar aus
func (e *ELFFile) Print() {
    arch, ok := machineNames[e.Header.Machine]
    if !ok {
        arch = fmt.Sprintf("0x%x", e.Header.Machine)
    }

    bits := map[uint8]string{Class32: "32-bit", Class64: "64-bit"}
    endian := map[uint8]string{DataLE: "Little Endian", DataBE: "Big Endian"}

    fmt.Printf("File:     %s\n", e.Path)
    fmt.Printf("Arch:     %s (%s)\n", arch, bits[e.Header.Class])
    fmt.Printf("Endian:   %s\n", endian[e.Header.Data])
    fmt.Printf("Entry:    0x%x\n", e.Header.Entry)
    fmt.Printf("Sections: %d\n", e.Header.ShNum)
}