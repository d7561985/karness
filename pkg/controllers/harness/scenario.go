package harness

import (
	"context"
	"fmt"
	"github.com/d7561985/karness/pkg/controllers/harness/models"
	"sync"
	"time"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/controllers"
	"github.com/d7561985/karness/pkg/controllers/harness/checker"
	"k8s.io/klog/v2"
)

type scenarioProcessor struct {
	entity  *v1alpha1.Scenario
	control controllers.Kube
	store   sync.Map
	// only complete function is possible to increment current check
	current int
}

func newScenarioProcessor(c controllers.Kube, item *v1alpha1.Scenario) Processor {
	item.Status.Progress = sFmt(0, len(item.Spec.Events))
	item.Status.State = v1alpha1.Ready

	p := &scenarioProcessor{control: c, entity: item}

	for k, v := range item.Spec.Variables {
		p.store.Store(k, v)
	}

	return p
}

// Start ...
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

	if err := s.process(ctx, s.entity.Spec.Events[s.current]); err != nil {
		klog.Errorf("process error: %s", err.Error())
		s.entity.Status.State = v1alpha1.Failed
		s.entity.Status.Message = err.Error()
		// exit on fail
		return true
	}

	if len(ev) <= s.current {
		s.entity.Status.State = v1alpha1.Complete
		return true
	}

	return false
}

func (s *scenarioProcessor) process(ctx context.Context, event v1alpha1.Event) error {
	res, err := s.action(ctx, models.NewAction(event.Action))
	if err != nil {
		return fmt.Errorf("action %w", err)
	}

	return s.checkComplete(event.Complete.Condition, res)
}

func (s *scenarioProcessor) action(ctx context.Context, a *models.Action) (res *ActionResult, err error) {
	res = OK()

	resp, err := a.GetBody(&s.store)
	if err != nil {
		return nil, err
	}

	if a.GRPC != nil {
		res, err = NewGRPC(a).Call(ctx, resp)
		if err != nil {
			klog.Errorf("scenario progress with action %q grpc call error %w", a.Name, err)
			// ok=true:  we want to try again
			return nil, err
		}
	}

	for variable, jpath := range a.BindResult {
		val, err := res.GetKeyValue(jpath)
		if err != nil {
			return nil, fmt.Errorf("binding result key %s err %w", variable, err)
		}

		s.store.Store(variable, val)
	}

	return res, nil
}

func (s *scenarioProcessor) checkComplete(c []v1alpha1.Condition, result *ActionResult) error {
	for _, condition := range c {
		if condition.Response != nil {
			if err := checker.ResCheck(&s.store, condition.Response).
				Is(result.Code, result.Body); err != nil {
				return err
			}
		}
	}

	s.current++
	return nil
}

func sFmt(start, end int) string {
	return fmt.Sprintf("%d of %d", start, end)
}
