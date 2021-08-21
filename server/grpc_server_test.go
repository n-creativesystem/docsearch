package server

import (
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

// func init() {
// 	lis = bufconn.Listen(bufSize)
// 	s := grpc.NewServer()
// 	service, err := NewGRPCService("", "", "commonName", logrus.New())
// 	protobuf.RegisterIndexServer(s, service)
// 	go func() {
// 		if err := s.Serve(lis); err != nil {
// 			log.Fatal(err)
// 		}
// 	}()
// }
// func TestGrpc(t *testing.T) {

// }
