package main

import (
	"fmt"

	"lukechampine.com/lthash"
)

func main() {
	h := lthash.New16()

	// compute the combined hash of "Apple", "Banana", and "Orange"
	h.Add([]byte("Apple"))
	h.Add([]byte("Banana"))
	h.Add([]byte("Orange"))
	oldSum := h.Sum(nil)

	// print size of oldSum
	fmt.Println(len(oldSum))
	// replace "Banana" with "Grape"; the resulting hash is the same
	// as the combined hash of "Apple", "Grape", and "Orange".
	h.Remove([]byte("Banana"))
	h.Add([]byte("Grape"))
	h.Add([]byte("Grape"))
	h.Add([]byte("Grape"))
	h.Add([]byte("12323s"))
	newSum := h.Sum(nil)

	fmt.Println(len(newSum))
}
