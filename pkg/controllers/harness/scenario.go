package harness

import (
	"context"
	"fmt"
	"time"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/controllers"
	"github.com/d7561985/karness/pkg/controllers/harness/checker"
	"github.com/d7561985/karness/pkg/executor/grpcexec"
	"k8s.io/klog/v2"
)

type scenarioProcessor struct {
	entity  *v1alpha1.Scenario
	control controllers.Kube

	// only complete function is possible to increment current check
	current int
}

func newScenarioProcessor(c controllers.Kube, item *v1alpha1.Scenario) Processor {
	item.Status.Progress = sFmt(0, len(item.Spec.Events))
	item.Status.State = v1alpha1.Ready

	return &scenarioProcessor{control: c, entity: item}
}

// Start
func (s *scenarioProcessor) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			if s.Step(ctx) {
				return
			}
		}
	}
}

func (s *scenarioProcessor) Step(ctx context.Context) bool {
	s.entity.Status.State = v1alpha1.InProgress
	ev := s.entity.Spec.Events

	defer func() {
		s.entity.Status.Progress = sFmt(s.current, len(ev))

		if err := s.control.Update(s.entity); err != nil {
			klog.Errorf("scenario processor: %w", err)
		}
	}()

	if !s.process(ctx, s.entity.Spec.Events[s.current]) {
		s.entity.Status.State = v1alpha1.Failed
		// exit on fail
		return true
	}

	if len(ev) <= s.current {
		s.entity.Status.State = v1alpha1.Complete
		return true
	}

	return false
}

func (s *scenarioProcessor) process(ctx context.Context, event v1alpha1.Event) bool {
	status, actRes, err := s.action(ctx, event.Action)
	if err != nil {
		// ToDo: write error
		return true
	}

	return s.checkComplete(event.Complete.Condition, status, actRes)
}

func (s *scenarioProcessor) action(ctx context.Context, a v1alpha1.Action) (status string, res []byte, err error) {
	if a.GRPC != nil {
		gc := grpcexec.New()

		code, body, err := gc.Call(ctx, a.GRPC.Addr, grpcexec.Path{
			Package: a.GRPC.Package,
			Service: a.GRPC.Service,
			RPC:     a.GRPC.RPC,
		}, "")

		if err != nil {
			klog.Errorf("scenario progress with action %q grpc call error %w", a.Name, err)
			// ok=true:  we want to try again
			return "", nil, err
		}

		return code.String(), body, nil
	}

	return "OK", nil, nil
}

func (s *scenarioProcessor) checkComplete(c []v1alpha1.Condition, status string, actRes []byte) bool {
	for _, condition := range c {
		if condition.Response != nil {
			if !checker.ResCheck(*condition.Response).Is(status, actRes) {
				return false
			}
		}
	}

	s.current++
	return true
}

func sFmt(start, end int) string {
	return fmt.Sprintf("%d of %d", start, end)
}
