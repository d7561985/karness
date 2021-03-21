package harness

import (
	"context"
	"fmt"
	"time"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/controllers"
	"k8s.io/klog/v2"
)

type scenarioProcessor struct {
	entity  *v1alpha1.Scenario
	control controllers.Kube

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
			if s.Step() {
				return
			}
		}
	}
}

func (s *scenarioProcessor) Step() bool {
	s.entity.Status.State = v1alpha1.InProgress
	ev := s.entity.Spec.Events

	defer func() {
		s.entity.Status.Progress = sFmt(s.current, len(ev))

		if err := s.control.Update(s.entity); err != nil {
			klog.Errorf("scenario processor: %w", err)
		}
	}()

	if !s.process(s.entity.Spec.Events[s.current]) {
		s.entity.Status.State = v1alpha1.Failed
		return true
	}

	s.current++

	if len(ev) <= s.current {
		s.entity.Status.State = v1alpha1.Complete
		return true
	}

	return false
}

func (s *scenarioProcessor) process(event v1alpha1.Event) bool {
	return true
}

func sFmt(start, end int) string {
	return fmt.Sprintf("%d of %d", start, end)
}
