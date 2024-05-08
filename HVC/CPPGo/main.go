package main

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -L. -lhomomorphic_lib
// #include "homomorphic_lib.h"
import "C"

func main() {
	// Example call: pass two integers to the C++ function
	C.performHomomorphicAddition(50, 20)
}
