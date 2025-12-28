package a

import (
	"log"
	"os"
)

// BadFunction uses os.Exit outside of main - should be flagged
func BadFunction() {
	os.Exit(1) // want "os.Exit must only be called in main function of main package"
}

// AnotherBadFunc uses log.Fatal - should be flagged
func AnotherBadFunc() {
	log.Fatal("error") // want "log.Fatal must only be called in main function of main package"
}

// GoodFunction does not use os.Exit or panic
func GoodFunction() {
	// This is OK
}
