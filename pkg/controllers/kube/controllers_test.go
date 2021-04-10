package kube

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1/models/action"
	"github.com/d7561985/karness/pkg/executor/grpcexec"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/tools/cache"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/d7561985/karness/pkg/generated/clientset/versioned/fake"
	informers "github.com/d7561985/karness/pkg/generated/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	core "k8s.io/client-go/testing"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	client *fake.Clientset

	// Objects to put in the store.
	scenarioList []*v1alpha1.Scenario

	// Actions expected to happen on the client.
	actions []core.Action

	// Objects from here preloaded into NewSimpleFake.
	objects []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	return f
}

func newEvent(name string, action v1alpha1.Action, conditions ...v1alpha1.Condition) v1alpha1.Event {
	return v1alpha1.Event{
		Name:   "event-" + name,
		Action: action,
		Complete: v1alpha1.Completion{
			Name:      "complete-" + name,
			Condition: conditions,
		},
	}
}

func newScenario(
	name string, state v1alpha1.State, progres string, vars map[string]v1alpha1.Any, events ...v1alpha1.Event) *v1alpha1.Scenario {
	return &v1alpha1.Scenario{
		TypeMeta: v1.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.ScenarioSpec{
			Events:    events,
			Variables: vars,
		},
		Status: v1alpha1.ScenarioStatus{
			Progress: progres,
			State:    state,
		},
	}
}

func (f *fixture) newController() (*service, informers.SharedInformerFactory) {
	f.client = fake.NewSimpleClientset(f.objects...)

	i := informers.NewSharedInformerFactory(f.client, noResyncPeriodFunc())
	c := New(f.client, i.Karness().V1alpha1().Scenarios())
	c.scenarioSynced = alwaysReady

	for _, scenario := range f.scenarioList {
		_ = i.Karness().V1alpha1().Scenarios().Informer().GetIndexer().Add(scenario)
	}

	return c, i
}

func (f *fixture) run(scenarioName string, steps int) {
	f.runController(scenarioName, true, steps, false)
}

func (f *fixture) runController(key string, startInformers bool, steps int, expectError bool) {
	c, i := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		i.Start(stopCh)
	}

	// skip worker process
	err := c.syncHandler(context.Background(), key)
	if !expectError && err != nil {
		f.t.Errorf("error syncing foo: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing foo, got nil")
	}

	if steps > 0 {
		s, ok := c.harness.GetProcessor(key)
		if !ok {
			f.t.Errorf("processor can't extract")
			return
		}

		for i := 0; i < steps; i++ {
			s.(interface {
				Step(ctx context.Context) bool
			}).Step(context.Background())
		}
	}

	actions := filterInformerActions(f.client.Actions())
	for i, a := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, a, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}

	for _, act := range actions {
		if len(act.GetNamespace()) == 0 &&
			(act.Matches("list", "scenarios") ||
				act.Matches("watch", "scenarios") ||
				act.Matches("list", "deployments") ||
				act.Matches("watch", "deployments")) {
			continue
		}

		ret = append(ret, act)
	}

	return ret
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

func (f *fixture) expectUpdateFooStatusAction(items ...*v1alpha1.Scenario) {
	for _, item := range items {
		a := core.NewUpdateAction(schema.GroupVersionResource{Resource: "scenarios"}, item.Namespace, item)
		// TODO: Until #38113 is merged, we can't use Subresource
		a.Subresource = "status"

		f.actions = append(f.actions, a)
	}
}

func getKey(foo *v1alpha1.Scenario, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(foo)
	if err != nil {
		t.Errorf("Unexpected error getting key for scenario %v: %v", foo.Name, err)
		return ""
	}

	return key
}

func TestDoNothing(t *testing.T) {
	f := newFixture(t)

	// scenario which exist in store right now
	scena := newScenario("test", "", "", nil)
	f.scenarioList = append(f.scenarioList, scena)
	f.objects = append(f.objects, scena)

	// create update action + desired scena update
	f.expectUpdateFooStatusAction(
		newScenario("test", v1alpha1.Ready, "0 of 0", nil),
	)

	f.run(getKey(scena, t), 0)
}

func TestDoNothingStartProcessor(t *testing.T) {
	f := newFixture(t)

	e := newEvent("grpc",
		v1alpha1.Action{
			Body: v1alpha1.Body{
				KV: map[string]v1alpha1.Any{
					"X": "Q",
				},
			},
		},
		v1alpha1.Condition{
			Response: &v1alpha1.ConditionResponse{
				Body: v1alpha1.Body{
					//KV: map[string]v1alpha1.Any{
					//	"X": "Y",
					//},
				},
			},
		},
	)

	// scenario which exist in store right now
	scena := newScenario("test", "", "", nil, e)
	f.scenarioList = append(f.scenarioList, scena)
	f.objects = append(f.objects, scena)

	// create update action + desired scena update
	f.expectUpdateFooStatusAction(
		newScenario("test", v1alpha1.Ready, "0 of 1", nil, e),
		newScenario("test", v1alpha1.Complete, "1 of 1", nil, e),
	)

	f.run(getKey(scena, t), 1)
}

func TestGRPCCall(t *testing.T) {
	const responseMSG = "OK"
	var expect = `{
  "message": "OK"
}
`
	l, srv := grpcexec.CreateMockServer(grpcexec.Fixture{
		Res: &pb.HelloReply{Message: responseMSG},
		CB:  func(req *pb.HelloRequest) {},
	})

	defer l.Close()
	defer srv.Stop()

	f := newFixture(t)

	e := newEvent("grpc",
		v1alpha1.Action{
			Name: "Grpc-Test",
			GRPC: &action.GRPC{
				Addr:    l.Addr().String(),
				Package: "helloworld",
				Service: "Greeter",
				RPC:     "SayHello",
			},

			Body: v1alpha1.Body{
				KV: map[string]v1alpha1.Any{
					"name": "hello",
				},
			},
			BindResult: map[string]string{"MSG": `{.message}`},
		},
		v1alpha1.Condition{
			Response: &v1alpha1.ConditionResponse{
				Status: codes.OK.String(),
				Body: v1alpha1.Body{
					KV: map[string]v1alpha1.Any{
						"message": responseMSG,
					},
					JSON: &expect,
					Byte: []byte(expect),
				},
			},
		},
	)

	scena := newScenario("test", "", "", nil, e)
	f.scenarioList = append(f.scenarioList, scena)
	f.objects = append(f.objects, scena)

	// create update action + desired scena update
	f.expectUpdateFooStatusAction(
		newScenario("test", v1alpha1.Ready, "0 of 1", nil, e),
		newScenario("test", v1alpha1.Complete, "1 of 1", nil, e),
	)

	f.run(getKey(scena, t), 1)
}
