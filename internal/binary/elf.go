package binary

import (
    "encoding/binary"
    "fmt"
    "os"
)

type ELFHeader struct {
    Magic   [4]byte
    Class   uint8  // 1 = 32bit, 2 = 64bit
    Data    uint8  // 1 = little endian, 2 = big endian
    Machine uint16 // 0x3E = x86_64, 0x28 = ARM
    Entry   uint64 // entry point address
}

func ParseELF(path string) (*ELFHeader, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    raw := make([]byte, 64)
    if _, err := f.Read(raw); err != nil {
        return nil, err
    }

    if raw[0] != 0x7f || raw[1] != 'E' || raw[2] != 'L' || raw[3] != 'F' {
        return nil, fmt.Errorf("not an ELF file")
    }

    h := &ELFHeader{}
    copy(h.Magic[:], raw[0:4])
    h.Class = raw[4]
    h.Data = raw[5]
    h.Machine = binary.LittleEndian.Uint16(raw[18:20])
    h.Entry = binary.LittleEndian.Uint64(raw[24:32])

    return h, nil
}

func (h *ELFHeader) Print() {
    fmt.Printf("Magic:    % x\n", h.Magic)
    fmt.Printf("Class:    %s\n", map[uint8]string{1: "32-bit", 2: "64-bit"}[h.Class])
    fmt.Printf("Endian:   %s\n", map[uint8]string{1: "Little", 2: "Big"}[h.Data])
    fmt.Printf("Machine:  0x%x\n", h.Machine)
    fmt.Printf("Entry:    0x%x\n", h.Entry)
}