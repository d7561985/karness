package harness

import (
	"context"
	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/executor/grpcexec"
)

type grpcAction struct {
	v1alpha1.Action
}

func NewGRPC(in v1alpha1.Action) Action {
	return &grpcAction{Action: in}
}

func (g *grpcAction) Call(ctx context.Context) (*ActionResult, error) {
	gc := grpcexec.New()

	code, body, err := gc.Call(ctx, g.GRPC.Addr, grpcexec.Path{
		Package: g.GRPC.Package,
		Service: g.GRPC.Service,
		RPC:     g.GRPC.RPC,
	}, "")

	if err != nil {
		return nil, err
	}

	return &ActionResult{Code: code.String(), Body: body}, nil
}
