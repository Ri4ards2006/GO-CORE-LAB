# Go Learning Notes & C++ Comparisons

These notes capture low-level syntax comparisons and core Go mechanics explored during initial development:

```go
package binary

import "fmt"

const (
    ArchX86_64     = uint16(0x3E)
    ArchARM        = uint16(0x28)
    ArchARMAArch64 = uint16(0xB7)
)

func LearnControlFlow() {
    x := 42

    if x > 10 {
        fmt.Println("x is greater than 10:", x)
    } else if x == 10 {
        fmt.Println("x is equal to 10:", x)
    } else {
        fmt.Println("x is less than 10:", x)
    }

    // For Schleife: einzige Schleife in Go (Initialisierung; Bedingung; Inkrementierung)
    for i := 0; i < 5; i++ {
        fmt.Printf("i: %d\n", i)
    }

    m := 0
    for m < 5 {
        fmt.Printf("m: %d\n", m)
        m++
    }

    arch := ArchX86_64

    switch arch {
    case ArchX86_64:
        fmt.Println("x86-64 architecture")
    case ArchARM:
        fmt.Println("ARM architecture")
    case ArchARMAArch64:
        fmt.Println("AArch64 architecture")
    default:
        fmt.Printf("Unknown architecture: 0x%X\n", arch)
    }
}
```

---

## Language Concept Q&A

### 1. `fmt` Package
`fmt` in Go is part of the standard library and provides I/O formatting functions analogous to `printf` / `iostream` in C/C++.
- `fmt.Printf`: Formatted printing using verbs like `%d`, `%s`, `%x`, `%v`.
- `fmt.Sprintf`: Returns the formatted string instead of printing to standard output.
- `fmt.Fprintf`: Writes formatted output to any `io.Writer` (e.g., `os.Stderr`, files, network connections).

### 2. Variables & Short Declaration (`:=`)
- `x := 42` declares and infers the type of `x` at compile time (equivalent to `auto x = 42;` in C++).
- Only available within function bodies.

### 3. Slices vs Fixed Arrays
- `[4]byte`: Fixed-size array of 4 bytes allocated on stack or inline in structs.
- `[]byte`: Slice header containing a pointer to backing array, length (`len`), and capacity (`cap`).

### 4. Structs & Pointer Receivers
- Go uses composition rather than class inheritance.
- Methods are declared outside structs with explicit receiver parameters:
  ```go
  func (e *ELFFile) Print() { ... }
  ```
  `*ELFFile` uses a pointer receiver (modifies original, zero copying) similar to `this` pointer in C++.

### 5. Interfaces
- Interfaces in Go are satisfied implicitly (structural typing/duck typing).
- Any struct that implements the method signature `Print()` satisfies `type Printable interface { Print() }`.
