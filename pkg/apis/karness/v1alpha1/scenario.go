package v1alpha1

import (
	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1/models/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type State string

const (
	Ready      State = "READY"
	InProgress State = "IN_PROGRESS"
	Complete   State = "COMPLETE"
	Failed     State = "FAILED"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Scenario HostAlias is a top-level type
type Scenario struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ScenarioStatus `json:"status"`
	// This is where you can define
	// your own custom spec
	Spec ScenarioSpec `json:"spec"`
}

// ScenarioSpec custom spec
type ScenarioSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Events    []Event        `json:"events"`
	Variables map[string]Any `json:"variables"`
}

// ScenarioStatus custom status
type ScenarioStatus struct {
	Progress string `json:"progress"`
	State    State  `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// no client needed for list as it's been created in above
type ScenarioList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `son:"metadata,omitempty"`

	Items []Scenario `json:"items"`
}

//
type Event struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Action   Action     `json:"action"`
	Complete Completion `json:"complete"`
}

type Action struct {
	Name string `json:"name"`

	GRPC *action.GRPC `json:"grpc"`
	HTTP *action.HTTP `json:"http"`

	Body Body `json:"body"`

	// BindResult save result KV representation in global variable storage
	// This works only when result returns as JSON or maybe anything marshalable
	// Right now only JSON supposed to be
	// Key: result_key
	// Val: variable name for binding
	BindResult map[string]string `json:"bind_result"`
}

type Any string

type Body struct {
	KV   map[string]Any `json:"kv"`
	Byte []byte         `json:"byte"`
	JSON *string        `json:"json"`
}

type Completion struct {
	Name      string      `json:"name"`
	Condition []Condition `json:"condition"`
}

// Condition of complete show reason
type Condition struct {
	// Response of condition check
	Response *ConditionResponse `json:"response"`
}

// ConditionResponse contains competition condition for source
type ConditionResponse struct {
	Status string `json:"status"`
	Body   Body   `json:"body"`
}

type KV struct {
	Field []KVFieldMatch `json:"field_match"`
}

// KVFields mean that key sh
type KVFieldMatch struct {
	Key   string `json:"key"`
	Value Any    `json:"value"`
}
