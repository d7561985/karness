package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"

	api "github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/generated/clientset/versioned"
	"github.com/d7561985/karness/pkg/generated/clientset/versioned/scheme"
	"github.com/d7561985/karness/pkg/generated/informers/externalversions/karness/v1alpha1"
)

const controllerAgentName = "karness-controller"

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
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// sampleclientset is a clientset for our own API group
	appClientSet versioned.Interface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	depInformer       appsinformers.DeploymentInformer
	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced

	scenarioInformer v1alpha1.ScenarioInformer
	scenarioSynced   cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
}

func New(
	k *kubernetes.Clientset,
	a *versioned.Clientset,
	d appsinformers.DeploymentInformer,
	s v1alpha1.ScenarioInformer) *service {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(scheme.AddToScheme(kscheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: k.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(kscheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	klog.Info("Setting up event handlers")

	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Foo resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	d.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			v := obj.(*appsv1.Deployment)
			fmt.Println("add >>>>", v.Name)
		},
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}

			fmt.Println("update >>>>1", newDepl.Name)
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Println("delete >>>>2")
		},
	})

	x := &service{
		kubeclientset:     k,
		appClientSet:      a,
		recorder:          recorder,
		depInformer:       d,
		deploymentsLister: d.Lister(),
		deploymentsSynced: d.Informer().HasSynced,
		scenarioInformer:  s,
		scenarioSynced:    s.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Scenarios"),
	}

	s.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: x.enqueueScenario,
		UpdateFunc: func(oldObj, newObj interface{}) {

		},
		DeleteFunc: func(obj interface{}) {

		},
	})

	return x
}

// enqueueScenario takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *service) enqueueScenario(obj interface{}) {
	v := obj.(*api.Scenario)
	fmt.Println("add >>>>", v.Name)

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *service) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Foo controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.scenarioSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *service) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *service) processNextWorkItem() bool {
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
		if err := c.syncHandler(key); err != nil {
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
func (c *service) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	foo, err := c.scenarioInformer.Lister().Scenarios(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	fmt.Println("STATUS", foo.Status)

	// Finally, we update the status block of the Foo resource to reflect the
	// current state of the world
	err = c.updateScenarioStatus(foo)
	if err != nil {
		return err
	}

	c.recorder.Event(foo, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *service) updateScenarioStatus(foo *api.Scenario) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	fooCopy := foo.DeepCopy()
	fooCopy.Status.Progress = fmt.Sprintf("%d of %d", 0, len(foo.Spec.Event))
	fooCopy.Status.State = "OK"
	// If the CustomResourceSubresources feature gate is not enabled,
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Foo resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.

	_, err := c.appClientSet.KarnessV1alpha1().Scenarios(foo.Namespace).UpdateStatus(context.TODO(), fooCopy, metav1.UpdateOptions{})
	return err
}
