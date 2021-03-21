package harness

import (
	"context"
	"fmt"
	"sync"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/controllers"
)

type Harness struct {
	// key: object
	// value: context.CancelFunc
	cancels sync.Map

	store sync.Map
}

func (h *Harness) Factory(root context.Context, c controllers.Kube, key string, obj interface{}) error {
	ctx, cancel := context.WithCancel(root)
	h.cancels.Store(key, cancel)

	switch item := obj.(type) {
	case *v1alpha1.Scenario:
		p := newScenarioProcessor(c, item)
		h.store.Store(key, p)

		go p.Start(ctx)

		return c.Update(item)
	default:
		panic(fmt.Errorf("upredictable object: %v[%[1]T]", item))
	}

	return nil
}

func (h *Harness) GetProcessor(key string) (interface{}, bool) {
	return h.store.Load(key)
}

func (h *Harness) Stop(key string) {
	obj, ok := h.cancels.Load(key)
	if !ok {
		return
	}

	obj.(context.CancelFunc)()
}
