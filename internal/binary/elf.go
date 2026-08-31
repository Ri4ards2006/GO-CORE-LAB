// ═══════════════════════════════════════════════════════════════════════════
// Package binary provides low-level decoding and analysis for executable
// binary formats (ELF32, ELF64, PE, Mach-O).
// ═══════════════════════════════════════════════════════════════════════════
package binary

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// Magic bytes identifying an ELF binary: 0x7F 'E' 'L' 'F'
var elfMagic = [4]byte{0x7f, 'E', 'L', 'F'}

// ELF Header identification constants
const (
	Class32 = 1 // 32-bit architecture
	Class64 = 2 // 64-bit architecture
	DataLE  = 1 // 2's complement, Little Endian
	DataBE  = 2 // 2's complement, Big Endian
)

// Section Types (sh_type)
const (
	SHT_NULL          uint32 = 0  // Inactive section header
	SHT_PROGBITS      uint32 = 1  // Program code or initialized data
	SHT_SYMTAB        uint32 = 2  // Symbol table (static)
	SHT_STRTAB        uint32 = 3  // String table
	SHT_RELA          uint32 = 4  // Relocation entries with explicit addends
	SHT_HASH          uint32 = 5  // Symbol hash table
	SHT_DYNAMIC       uint32 = 6  // Information for dynamic linking
	SHT_NOTE          uint32 = 7  // Note information
	SHT_NOBITS        uint32 = 8  // Uninitialized data (.bss) - no file space
	SHT_REL           uint32 = 9  // Relocation entries without explicit addends
	SHT_SHLIB         uint32 = 10 // Reserved
	SHT_DYNSYM        uint32 = 11 // Dynamic symbol table
	SHT_INIT_ARRAY    uint32 = 14 // Array of constructors
	SHT_FINI_ARRAY    uint32 = 15 // Array of destructors
	SHT_PREINIT_ARRAY uint32 = 16 // Array of pre-constructors
	SHT_GROUP         uint32 = 17 // Section group
	SHT_SYMTAB_SHNDX  uint32 = 18 // Extended section indices
)

// Section Flags (sh_flags)
const (
	SHF_WRITE            uint64 = 0x1        // Writable during process execution
	SHF_ALLOC            uint64 = 0x2        // Occupies memory during execution
	SHF_EXECINSTR        uint64 = 0x4        // Executable machine instructions
	SHF_MERGE            uint64 = 0x10       // Might be merged
	SHF_STRINGS          uint64 = 0x20       // Contains null-terminated strings
	SHF_INFO_LINK        uint64 = 0x40       // 'sh_info' contains SHT index
	SHF_LINK_ORDER       uint64 = 0x80       // Preserve order after combining
	SHF_OS_NONCONFORMING uint64 = 0x100      // Non-standard OS-specific handling
	SHF_GROUP            uint64 = 0x200      // Section is a member of a group
	SHF_TLS              uint64 = 0x400      // Thread-local storage
	SHF_COMPRESSED       uint64 = 0x800      // Section with compressed data
	SHF_MASKOS           uint64 = 0x0ff00000 // OS-specific flags
	SHF_MASKPROC         uint64 = 0xf0000000 // Processor-specific flags
)

// SectionTypeNames maps sh_type constants to human-readable names
var sectionTypeNames = map[uint32]string{
	SHT_NULL:          "NULL",
	SHT_PROGBITS:      "PROGBITS",
	SHT_SYMTAB:        "SYMTAB",
	SHT_STRTAB:        "STRTAB",
	SHT_RELA:          "RELA",
	SHT_HASH:          "HASH",
	SHT_DYNAMIC:       "DYNAMIC",
	SHT_NOTE:          "NOTE",
	SHT_NOBITS:        "NOBITS",
	SHT_REL:           "REL",
	SHT_SHLIB:         "SHLIB",
	SHT_DYNSYM:        "DYNSYM",
	SHT_INIT_ARRAY:    "INIT_ARRAY",
	SHT_FINI_ARRAY:    "FINI_ARRAY",
	SHT_PREINIT_ARRAY: "PREINIT_ARRAY",
	SHT_GROUP:         "GROUP",
	SHT_SYMTAB_SHNDX:  "SYMTAB_SHNDX",
}

// ELFHeader represents the normalized ELF File Header (32-bit and 64-bit).
type ELFHeader struct {
	Class    uint8  // 1 = 32-bit, 2 = 64-bit
	Data     uint8  // 1 = Little Endian, 2 = Big Endian
	Machine  uint16 // Target instruction set architecture
	Entry    uint64 // Virtual entry point address
	ShOff    uint64 // Section header table file offset
	ShNum    uint16 // Number of section header entries
	ShStrIdx uint16 // Section header index of the string table (.shstrtab)
}

// Section models an ELF section header and its resolved metadata.
type Section struct {
	Index     int    // Index in the section header table
	Name      string // Resolved section name (e.g. ".text", ".rodata")
	NameOff   uint32 // Offset into .shstrtab (sh_name)
	Type      uint32 // Section type (sh_type)
	Flags     uint64 // Section attributes (sh_flags)
	Addr      uint64 // Virtual address in memory (sh_addr)
	Offset    uint64 // Offset in binary file (sh_offset)
	Size      uint64 // Size of section in bytes (sh_size)
	Link      uint32 // Link to another section index (sh_link)
	Info      uint32 // Extra information depending on type (sh_info)
	AddrAlign uint64 // Address alignment constraint (sh_addralign)
	EntSize   uint64 // Entry size for sections holding fixed-size tables (sh_entsize)
}

// ELFFile represents a parsed ELF binary with its header and section table.
type ELFFile struct {
	Header   ELFHeader
	Sections []Section
	Path     string
}

// ParseELF opens a file, verifies its ELF identity, and parses both
// the file header and the section header table with string resolution.
func ParseELF(path string) (*ELFFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	return ParseELFReader(f, path)
}

// ParseELFReader parses an ELF binary from any io.ReadSeeker source.
func ParseELFReader(r io.ReadSeeker, path string) (*ELFFile, error) {
	raw := make([]byte, 64)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// Validate 4-byte Magic Number
	if raw[0] != elfMagic[0] || raw[1] != elfMagic[1] ||
		raw[2] != elfMagic[2] || raw[3] != elfMagic[3] {
		return nil, fmt.Errorf("not an ELF file: invalid magic % x", raw[0:4])
	}

	elf := &ELFFile{Path: path}
	elf.Header.Class = raw[4]
	elf.Header.Data = raw[5]

	if elf.Header.Class != Class32 && elf.Header.Class != Class64 {
		return nil, fmt.Errorf("unsupported ELF class: %d", elf.Header.Class)
	}

	var bo binary.ByteOrder
	if elf.Header.Data == DataLE {
		bo = binary.LittleEndian
	} else if elf.Header.Data == DataBE {
		bo = binary.BigEndian
	} else {
		return nil, fmt.Errorf("unsupported ELF data encoding: %d", elf.Header.Data)
	}

	elf.Header.Machine = bo.Uint16(raw[18:20])

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

	// Parse Section Header Table
	sections, err := parseSections(r, &elf.Header, bo)
	if err != nil {
		return nil, fmt.Errorf("parse sections: %w", err)
	}
	elf.Sections = sections

	return elf, nil
}

// parseSections traverses the Section Header Table at e_shoff and resolves section names.
func parseSections(r io.ReadSeeker, h *ELFHeader, bo binary.ByteOrder) ([]Section, error) {
	if h.ShOff == 0 || h.ShNum == 0 {
		return nil, nil
	}

	if _, err := r.Seek(int64(h.ShOff), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to section header table (offset 0x%x): %w", h.ShOff, err)
	}

	sections := make([]Section, h.ShNum)

	if h.Class == Class64 {
		const shdrSize64 = 64
		buf := make([]byte, int(h.ShNum)*shdrSize64)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read 64-bit section headers: %w", err)
		}

		for i := 0; i < int(h.ShNum); i++ {
			off := i * shdrSize64
			sections[i] = Section{
				Index:     i,
				NameOff:   bo.Uint32(buf[off : off+4]),
				Type:      bo.Uint32(buf[off+4 : off+8]),
				Flags:     bo.Uint64(buf[off+8 : off+16]),
				Addr:      bo.Uint64(buf[off+16 : off+24]),
				Offset:    bo.Uint64(buf[off+24 : off+32]),
				Size:      bo.Uint64(buf[off+32 : off+40]),
				Link:      bo.Uint32(buf[off+40 : off+44]),
				Info:      bo.Uint32(buf[off+44 : off+48]),
				AddrAlign: bo.Uint64(buf[off+48 : off+56]),
				EntSize:   bo.Uint64(buf[off+56 : off+64]),
			}
		}
	} else {
		const shdrSize32 = 40
		buf := make([]byte, int(h.ShNum)*shdrSize32)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read 32-bit section headers: %w", err)
		}

		for i := 0; i < int(h.ShNum); i++ {
			off := i * shdrSize32
			sections[i] = Section{
				Index:     i,
				NameOff:   bo.Uint32(buf[off : off+4]),
				Type:      bo.Uint32(buf[off+4 : off+8]),
				Flags:     uint64(bo.Uint32(buf[off+8 : off+12])),
				Addr:      uint64(bo.Uint32(buf[off+12 : off+16])),
				Offset:    uint64(bo.Uint32(buf[off+16 : off+20])),
				Size:      uint64(bo.Uint32(buf[off+20 : off+24])),
				Link:      bo.Uint32(buf[off+24 : off+28]),
				Info:      bo.Uint32(buf[off+28 : off+32]),
				AddrAlign: uint64(bo.Uint32(buf[off+32 : off+36])),
				EntSize:   uint64(bo.Uint32(buf[off+36 : off+40])),
			}
		}
	}

	// Resolve Section Names using the String Table (.shstrtab)
	if int(h.ShStrIdx) < len(sections) && h.ShStrIdx != 0 {
		strSec := sections[h.ShStrIdx]
		if strSec.Size > 0 && strSec.Type == SHT_STRTAB {
			strTable := make([]byte, strSec.Size)
			if _, err := r.Seek(int64(strSec.Offset), io.SeekStart); err == nil {
				if _, err := io.ReadFull(r, strTable); err == nil {
					for i := range sections {
						sections[i].Name = extractNullString(strTable, sections[i].NameOff)
					}
				}
			}
		}
	}

	return sections, nil
}

// extractNullString extracts a null-terminated ASCII string from table starting at offset.
func extractNullString(table []byte, offset uint32) string {
	if int(offset) >= len(table) {
		return ""
	}
	end := int(offset)
	for end < len(table) && table[end] != 0 {
		end++
	}
	return string(table[offset:end])
}

// TypeString returns the mnemonic string for a section type.
func (s *Section) TypeString() string {
	if name, ok := sectionTypeNames[s.Type]; ok {
		return name
	}
	return fmt.Sprintf("0x%x", s.Type)
}

// FlagsString returns formatted flag characters (e.g. "WAX" for Write, Alloc, Execute).
func (s *Section) FlagsString() string {
	var flags []string
	if s.Flags&SHF_WRITE != 0 {
		flags = append(flags, "W")
	}
	if s.Flags&SHF_ALLOC != 0 {
		flags = append(flags, "A")
	}
	if s.Flags&SHF_EXECINSTR != 0 {
		flags = append(flags, "X")
	}
	if s.Flags&SHF_MERGE != 0 {
		flags = append(flags, "M")
	}
	if s.Flags&SHF_STRINGS != 0 {
		flags = append(flags, "S")
	}
	if s.Flags&SHF_INFO_LINK != 0 {
		flags = append(flags, "I")
	}
	if s.Flags&SHF_LINK_ORDER != 0 {
		flags = append(flags, "L")
	}
	if s.Flags&SHF_GROUP != 0 {
		flags = append(flags, "G")
	}
	if s.Flags&SHF_TLS != 0 {
		flags = append(flags, "T")
	}
	if s.Flags&SHF_COMPRESSED != 0 {
		flags = append(flags, "C")
	}
	return strings.Join(flags, "")
}

// Print outputs the core ELF header metadata to standard output.
func (e *ELFFile) Print() {
	arch := ArchitectureName(e.Header.Machine)

	bits := map[uint8]string{Class32: "32-bit", Class64: "64-bit"}
	endian := map[uint8]string{DataLE: "Little Endian", DataBE: "Big Endian"}

	fmt.Printf("File:        %s\n", e.Path)
	fmt.Printf("Arch:        %s (%s)\n", arch, bits[e.Header.Class])
	fmt.Printf("Endian:      %s\n", endian[e.Header.Data])
	fmt.Printf("Entry:       0x%x\n", e.Header.Entry)
	fmt.Printf("Section Hdr: 0x%x (Count: %d, StrIdx: %d)\n", e.Header.ShOff, e.Header.ShNum, e.Header.ShStrIdx)
}

// PrintSections displays a formatted table of all section headers.
func (e *ELFFile) PrintSections() {
	if len(e.Sections) == 0 {
		fmt.Println("\nNo section headers present.")
		return
	}

	fmt.Printf("\n[Section Headers (%d entries)]\n", len(e.Sections))
	if e.Header.Class == Class64 {
		fmt.Printf("  [Nr] %-20s %-12s %-16s %-8s %-8s %-6s %-4s %-4s %-5s\n",
			"Name", "Type", "Address", "Offset", "Size", "Flags", "Link", "Info", "Align")
		for _, s := range e.Sections {
			fmt.Printf("  [%2d] %-20s %-12s %016x %08x %08x %-6s %-4d %-4d %-5d\n",
				s.Index, s.Name, s.TypeString(), s.Addr, s.Offset, s.Size, s.FlagsString(), s.Link, s.Info, s.AddrAlign)
		}
	} else {
		fmt.Printf("  [Nr] %-20s %-12s %-8s %-8s %-8s %-6s %-4s %-4s %-5s\n",
			"Name", "Type", "Address", "Offset", "Size", "Flags", "Link", "Info", "Align")
		for _, s := range e.Sections {
			fmt.Printf("  [%2d] %-20s %-12s %08x %08x %08x %-6s %-4d %-4d %-5d\n",
				s.Index, s.Name, s.TypeString(), s.Addr, s.Offset, s.Size, s.FlagsString(), s.Link, s.Info, s.AddrAlign)
		}
	}
	fmt.Println("\nKey to Flags: W (write), A (alloc), X (execute), M (merge), S (strings), I (info), L (link order), G (group), T (TLS), C (compressed)")
}