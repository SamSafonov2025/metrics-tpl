package main

import (
	"log"
	"os"
)

// main is allowed to call os.Exit
func main() {
	if len(os.Args) < 2 {
		os.Exit(1) // This is OK in main
	}
}

// helper is NOT in main, so os.Exit should be flagged
func helper() {
	os.Exit(1) // want "os.Exit must only be called in main function of main package"
}

// anotherHelper uses log.Fatal - should be flagged
func anotherHelper() {
	log.Fatal("error") // want "log.Fatal must only be called in main function of main package"
}
