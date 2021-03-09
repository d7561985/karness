package v1alpha1

import (
	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostAliasType is a top-level type
type ScenarioType struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Status ScenarioTypeStatus `json:"status,omitempty"`
	// This is where you can define
	// your own custom spec
	Spec ScenarioSpec `json:"spec,omitempty"`
}

// custom spec
type ScenarioSpec struct {
	Event []models.Event `json:"events"`
}

// custom status
type ScenarioTypeStatus struct {
	Name string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// no client needed for list as it's been created in above
type ScenarioTypeList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `son:"metadata,omitempty"`

	Items []ScenarioType `json:"items"`
}
