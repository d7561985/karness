package harness

import (
	"context"
	"github.com/d7561985/karness/pkg/controllers/harness/models"
	"github.com/d7561985/karness/pkg/executor/grpcexec"
)

type grpcAction struct {
	*models.Action
}

func NewGRPC(in *models.Action) ActionInterface {
	return &grpcAction{Action: in}
}

func (g *grpcAction) Call(ctx context.Context, request []byte) (*ActionResult, error) {
	gc := grpcexec.New()

	code, body, err := gc.Call(ctx, g.GRPC.Addr, grpcexec.Path{
		Package: g.GRPC.Package,
		Service: g.GRPC.Service,
		RPC:     g.GRPC.RPC,
	}, request)

	if err != nil {
		return nil, err
	}

	return &ActionResult{Code: code.String(), Body: body}, nil
}
