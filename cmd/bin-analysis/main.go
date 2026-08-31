package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/Ri4ards2006/go-core-lab/internal/binary"
	"github.com/Ri4ards2006/go-core-lab/internal/binary/flx"
)

func printUsage() {
	fmt.Println("Usage: bin-analysis [command] [options] <file>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  elf <file>                Inspect ELF file header and section table")
	fmt.Println("  flx <file>                Full inspection of flux-lang bytecode (.flx)")
	fmt.Println("  flx header <file>         Print .flx container header and metadata")
	fmt.Println("  flx pool <file>           Dump .flx constant pool table")
	fmt.Println("  flx disasm <file>         Print .flx bytecode disassembly")
	fmt.Println("  <file>                    Auto-detect file format (ELF / FLX) and inspect")
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	arg1 := os.Args[1]

	switch arg1 {
	case "help", "-h", "--help":
		printUsage()
		return

	case "elf":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing target file for 'elf' command")
			fmt.Println("Usage: bin-analysis elf <file>")
			os.Exit(1)
		}
		analyzeELF(os.Args[2])

	case "flx":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing target file or subcommand for 'flx'")
			fmt.Println("Usage: bin-analysis flx [header|pool|disasm] <file>")
			os.Exit(1)
		}

		subCmd := os.Args[2]
		switch subCmd {
		case "header":
			if len(os.Args) < 4 {
				fmt.Println("Error: missing target file for 'flx header'")
				os.Exit(1)
			}
			analyzeFLXHeader(os.Args[3])

		case "pool":
			if len(os.Args) < 4 {
				fmt.Println("Error: missing target file for 'flx pool'")
				os.Exit(1)
			}
			analyzeFLXPool(os.Args[3])

		case "disasm":
			if len(os.Args) < 4 {
				fmt.Println("Error: missing target file for 'flx disasm'")
				os.Exit(1)
			}
			analyzeFLXDisasm(os.Args[3])

		default:
			// If 3rd arg is the file itself (e.g. `bin-analysis flx sample.flx`)
			analyzeFLXFull(subCmd)
		}

	default:
		// Auto-detect format based on magic bytes
		autoDetectAndAnalyze(arg1)
	}
}

func analyzeELF(path string) {
	elfFile, err := binary.ParseELF(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing ELF %q: %v\n", path, err)
		os.Exit(1)
	}

	elfFile.Print()
	elfFile.PrintSections()
}

func analyzeFLXHeader(path string) {
	flxFile, err := flx.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing .flx header %q: %v\n", path, err)
		os.Exit(1)
	}
	flxFile.PrintHeader()
}

func analyzeFLXPool(path string) {
	flxFile, err := flx.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing .flx pool %q: %v\n", path, err)
		os.Exit(1)
	}
	flxFile.Pool.Print()
}

func analyzeFLXDisasm(path string) {
	flxFile, err := flx.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing .flx disassembly %q: %v\n", path, err)
		os.Exit(1)
	}
	flx.PrintDisassembly(flxFile.Instructions)
}

func analyzeFLXFull(path string) {
	flxFile, err := flx.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing .flx file %q: %v\n", path, err)
		os.Exit(1)
	}
	flxFile.PrintSummary()
}

func autoDetectAndAnalyze(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file %q: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()

	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading magic bytes from %q: %v\n", path, err)
		os.Exit(1)
	}

	// Rewind file position
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		fmt.Fprintf(os.Stderr, "Error rewinding file %q: %v\n", path, err)
		os.Exit(1)
	}

	// Check for ELF magic (0x7F 'E' 'L' 'F')
	if bytes.Equal(magic, []byte{0x7f, 'E', 'L', 'F'}) {
		elfFile, err := binary.ParseELFReader(f, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing ELF: %v\n", err)
			os.Exit(1)
		}
		elfFile.Print()
		elfFile.PrintSections()
		return
	}

	// Check for FLX magic ('F' 'L' 'X' '\x01')
	if bytes.Equal(magic, flx.Magic[:]) {
		flxFile, err := flx.ParseReader(f, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing .flx: %v\n", err)
			os.Exit(1)
		}
		flxFile.PrintSummary()
		return
	}

	fmt.Fprintf(os.Stderr, "Error: unsupported file format for %q (magic: % x)\n", path, magic)
	os.Exit(1)
}