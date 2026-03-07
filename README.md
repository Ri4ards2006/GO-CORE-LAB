<div align="center">
  <img src="https://capsule-render.vercel.app/api?type=rect&color=0D1117&height=140&section=header&text=GO-CORE-LAB&fontSize=60&fontColor=00FF9C&theme=dark&stroke=00FF9C&strokeWidth=2" width="100%" />

  <br/>

  ### ⚡ Low-Level Research Framework — Hardware to Binary ⚡

  <sub>Arch Linux Native &nbsp;•&nbsp; Go-Powered Runtime &nbsp;•&nbsp; Embedded Hardware Integration</sub>

  <br/>

  [![OS](https://img.shields.io/badge/OS-Arch_Linux-1793D1?style=flat-square&logo=arch-linux&logoColor=white)](https://archlinux.org)
  [![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
  [![C++](https://img.shields.io/badge/C++-GCC_13-00599C?style=flat-square&logo=c%2B%2B&logoColor=white)](https://gcc.gnu.org)
  [![Status](https://img.shields.io/badge/Status-Active_Development-FF4444?style=flat-square)]()
  [![License](https://img.shields.io/badge/License-MIT-22C55E?style=flat-square)](LICENSE)

</div>

---

## What This Is

**Go-Core-Lab** is a personal research framework for low-level systems engineering. It connects physical hardware (STM32, ESP32, logic analyzers) with software-side analysis tools — binary inspection, network capture, and protocol dissection — all wired together in a modular Go runtime.

This is not a CTF toolkit or a script collection. It's a structured lab environment built to learn by doing: writing real parsers, talking to real hardware over SPI/I2C/UART, and understanding what happens at every layer of the stack.

**Core focus areas:**
- Binary analysis (ELF/PE, hex dumps, syscall tracing)
- Hardware signal capture and parsing (SPI, I2C, UART via Linux spidev/i2cdev)
- Network traffic inspection and protocol reconstruction
- Lab automation (Proxmox, Docker, OpenOCD)

---

## Module Overview

Each module is a self-contained, statically compiled binary. No shared runtime dependencies — drop it anywhere and run.

| Module | Layer | Purpose | Status |
|:---|:---|:---|:---|
| **`BIN-ANALYS`** | Binary | ELF/PE header parsing, symbol extraction, hex diff | `Stable` |
| **`HW-BRIDGE`** | Hardware | Real-time SPI/I2C/UART capture from STM32/ESP32 | `Active` |
| **`NET-PROBE`** | Network | TCP/UDP packet capture, protocol filtering, pcap export | `Building` |
| **`Z-ORCH`** | Infrastructure | Lab automation — Proxmox VM spin-up, Docker compose orchestration | `Backlog` |

---

## Repository Structure

```
go-core-lab/
├── cmd/                        # One main() per tool
│   ├── bin-analys/
│   ├── hw-bridge/
│   ├── net-probe/
│   └── z-orch/
├── internal/                   # Private application logic
│   ├── hw/                     # Hardware abstraction (spidev, i2cdev, UART)
│   ├── binary/                 # ELF/PE parsers, hex utilities
│   ├── net/                    # Packet capture, protocol parsers
│   └── store/                  # Output persistence (SQLite, flatfiles)
├── pkg/                        # Exported, reusable packages
│   ├── signal/                 # Signal types, transforms (FFT, filtering)
│   ├── codec/                  # Encoding/decoding (hex, base64, custom)
│   └── report/                 # Output formatters (JSON, hex dump, table)
├── modules/                    # Pluggable research modules
├── testdata/                   # Sample captures, ELFs, PCAPs for unit tests
├── docs/
│   ├── architecture.md
│   ├── hardware-setup.md       # Wiring diagrams, device config
│   └── research-notes/         # Per-session findings and protocol specs
├── scripts/
│   ├── build.sh                # Cross-compile targets (amd64, arm, arm64)
│   ├── flash.sh                # STM32 flashing via OpenOCD
│   └── lint.sh
├── configs/                    # Runtime config (YAML/TOML, no secrets)
├── Makefile
├── go.mod
└── go.sum
```

---

## Build

### Requirements

```bash
# Arch Linux
sudo pacman -S go make gcc libpcap openocd
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Compile

```bash
# Build all binaries (host)
make build

# Static binary — no libc, maximum portability
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o dist/bin-analys ./cmd/bin-analys/

# Cross-compile for ARM (Raspberry Pi, embedded companion)
GOOS=linux GOARCH=arm64 \
  go build -trimpath -ldflags="-s -w" -o dist/hw-bridge-arm ./cmd/hw-bridge/

# Run tests with coverage
make test
```

> `-trimpath -ldflags="-s -w"` strips debug info and build paths — smaller binary, no source paths leaked in the output.

---

## Hardware Setup

Tested hardware:

| Component | Role |
|:---|:---|
| STM32F4 series | Target MCU — firmware analysis, signal generation |
| ESP32 | Wireless protocol target |
| Saleae Logic / sigrok | Bus capture (SPI, I2C, UART) |
| Digital oscilloscope | Signal timing and integrity |
| Professional soldering station | Board-level rework |

Full wiring diagrams and device config → [`docs/hardware-setup.md`](docs/hardware-setup.md)

---

## Lab Environment

```
Host:       Arch Linux (riced, kernel 6.x)
Hypervisor: Proxmox VE
Containers: Docker (isolated analysis environments)
VMs:        Kali (network work), Windows (PE analysis)
Toolchain:  Ghidra · GDB · Wireshark · OpenOCD · sigrok
```

---

## Research Workflow

```
1. Capture  →  hardware signal or network traffic  →  raw bytes to testdata/
2. Parse    →  run module against capture           →  structured output
3. Analyze  →  correlate binary ↔ protocol ↔ signal
4. Document →  write findings in docs/research-notes/
5. Build    →  generalize useful logic into pkg/    →  reusable for next module
```

---

## Background

19-year-old IT systems electronics apprentice (IT-Systemelektroniker). Working at the intersection of hardware and software — from soldering embedded targets to writing Go parsers that consume the data those targets produce.

**Languages:** Go (primary), C++ (firmware/drivers), Python (scripting), Assembly (learning — x86 + ARM Thumb)

---

## Roadmap

- [ ] `HW-BRIDGE` — complete SPI passive capture + Go frame parser
- [ ] `BIN-ANALYS` — add syscall trace correlation to ELF symbols
- [ ] `NET-PROBE` — BPF filter interface, pcapng export
- [ ] Assembly sandbox — annotated x86 disassembly environment
- [ ] Cross-module pipeline: hardware capture → binary diff → network correlation

---

## License

MIT — see [LICENSE](LICENSE).
