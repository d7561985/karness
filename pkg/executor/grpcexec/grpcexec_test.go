package grpcexec

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/util/json"
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

// TestGRPC shows ˚all flow about server requests
func TestGRPC(t *testing.T) {
	name := "TEST_NAME"

	desire := map[string]string{
		"message": "OK",
	}

	l, srv := createServer(fixture{
		err: nil,
		res: &pb.HelloReply{Message: "OK"},
		cb: func(req *pb.HelloRequest) {
			assert.Equal(t, name, req.Name)
		},
	})

	defer l.Close()
	defer srv.Stop()

	g := New()

	path := Path{
		Package: "helloworld",
		Service: "Greeter",
		RPC:     "SayHello",
	}

	c, body, err := g.Call(context.Background(), l.Addr().String(), path, fmt.Sprintf(`{"name":"%s"}`, name))
	assert.NoError(t, err)
	assert.Equal(t, codes.OK, c)

	out := make(map[string]string)
	assert.NoError(t, json.Unmarshal(body, &out))

	assert.Equal(t, desire, out)
}
