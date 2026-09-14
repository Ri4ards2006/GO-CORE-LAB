// ═══════════════════════════════════════════════════════════════════════════
// Package binary provides Program Header Table (Segment) extraction and
// permissions/type decoding for ELF binaries.
// ═══════════════════════════════════════════════════
package binary

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// Segment Types (p_type)
const (
	PT_NULL         uint32 = 0          // Unused segment entry
	PT_LOAD         uint32 = 1          // Loadable segment
	PT_DYNAMIC      uint32 = 2          // Dynamic linking information
	PT_INTERP       uint32 = 3          // Program interpreter path
	PT_NOTE         uint32 = 4          // Auxiliary information (notes)
	PT_SHLIB        uint32 = 5          // Reserved
	PT_PHDR         uint32 = 6          // Location of program header table itself
	PT_TLS          uint32 = 7          // Thread-local storage template
	PT_LOOS         uint32 = 0x60000000 // Start of OS-specific range
	PT_GNU_EH_FRAME uint32 = 0x6474e550 // GCC .eh_frame_hdr segment
	PT_GNU_STACK    uint32 = 0x6474e551 // Stack execution permissions
	PT_GNU_RELRO    uint32 = 0x6474e552 // Read-only after relocation
	PT_GNU_PROPERTY uint32 = 0x6474e553 // GNU .note.gnu.property segment
	PT_HIOS         uint32 = 0x6fffffff // End of OS-specific range
	PT_LOPROC       uint32 = 0x70000000 // Start of processor-specific range
	PT_ARM_EXIDX    uint32 = 0x70000001 // ARM exception unwinding index table
	PT_HIPROC       uint32 = 0x7fffffff // End of processor-specific range
)

// Segment Flags (p_flags)
const (
	PF_X        uint32 = 0x1        // Execute permission
	PF_W        uint32 = 0x2        // Write permission
	PF_R        uint32 = 0x4        // Read permission
	PF_MASKOS   uint32 = 0x0ff00000 // OS-specific flags
	PF_MASKPROC uint32 = 0xf0000000 // Processor-specific flags
)

// segmentTypeNames maps segment types to human-readable names.
var segmentTypeNames = map[uint32]string{
	PT_NULL:         "NULL",
	PT_LOAD:         "LOAD",
	PT_DYNAMIC:      "DYNAMIC",
	PT_INTERP:       "INTERP",
	PT_NOTE:         "NOTE",
	PT_SHLIB:        "SHLIB",
	PT_PHDR:         "PHDR",
	PT_TLS:          "TLS",
	PT_GNU_EH_FRAME: "GNU_EH_FRAME",
	PT_GNU_STACK:    "GNU_STACK",
	PT_GNU_RELRO:    "GNU_RELRO",
	PT_GNU_PROPERTY: "GNU_PROPERTY",
	PT_ARM_EXIDX:    "ARM_EXIDX",
}

// Segment models an unpacked ELF Program Header entry.
type Segment struct {
	Index    int    // Index in program header table
	Type     uint32 // Segment type (p_type)
	Flags    uint32 // Segment flags / permissions (p_flags)
	Offset   uint64 // Segment file offset (p_offset)
	VAddr    uint64 // Segment virtual address in memory (p_vaddr)
	PAddr    uint64 // Segment physical address (p_paddr)
	FileSize uint64 // Segment size in file in bytes (p_filesz)
	MemSize  uint64 // Segment size in memory in bytes (p_memsz)
	Align    uint64 // Segment alignment constraint (p_align)
}

// TypeString returns the mnemonic string for a segment type.
func (s Segment) TypeString() string {
	if name, ok := segmentTypeNames[s.Type]; ok {
		return name
	}
	return fmt.Sprintf("0x%x", s.Type)
}

// FlagsString returns a formatted permission string (e.g. "R E", "RW ", "R  ").
func (s Segment) FlagsString() string {
	var perms []string
	if s.Flags&PF_R != 0 {
		perms = append(perms, "R")
	} else {
		perms = append(perms, " ")
	}

	if s.Flags&PF_W != 0 {
		perms = append(perms, "W")
	} else {
		perms = append(perms, " ")
	}

	if s.Flags&PF_X != 0 {
		perms = append(perms, "E")
	} else {
		perms = append(perms, " ")
	}

	return strings.Join(perms, "")
}

// parseSegments traverses the Program Header Table at e_phoff and decodes all entries.
func parseSegments(r io.ReadSeeker, h *ELFHeader, bo binary.ByteOrder) ([]Segment, error) {
	if h.PhOff == 0 || h.PhNum == 0 {
		return nil, nil
	}

	if _, err := r.Seek(int64(h.PhOff), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to program header table (offset 0x%x): %w", h.PhOff, err)
	}

	segments := make([]Segment, h.PhNum)

	if h.Class == Class64 {
		const phdrSize64 = 56 // Elf64_Phdr is 56 bytes
		buf := make([]byte, int(h.PhNum)*phdrSize64)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read 64-bit program headers: %w", err)
		}

		for i := 0; i < int(h.PhNum); i++ {
			off := i * phdrSize64
			segments[i] = Segment{
				Index:    i,
				Type:     bo.Uint32(buf[off : off+4]),
				Flags:    bo.Uint32(buf[off+4 : off+8]),
				Offset:   bo.Uint64(buf[off+8 : off+16]),
				VAddr:    bo.Uint64(buf[off+16 : off+24]),
				PAddr:    bo.Uint64(buf[off+24 : off+32]),
				FileSize: bo.Uint64(buf[off+32 : off+40]),
				MemSize:  bo.Uint64(buf[off+40 : off+48]),
				Align:    bo.Uint64(buf[off+48 : off+56]),
			}
		}
	} else {
		const phdrSize32 = 32 // Elf32_Phdr is 32 bytes
		buf := make([]byte, int(h.PhNum)*phdrSize32)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read 32-bit program headers: %w", err)
		}

		for i := 0; i < int(h.PhNum); i++ {
			off := i * phdrSize32
			segments[i] = Segment{
				Index:    i,
				Type:     bo.Uint32(buf[off : off+4]),
				Offset:   uint64(bo.Uint32(buf[off+4 : off+8])),
				VAddr:    uint64(bo.Uint32(buf[off+8 : off+12])),
				PAddr:    uint64(bo.Uint32(buf[off+12 : off+16])),
				FileSize: uint64(bo.Uint32(buf[off+16 : off+20])),
				MemSize:  uint64(bo.Uint32(buf[off+20 : off+24])),
				Flags:    bo.Uint32(buf[off+24 : off+28]),
				Align:    uint64(bo.Uint32(buf[off+28 : off+32])),
			}
		}
	}

	return segments, nil
}

// PrintSegments displays a formatted table of all program header segments.
func (e *ELFFile) PrintSegments() {
	if len(e.Segments) == 0 {
		fmt.Println("\nNo program headers (segments) present in binary.")
		return
	}

	fmt.Printf("\n[Program Headers / Segments (%d entries)]\n", len(e.Segments))
	if e.Header.Class == Class64 {
		fmt.Printf("  [Nr] %-14s %-8s %-16s %-16s %-10s %-10s %-8s %-6s\n",
			"Type", "Flags", "Offset", "VirtAddr", "FileSiz", "MemSiz", "Align", "Perms")
		for _, s := range e.Segments {
			fmt.Printf("  [%2d] %-14s %08x %016x %016x %08x   %08x   %-8d %s\n",
				s.Index, s.TypeString(), s.Flags, s.Offset, s.VAddr, s.FileSize, s.MemSize, s.Align, s.FlagsString())
		}
	} else {
		fmt.Printf("  [Nr] %-14s %-8s %-8s %-8s %-8s %-8s %-8s %-6s\n",
			"Type", "Flags", "Offset", "VirtAddr", "FileSiz", "MemSiz", "Align", "Perms")
		for _, s := range e.Segments {
			fmt.Printf("  [%2d] %-14s %08x %08x %08x %08x %08x %-8d %s\n",
				s.Index, s.TypeString(), s.Flags, s.Offset, s.VAddr, s.FileSize, s.MemSize, s.Align, s.FlagsString())
		}
	}
}

