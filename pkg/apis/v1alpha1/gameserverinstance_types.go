package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GameServerInstance is the Schema for the GameServerInstances API
type GameServerInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerInstanceSpec   `json:"spec,omitempty"`
	Status GameServerInstanceStatus `json:"status,omitempty"`
}

// GameServerInstanceTemplate is the template for the GameServerInstances API
type GameServerInstanceTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GameServerInstanceSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// GameServerInstanceList contains a list of GameServerInstance
type GameServerInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServerInstance `json:"items"`
}

// GameServerInstanceSpec defines the desired state of GameServerInstance
type GameServerInstanceSpec struct {
	Capacity uint32 `json:"capacity"`
	// map
}

// GameServerInstanceStatus defines the observed state of GameServerInstance
type GameServerInstanceStatus struct {
	// +kubebuilder:validation:Enum=NotReady;Ready;Allocated
	State string `json:"state"`
}

func init() {
	SchemeBuilder.Register(&GameServerInstance{}, &GameServerInstanceList{})
}
