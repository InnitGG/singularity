package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// GameServerInstanceStateStarting indicates that the GameServer is starting
	GameServerInstanceStateStarting GameServerInstanceState = "Starting"
	// GameServerInstanceStateReady indicates that the GameServerInstance is ready to accept players
	GameServerInstanceStateReady GameServerInstanceState = "Ready"
	// GameServerInstanceStateAllocated indicates that the GameServerInstance is currently running a game
	GameServerInstanceStateAllocated GameServerInstanceState = "Allocated"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
	Map      string `json:"map"`
	Extra    string `json:"extra,omitempty"`
}

type GameServerInstanceState string

// GameServerInstanceStatus defines the observed state of GameServerInstance
type GameServerInstanceStatus struct {
	State GameServerInstanceState `json:"state"`
}

func init() {
	SchemeBuilder.Register(&GameServerInstance{}, &GameServerInstanceList{})
}
