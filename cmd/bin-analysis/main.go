package main

import (
    "fmt"
    "os"

    "github.com/Ri4ards2006/go-core-lab/internal/binary"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: bin-analysis <elf-file>")
        os.Exit(1)
    }

    elf, err := binary.ParseELF(os.Args[1])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    elf.Print()
}