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

// HostAlias is a top-level type
type Scenario struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ScenarioStatus `json:"status"`
	// This is where you can define
	// your own custom spec
	Spec ScenarioSpec `json:"spec"`
}

// custom spec
type ScenarioSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Events []Event `json:"events"`
}

// custom status
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

	GRPC action.GRPC `json:"grpc"`
	HTTP action.HTTP `json:"http"`

	Body Body `json:"body"`
}

type Any string

type Body struct {
	KV     map[string]Any `json:"kv"`
	Byte   []byte         `json:"byte"`
	String *string        `json:"string"`
}

type Completion struct {
	Name      string      `json:"name"`
	Condition []Condition `json:"condition"`
}

// Condition of complete show reason
type Condition struct {
	// Source of condition check
	Source *ConditionSource `json:"source"`
}

// ConditionSource contains competition condition for source
type ConditionSource struct {
	KV *KV `json:"kv"`
}

type KV struct {
	Field []KVFieldMatch `json:"field_match"`
}

// KVFields mean that key sh
type KVFieldMatch struct {
	Key   string `json:"key"`
	Value Any    `json:"value"`
}
