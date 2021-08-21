package server

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func recoveryFunc(p interface{}) error {
	fmt.Printf("p: %+v\n", p)
	return status.Errorf(codes.Internal, "Unexpected error")
}
