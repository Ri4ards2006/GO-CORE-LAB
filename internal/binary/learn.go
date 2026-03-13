package binary

import "fmt"


x := 42

if x > 10 {

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
	// Ja, das ist eine Kurzschreibweise für die Deklaration und Initialisierung einer Variable in Go. Es bedeutet, dass die Variable "i" deklariert und gleichzeitig mit dem Wert 0 initialisiert wird. In diesem Fall wird "i" als Schleifenvariable verwendet, die von 0 bis 4 iteriert.
	

	fmt.Printf("i: %d\n", i)// Was bedeutet das tf ist denn das ?
}