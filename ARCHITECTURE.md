# GO-CORE-LAB Architecture Specification

## 1. Project Overview & Philosophy

**GO-CORE-LAB** is a high-performance, modular research framework developed in Go for systems-level binary inspection, hardware telemetry, and network protocol probing. 

Designed for low-level reverse engineering, embedded firmware analysis, and protocol dissection, the project bridges raw hardware interactions (STM32, ESP32, logic analyzers via SPI/I2C/UART) with software-side binary forensics and packet telemetry.

```
+-------------------------------------------------------------------------+
|                               GO-CORE-LAB                               |
+------------------------------------+------------------------------------+
|          Hardware & Telemetry      |          Binary Forensics          |
|  +-------------------------------+ |  +-------------------------------+ |
|  |     Hardware Bridge (UART)    | |  |    ELF / PE / Mach-O Parser   | |
|  |   SPI / I2C Signal Capture    | |  |   Section & Symbol Extractor  | |
|  +-------------------------------+ |  +-------------------------------+ |
|  |      Network Probing Engine   | |  |   Custom Bytecode Inspector   | |
|  |    Raw Socket / Packet Sniff  | |  |   (.flx / Flux-Lang Stream)   | |
|  +-------------------------------+ |  +-------------------------------+ |
+------------------------------------+------------------------------------+
|                         Low-Level Core Engine                           |
|        Zero-Copy Slices  •  Endianness Decoders  •  Binary Encoders      |
+-------------------------------------------------------------------------+
```

### Core Tenets
- **Zero Heavy Dependencies:** Written primarily with the Go standard library (`encoding/binary`, `os`, `io`, `syscall`, `net`) to produce self-contained, statically-linked binaries (`CGO_ENABLED=0`) runnable across Arch Linux, embedded Linux (ARM64/AArch64), and containerized analysis labs.
- **Direct Byte Stream Parsing:** Avoids opaque black-box libraries; implements explicit binary format specifications from first principles to ensure complete control over memory layouts and offset arithmetic.
- **Predictable Memory Footprint:** Employs bounded buffer allocations, memory reuse, and slice-based windowing over memory-mapped files to handle large firmware images and packet dumps with minimal garbage collection pressure.

---

## 2. Directory Layout & Package Boundaries

The repository follows standard Go project layout conventions (`cmd/`, `internal/`, `pkg/`):

```
GO-CORE-LAB/
├── cmd/
│   ├── bin-analysis/           # CLI tool for binary inspection & header extraction
│   │   └── main.go             # Entrypoint: parses arguments, invokes internal/binary
│   └── net-probe/              # CLI tool for packet capture & network telemetry (Planned)
├── internal/                   # Private application logic (non-exportable)
│   ├── binary/                 # Binary format parsers, architecture decoders
│   │   ├── elf.go              # [Legacy v1] Basic ELF header parser
│   │   ├── elf2.go             # [Refactored v2] 32/64-bit ELF parser & section model
│   │   ├── architecture_scanner.go # [Prototype] Architecture constant mappings
│   │   └── playground.go       # [Scratchpad] Language experiments & structural tests
│   └── learn.go                # [Notes] Go syntax comparisons & didactic scratchpad
├── pkg/                        # Reusable, public libraries (Exportable)
│   ├── codec/                  # Hex dumps, endianness helpers, binary packing
│   ├── report/                 # Formatted output engines (JSON, terminal tables, tree)
│   └── signal/                 # Hardware signal processing & protocol frames
├── modules/                    # Pluggable research & specialized domain engines
│   ├── flux-bytecode/          # .flx custom bytecode analyzer & disassembler (Phase 2)
│   └── hw-bridge/              # UART/SPI/I2C bridge controllers (Phase 3)
├── testdata/                   # Fixtures: ELF binaries, PCAPs, raw dumps, firmware ROMs
├── configs/                    # Static configuration files
├── docs/                       # Architecture diagrams, hardware wiring schematics
├── go.mod                      # Module definition (github.com/Ri4ards2006/go-core-lab)
├── ARCHITECTURE.md             # This document
└── ROADMAP.md                  # Development phases & milestone tracking
```

### Package Responsibilities

| Directory | Scope | Purpose |
|:---|:---|:---|
| `cmd/bin-analysis` | Executable | Command-line interface for analyzing ELF binaries, printing metadata, and dumping structural tables. |
| `cmd/net-probe` | Executable | Network monitoring daemon for raw socket capture, packet dissection, and telemetry streaming. |
| `internal/binary` | Internal | Core engine for parsing binary file formats (ELF32/64, PE32+, Mach-O, custom bytecode). |
| `pkg/report` | Exported | Reusable terminal formatting utilities (colored hex-view, table visualizers, JSON exporters). |
| `pkg/codec` | Exported | Low-level byte manipulation, bitmask operations, and custom deserializers. |
| `modules/` | Extensions | Self-contained subsystems for specialized targets (e.g., `.flx` bytecode runner, logic analyzer decoders). |

---

## 3. Binary Analysis Pipeline

The binary analysis pipeline decodes raw binary files into structured domain objects through a staged parsing process:

```
[Raw File on Disk / Stream]
            │
            ▼
┌───────────────────────────┐
│ 1. Identification Phase   │  Read 16-byte e_ident:
│    (Magic Byte Validation)│  Check [0x7f, 'E', 'L', 'F']
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│ 2. Header Extraction      │  Detect Class (32-bit vs 64-bit) & Endianness (LE/BE)
│    (ELF Identification)   │  Bind binary.ByteOrder (binary.LittleEndian / BigEndian)
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│ 3. Architecture & Offsets │  Decode Target Machine (x86_64, ARM, AArch64, RISC-V)
│    (e_machine, e_entry)   │  Read Entrypoint, Section Header Offset (e_shoff),
│                           │  Section Count (e_shnum), String Table Index (e_shstrndx)
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│ 4. Section Table Parsing  │  Seek to e_shoff, iterate e_shnum entries
│    (Section Headers)      │  Extract: Name Offset, Type, Flags, Addr, Offset, Size
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│ 5. String Resolution      │  Read .shstrtab section content
│    (.shstrtab / .strtab)  │  Resolve null-terminated ASCII strings for section names
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│ 6. Output & Symbol Dump   │  Emit structured *ELFFile model to CLI formatter
└───────────────────────────┘
```

### ELF Header Layout Specification

The pipeline parses standard System V / Linux ELF binaries conforming to the following binary structure:

```
0x00 - 0x03:  Magic Bytes (0x7F 0x45 0x4C 0x46)
0x04:        EI_CLASS (1 = 32-bit, 2 = 64-bit)
0x05:        EI_DATA  (1 = 2's complement Little Endian, 2 = 2's complement Big Endian)
0x06:        EI_VERSION (1 = Current)
0x07:        EI_OSABI (Target OS ABI, e.g., 0x00 System V, 0x03 Linux)
0x08 - 0x0F: Padding / Unused bytes
0x10 - 0x11: e_type (1 = Relocatable, 2 = Executable, 3 = Shared Object, 4 = Core)
0x12 - 0x13: e_machine (0x03 = x86, 0x28 = ARM, 0x3E = x86_64, 0xB7 = AArch64, 0xF3 = RISC-V)
0x14 - 0x17: e_version
0x18 - 0x1F: e_entry (Virtual entrypoint address — 4 bytes on 32-bit, 8 bytes on 64-bit)
0x20 - 0x27: e_phoff (Program header table file offset)
0x28 - 0x2F: e_shoff (Section header table file offset)
0x30 - 0x33: e_flags (Processor-specific flags)
0x34 - 0x35: e_ehsize (ELF header size in bytes)
0x36 - 0x37: e_phentsize (Size of one program header entry)
0x38 - 0x39: e_phnum (Number of program header entries)
0x3A - 0x3B: e_shentsize (Size of one section header entry)
0x3C - 0x3D: e_shnum (Number of section header entries)
0x3E - 0x3F: e_shstrndx (Section header index of section name string table)
```

---

## 4. Network Probing & Telemetry Engine (`net-probe`)

`cmd/net-probe` provides passive and active network analysis capabilities designed for embedded communications and edge devices.

```
┌─────────────────────────────────────────────────────────────┐
│                    Capture Interface                        │
│   AF_PACKET (Linux Raw Sockets) / libpcap / TAP Interfaces  │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                  BPF Filter Engine                          │
│   In-kernel or userspace packet filtering (e.g. port, proto)│
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                  Layer Dissectors                           │
│   Ethernet II ──► IPv4 / IPv6 ──► TCP / UDP / ICMP / Custom │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│             Stream Reconstruction & Telemetry               │
│   • Ring buffer circular queue                              │
│   • Real-time payload hex-streaming                         │
│   • Export to .pcapng or JSON-lines telemetry               │
└─────────────────────────────────────────────────────────────┘
```

### Key Modules:
1. **Raw Ingestion:** Zero-copy packet capture using Linux raw sockets (`syscall.AF_PACKET`, `syscall.SOCK_RAW`) or PCAP bindings.
2. **Protocol Dissectors:** Layer-by-layer header unpackers operating on byte slices:
   - Layer 2: Ethernet frame headers (MAC src/dst, EtherType).
   - Layer 3: IPv4/IPv6 headers (TTL, Protocol, IP src/dst, checksum verification).
   - Layer 4: TCP/UDP ports, sequence numbers, flags (SYN, ACK, FIN, RST, PSH).
3. **Telemetry Exporter:** Formatted event streams emitting structured telemetry to stdout or remote storage.

---

## 5. Memory & Allocation Strategy

Low-level binary parsing and high-throughput network monitoring require strict memory management to avoid GC pauses and allocation churn:

1. **Slice Slicing without Heap Reallocation:**
   - When extracting fields from raw buffers, the parser uses sub-slices (`raw[18:20]`, `raw[24:32]`) directly rather than allocating intermediate byte slices.
2. **Pre-sized Static Buffers for Headers:**
   - Standard ELF headers are fixed-size (64 bytes for 64-bit ELF, 52 bytes for 32-bit ELF). Reading uses fixed stack-allocated or reusable buffers (`[64]byte`).
3. **Memory Mapping for Massive Binaries (`mmap`):**
   - For multi-megabyte firmware dumps, ELF files, or PCAP traces, files are memory-mapped (`syscall.Mmap`) directly into the virtual address space. This allows random offset access to sections (`e_shoff`) and string tables without loading entire files into the Go heap.
4. **Buffered I/O and Ring Buffers:**
   - Packet streaming in `net-probe` utilizes circular pre-allocated ring buffers to decouple packet reception from decoding and terminal rendering.

