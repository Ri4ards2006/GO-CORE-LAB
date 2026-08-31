# GO-CORE-LAB Development Roadmap

This document outlines the strategic milestones and technical progression for **GO-CORE-LAB**.

---

## 🎯 Milestone Overview

```
Phase 1 ──► Phase 2 ──► Phase 3 ──► Phase 4
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Binary       │    │ Custom .flx  │    │ Network & HW │    │ Zero-Alloc   │
│ Forensics &  ├───►│ Bytecode     ├───►│ Telemetry    ├───►│ Optimization │
│ ELF Engine   │    │ Inspector    │    │ Engine       │    │ & CLI Suite  │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘
```

---

## 📍 Phase 1: Binary Formats & ELF Parsing Engine

**Objective:** Complete low-level parsing for ELF binaries (32-bit and 64-bit), section table resolution, and symbol table extraction from raw byte streams.

- [x] **ELF Magic & Identification:** Validate `0x7F 'E' 'L' 'F'`, parse Class (32/64-bit), and decode Endianness (`binary.LittleEndian` / `binary.BigEndian`).
- [x] **Architecture Identification:** Decode `e_machine` to identify x86, x86_64, ARM, AArch64, and RISC-V targets.
- [x] **ELF File Header Offsets:** Read Entrypoint (`e_entry`), Section Header Table Offset (`e_shoff`), Section Count (`e_shnum`), and String Table Index (`e_shstrndx`).
- [x] **Section Header Table Parser:**
  - [x] Implement `Elf64_Shdr` / `Elf32_Shdr` struct unpacking from `e_shoff`.
  - [x] Read section attributes: `sh_name`, `sh_type` (`SHT_PROGBITS`, `SHT_SYMTAB`, `SHT_STRTAB`, `SHT_NOBITS`), `sh_flags` (`SHF_WRITE`, `SHF_ALLOC`, `SHF_EXECINSTR`), `sh_addr`, `sh_offset`, `sh_size`.
- [x] **Section Name String Table (`.shstrtab`) Resolution:**
  - [x] Seek to `.shstrtab` raw data offset.
  - [x] Extract null-terminated strings using `sh_name` string table offsets to populate human-readable section names (`.text`, `.data`, `.rodata`, `.bss`).
- [ ] **Symbol Table (`.symtab` & `.dynsym`) Extractor:**
  - [ ] Parse `Elf64_Sym` / `Elf32_Sym` entries (`st_name`, `st_info`, `st_other`, `st_shndx`, `st_value`, `st_size`).
  - [ ] Cross-reference string table (`.strtab`) to resolve function and global variable names.
- [ ] **Program Header Table (Segments):**
  - [ ] Parse `Elf64_Phdr` entries (`p_type` `PT_LOAD`, `PT_DYNAMIC`, `PT_INTERP`, `p_vaddr`, `p_memsz`, `p_flags`).
- [ ] **Extensibility: PE (Portable Executable) & Mach-O Parsers:**
  - [ ] Implement PE DOS Header (`MZ`) and PE Header signature verification.
  - [ ] Implement Mach-O Magic (`0xFEEDFACE` / `0xFEEDFACF`) detector.

---

## 📍 Phase 2: Custom Bytecode & `.flx` Inspector

**Objective:** Cross-integrate with the `flux-lang` ecosystem by building an inspector and disassembler for `.flx` binary bytecode containers.

- [x] **`.flx` Container Specification:**
  - [x] Define container header (Magic `0x46 0x4C 0x58 0x01` -> `FLX\x01`, version, flags).
  - [x] Parse metadata table (author, compiler version, timestamp).
- [x] **Constant Pool Deserialization:**
  - [x] Decode integer constants, floating-point literals, UTF-8 strings, and symbol identifiers.
- [x] **Bytecode Stream Disassembler:**
  - [x] Implement instruction decoder matching opcode tables.
  - [x] Disassemble instructions with operand resolution (e.g., `LOAD_CONST`, `STORE_FAST`, `BINARY_ADD`, `JUMP_IF_FALSE`, `CALL_FUNCTION`).
- [ ] **Execution & Control Flow Graph (CFG) Visualizer:**
  - [ ] Trace basic blocks and branch targets.
  - [ ] Generate ASCII / DOT graphs of bytecode execution paths.

---

## 📍 Phase 3: Network Probing & Hardware Telemetry

**Objective:** Implement real-time packet sniffing in `cmd/net-probe` and hardware serial/bus monitoring.

- [ ] **Network Capture Engine (`cmd/net-probe`):**
  - [ ] Raw socket capture via Linux `AF_PACKET` (`syscall.SOCK_RAW`).
  - [ ] BPF (Berkeley Packet Filter) userspace filtering interface.
- [ ] **Protocol Dissection Stack:**
  - [ ] Layer 2: Ethernet frame decoder (MAC addresses, EtherType).
  - [ ] Layer 3: IPv4 / IPv6 header unpacker (TTL, protocols, IP addresses, checksums).
  - [ ] Layer 4: TCP / UDP / ICMP segment parser (port mappings, flags, seq/ack tracking).
- [ ] **Hardware Bridge Telemetry (`internal/hw`):**
  - [ ] UART / Serial streaming reader (configurable baudrates: 115200, 9600).
  - [ ] Linux `spidev` and `i2cdev` passive bus monitor.
  - [ ] Frame synchronizer with start-of-frame (SOF) / end-of-frame (EOF) delimiters.
- [ ] **Export & Persistence:**
  - [ ] Stream packet captures to standard `.pcap` / `.pcapng` format.
  - [ ] Real-time telemetry output in JSON-lines (`.jsonl`).

---

## 📍 Phase 4: CLI Tooling, Benchmarks & Zero-Alloc Optimization

**Objective:** Polish developer experience, provide modular reporting, and ensure zero-allocation performance on resource-constrained systems.

- [ ] **Consolidated CLI Suite:**
  - [ ] Multi-command CLI for `bin-analysis` (`bin-analysis elf <file>`, `bin-analysis sections <file>`, `bin-analysis symbols <file>`, `bin-analysis flx <file>`).
  - [ ] Interactive terminal UI mode using ANSI escape codes and terminal colorizers.
- [ ] **Hex & Report Visualizer (`pkg/report`):**
  - [ ] Canonical 16-byte hex dump grid with ASCII sidebar and offset indexing.
  - [ ] Tabular output formatters with colored permission flags (`r-x`, `rw-`, `r--`).
- [ ] **Zero-Allocation & Memory Mapping:**
  - [ ] Integrate `syscall.Mmap` for parsing multi-gigabyte files without heap exhaustion.
  - [ ] Utilize `sync.Pool` for buffer reuse in high-throughput network packet capture.
- [ ] **Testing, CI & Fuzzing:**
  - [ ] Automated Go unit tests against curated test fixtures in `testdata/`.
  - [ ] Go native fuzz testing (`go test -fuzz`) for binary header and packet parsers to prevent panics and bounds out-of-range crashes.
