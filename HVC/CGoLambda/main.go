package main

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -L. -lhomomorphic_lib
// #include "homomorphic_lib.h"
import "C"
import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

type MyEvent struct {
	A int `json:"a"`
	B int `json:"b"`
}

func HandleRequest(ctx context.Context, event MyEvent) (int, error) {
	result := C.performHomomorphicAddition(event.A, event.B)
	return int(result), nil
}

func main() {
	lambda.Start(HandleRequest)
}
