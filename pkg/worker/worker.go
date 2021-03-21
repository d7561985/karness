package worker

import (
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Instance is a rate limited work queue. This is used to queue work to be
// processed instead of performing it as soon as a change happens. This
// means we can ensure we only process a fixed amount of resources at a
// time, and makes it easy to ensure we are never processing the same item
// simultaneously in two different workers.
type Instance struct {
	workqueue.RateLimitingInterface
}

func New(w workqueue.RateLimitingInterface) *Instance {
	return &Instance{RateLimitingInterface: w}
}

// enqueueScenario takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (w *Instance) Enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	w.RateLimitingInterface.Add(key)
}
