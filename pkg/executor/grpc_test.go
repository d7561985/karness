package executor

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

type fixture struct {
	err error
	res *pb.HelloReply

	cb func(*pb.HelloRequest)
}

type server struct {
	pb.UnimplementedGreeterServer
	fixture
}

func (s server) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	s.cb(req)
	return s.res, s.err
}

func createServer(fx fixture) (net.Listener, *grpc.Server) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()

	// Enable reflection
	reflection.Register(s)

	pb.RegisterGreeterServer(s, &server{fixture: fx})

	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	return l, s
}

// TestGRPC shows all flow about server requests
func TestGRPC(t *testing.T) {
	name := "TEST_NAME"

	l, srv := createServer(fixture{
		err: nil,
		res: &pb.HelloReply{Message: "OK"},
		cb: func(req *pb.HelloRequest) {
			assert.Equal(t, name, req.Name)
		},
	})

	defer l.Close()
	defer srv.Stop()

	fmt.Println(l.Addr().String())

	const symbol = "helloworld.Greeter/SayHello"

	g := NewGRPC()

	err := g.Go(l.Addr().String(), symbol, fmt.Sprintf(`{"name":"%s"}`, name))

	assert.NoError(t, err)
}
