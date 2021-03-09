package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostAliasType is a top-level type
type HostAliasType struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Status HostAliasTypeStatus `json:"status,omitempty"`
	// This is where you can define
	// your own custom spec
	Spec HostAliasSpec `json:"spec,omitempty"`
}

// custom spec
type HostAliasSpec struct {
	Host  string `json:"host,omitempty"`
	Alias string `json:"alias,omitempty"`
}

// custom status
type HostAliasTypeStatus struct {
	Name string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// no client needed for list as it's been created in above
type HostAliasTypeList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `son:"metadata,omitempty"`

	Items []HostAliasType `json:"items"`
}
