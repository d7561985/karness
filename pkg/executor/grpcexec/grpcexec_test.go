package grpcexec

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"k8s.io/apimachinery/pkg/util/json"
)

// TestGRPC shows ˚all flow about MockServer requests
func TestGRPC(t *testing.T) {
	name := "TEST_NAME"

	desire := map[string]string{
		"message": "OK",
	}

	l, srv := CreateMockServer(Fixture{
		Err: nil,
		Res: &pb.HelloReply{Message: "OK"},
		CB: func(req *pb.HelloRequest) {
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
