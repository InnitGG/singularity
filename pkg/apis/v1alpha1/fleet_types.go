package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FleetSpec defines the desired state of Fleet
type FleetSpec struct {
	Replicas uint32 `json:"replicas"`

	// +kubebuilder:validation:Enum=Packed;Distributed
	Scheduling string `json:"scheduling"`
}

// FleetStatus defines the observed state of Fleet
type FleetStatus struct {
	Replicas      uint32 `json:"replicas"`
	ReadyReplicas uint32 `json:"readyReplicas"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.readyReplicas

// Fleet is the Schema for the fleets API
type Fleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetSpec   `json:"spec,omitempty"`
	Status FleetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FleetList contains a list of Fleet
type FleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fleet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Fleet{}, &FleetList{})
}
