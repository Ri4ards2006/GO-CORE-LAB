package binary

import (

	"encoding/binary"
	"fmt"
	"os"
)

var elfMagic = []byte{0x7f, 'E', 'L', 'F'}


// Konstante für die ELF-Header-Felder
const (
	class32 = uint8(1)
	class64 = uint8(2)

	dataLittle = uint8(1)
	dataBig    = uint8(2) 
// little endian = 1, big endian = 2
)

// Maschinen-Architekturen Mapping lesbar statt Hex-Werten

var machineNames = map[uint16]string{
	0x3E: "x86-64",
	0x28: "ARM",
	0xB7: "AArch64",
	0x00: "No specific instruction set",
	0x02: "SPARC",
	0x03: "x86",
	0x08: "MIPS",
	0x14: "PowerPC",
	0x16: "S390",
	0x28: "ARM",
	0x2A: "SuperH",
	0x32: "IA-64",
	0x3E: "x86-64",
	0xB7: "AArch64",
	0xF3: "RISC-V",
	0xF7: "Berkeley Packet Filter",
	0x101: "WDC 65C816",
	0xFF00: "LoOS (Operating system-specific)",
	0xFFFF: "HiOS (Operating system-specific)",
}

type ELFHeader struct {

	Class  uint8  // 1 = 32bit, 2 = 64bit
	Data   uint8  // 1 = little endian, 2 = big endian
	Machine uint16 // 0x3E = x86_64, 0x28 = ARM
	Entry   uint64 // entry point address
	Shoff   uint64 // section header offset
	Shnum   uint16 // number of section headers
	ShStrndx uint16 // section header string table index
	
}