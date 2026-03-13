package binary

import "fmt"

const (
ArchX86_64 = uint16(0x3E)
ArchARM    = uint16(0x28)
ArchARMAArch64 = uint16(0xB7)
)



func LearnControlFlow() {
x := 42

if x > 10 {
	fmt.Println("x is greater than 10:", x)
}
else if x == 10 {
	fmt.Println("x is equal to 10:", x)
}
else {
	fmt.Println("x is less than 10:", x)
}


// For Schleife einzige schleife in Go, die eine Initialisierung, eine Bedingung und eine Inkrementierung enthält.
for i := 0; i < 5; i++ {
	fmt.Printf("i: %d\n", i)
}

m := 0
for m < 5 {
	fmt.Printf("m: %d\n", m)
	m++
}


arch := ArchX86_64

switch  arch {


case ArchX86_64:
	fmt.Println("x86-64 architecture")
case ArchARM:
	fmt.Println("ARM architecture")
case ArchARMAArch64:
	fmt.Println("AArch64 architecture")
default:
	fmt.Printf("Unknown architecture: 0x%X\n", arch)	
	
}
