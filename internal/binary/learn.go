package binary

import "fmt"


x := 42
// was heißt genau fmt also die wörter nicht das package fmt in Go ist ein Standardpaket,
//  das Funktionen für die Formatierung von Ein- und Ausgabe bereitstellt. 
// Es enthält Funktionen wie Println, Printf, Sprintf und viele andere, 
// die es ermöglichen, Daten in verschiedenen Formaten auszugeben oder zu formatieren. 
// In diesem Fall wird fmt.Println verwendet, um den Wert von x auf der Konsole auszugeben.

if x > 10 

{

	fmt.Println("x is greater than 10")
}
else if x == 10 {

	fmt.Println("x is equal to 10")
}

else {

	fmt.Println("x is less than 10")
}



for i := 0; i < 5; i++ { // Bro diese schleife ist logisch 
	// Aber dieses := ist doch irgendwie komisch oder ?
	// Ja, das ist eine Kurzschreibweise für die Deklaration und Initialisierung einer Variable in Go.
	//  Es bedeutet, dass die Variable "i" deklariert und gleichzeitig mit dem Wert 0 initialisiert wird.
	//  In diesem Fall wird "i" als Schleifenvariable verwendet, die von 0 bis 4 iteriert.


	fmt.Printf("i: %d\n", i)// Was bedeutet das tf ist denn das ?
}

m := 0

for m < 5 { // Das ist doch auch eine schleife oder ?
	// Ja, das ist eine andere Art von Schleife in Go, die als 
	// "while"-Schleife bezeichnet wird. In diesem Fall wird die Schleife so lange ausgeführt, wie die Bedingung "m < 5" wahr ist. Innerhalb der Schleife kannst du den Wert von "m" ändern, um sicherzustellen, dass die Schleife irgendwann endet.
	
	fmt.Printf("m: %d\n", m)
	m++ // Das erhöht den Wert von m um 1 in jedem Durchlauf der Schleife.
}

// --Switch

arch := uint16(0x3E) // Das ist eine Variable vom Typ uint16, die den Wert 0x3E 
// (62 in Dezimal) enthält. In diesem Fall könnte es sich um eine Architekturkennung handeln, die in einem ELF-Header verwendet wird.

switch arch { // Das ist eine Switch-Anweisung, die den Wert von "arch" überprüft und je nach Fall unterschiedliche Aktionen ausführt.
case 0x3E: // Wenn "arch" den Wert 0x3E hat, wird dieser Fall ausgeführt.
	fmt.Println("x86-64 architecture") // In diesem Fall wird die Ausgabe "x86-64 architecture" angezeigt.
case 0x28: // Wenn "arch" den Wert 0x28 hat, wird dieser Fall ausgeführt.
	fmt.Println("ARM architecture") // In diesem Fall wird die Ausgabe "ARM architecture" angezeigt.
default: // Wenn "arch" keinen der vorherigen Werte hat, wird dieser Fall ausgeführt.
	fmt.Printf("Unknown architecture: 0x%X\n", arch) // In diesem Fall wird die Ausgabe "Unknown architecture: 0x3E" 
	// angezeigt, wobei der Wert von "arch" in Hexadezimalform dargestellt wird.
}		