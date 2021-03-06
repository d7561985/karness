package kube

import (
	"context"
	"fmt"
	"time"

	"github.com/d7561985/karness/pkg/controllers/harness"
	"github.com/d7561985/karness/pkg/worker"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	api "github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/generated/clientset/versioned"
	"github.com/d7561985/karness/pkg/generated/clientset/versioned/scheme"
	"github.com/d7561985/karness/pkg/generated/informers/externalversions/karness/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
)

const controllerAgentName = "karness-controllers"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Foo"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Scenario synced successfully"
)

type service struct {
	// sampleclientset is a clientset for our own API group
	appClientSet versioned.Interface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	scenarioInformer v1alpha1.ScenarioInformer
	scenarioSynced   cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface

	harness harness.Harness
}

func New(sClient versioned.Interface, sInformer v1alpha1.ScenarioInformer) *service {
	// Create event broadcaster
	// Add sample-controllers types to the default Kubernetes Scheme so Events can be
	// logged for sample-controllers types.
	utilruntime.Must(scheme.AddToScheme(kscheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	//eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: k.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(kscheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	klog.Info("Setting up event handlers")

	x := &service{
		appClientSet:     sClient,
		recorder:         recorder,
		scenarioInformer: sInformer,
		scenarioSynced:   sInformer.Informer().HasSynced,
		workqueue:        worker.New(workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Scenarios")),
	}

	x.scenarioInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: x.enqueue,
		UpdateFunc: func(oldObj, newObj interface{}) {
			x.delete(oldObj)
			x.enqueue(newObj)
		},
		DeleteFunc: x.delete,
	})

	return x
}

// enqueueScenario takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *service) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

// delete object from processing
func (c *service) delete(obj interface{}) {
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.harness.Stop(key)
	klog.Infof("delete object: %s", key)

	v := obj.(*api.Scenario)
	c.recorder.Event(v, corev1.EventTypeNormal, "Deleted", "stop processing")
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *service) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Foo controllers")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.scenarioSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *service) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *service) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Scenario resource to be synced.
		if err := c.syncHandler(ctx, key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)

		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *service) syncHandler(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	asset, err := c.scenarioInformer.Lister().Scenarios(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("asset '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// factory process object and call controller functions to show progress
	return c.harness.Factory(ctx, c, key, asset)
}

func (c *service) Update(item *api.Scenario) error {
	fmt.Println("STATUS", item.Status)

	// Finally, we update the status block of the Foo resource to reflect the
	// current state of the world
	err := c.updateScenarioStatus(item)
	if err != nil {
		return err
	}

	c.recorder.Event(item, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *service) updateScenarioStatus(item *api.Scenario) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	fooCopy := item.DeepCopy()

	// If the CustomResourceSubresources feature gate is not enabled,
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Foo resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.

	_, err := c.appClientSet.KarnessV1alpha1().Scenarios(item.Namespace).UpdateStatus(context.TODO(), fooCopy, metav1.UpdateOptions{})
	return err
}
