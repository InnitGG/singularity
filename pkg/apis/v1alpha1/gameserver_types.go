package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GameServer is the Schema for the GameServers API
type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec,omitempty"`
	Status GameServerStatus `json:"status,omitempty"`
}

// GameServerTemplate is the template for the GameServers API
type GameServerTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GameServerSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// GameServerList contains a list of GameServer
type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServer `json:"items"`
}

// GameServerSpec defines the desired state of GameServer
type GameServerSpec struct {
	// +kubebuilder:validation:Enum=Game;Ephemeral;Static
	Type             string                     `json:"type"`
	DrainStrategy    GameServerDrainStrategy    `json:"drainStrategy"`
	Ports            []GameServerPort           `json:"ports"`
	Instances        uint32                     `json:"instances"`
	InstanceTemplate GameServerInstanceTemplate `json:"instanceTemplate"`
	Template         v1.PodTemplateSpec         `json:"template"`
}

// GameServerStatus defines the observed state of GameServer
type GameServerStatus struct {
	// +kubebuilder:validation:Enum=NotReady;Ready;Allocated;Drain
	State string `json:"state"`
}

type GameServerDrainStrategy struct {
	Timeout uint32 `json:"timeout,omitempty"`
}

type GameServerPort struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=Internal;Dynamic
	PortPolicy    string `json:"portPolicy"`
	ContainerPort string `json:"containerPort"`
}

func init() {
	SchemeBuilder.Register(&GameServer{}, &GameServerList{})
}
