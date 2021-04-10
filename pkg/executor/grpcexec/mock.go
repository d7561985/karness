package grpcexec

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

type Fixture struct {
	Err error
	Res *helloworld.HelloReply

	CB func(*helloworld.HelloRequest)
}

type MockServer struct {
	helloworld.UnimplementedGreeterServer
	Fixture
}

func (s MockServer) SayHello(_ context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	s.CB(req)
	return s.Res, s.Err
}

func CreateMockServer(fx Fixture) (net.Listener, *grpc.Server) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()

	// Enable reflection
	reflection.Register(s)

	helloworld.RegisterGreeterServer(s, &MockServer{Fixture: fx})

	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	return l, s
}
