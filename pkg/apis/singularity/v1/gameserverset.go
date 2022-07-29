package v1

import (
	"innit.gg/singularity/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GameServerSet is the Schema for the GameServerSets API
type GameServerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSetSpec   `json:"spec,omitempty"`
	Status GameServerSetStatus `json:"status,omitempty"`
}

// GameServerSetTemplate is the template for the GameServerSets API
type GameServerSetTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GameServerSetSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// GameServerSetList contains a list of GameServerSet
type GameServerSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServerSet `json:"items"`
}

// GameServerSetSpec defines the desired state of GameServerSet
type GameServerSetSpec struct {
	Replicas   uint32                  `json:"replicas"`
	Scheduling apis.SchedulingStrategy `json:"scheduling"`
	Template   GameServerTemplate      `json:"template"`
}

// GameServerSetStatus defines the observed state of GameServerSet
type GameServerSetStatus struct {
	Replicas      uint32 `json:"replicas"`
	ReadyReplicas uint32 `json:"readyReplicas"`
}

// GameServer returns a single GameServer for this GameServerSet specification
func (gsSet *GameServerSet) GameServer() *GameServer {
	gs := &GameServer{
		ObjectMeta: *gsSet.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gsSet.Spec.Template.Spec.DeepCopy(),
	}

	gs.Spec.Scheduling = gsSet.Spec.Scheduling

	// Generate a unique name for GameServerSet, ensuring there are no collisions.
	gs.ObjectMeta.GenerateName = gsSet.ObjectMeta.Name + "-"
	gs.ObjectMeta.Namespace = gsSet.ObjectMeta.Namespace

	ref := metav1.NewControllerRef(gsSet, GroupVersion.WithKind("GameServerSet"))
	gs.ObjectMeta.OwnerReferences = append(gs.ObjectMeta.OwnerReferences, *ref)

	// Append Fleet name
	if gs.ObjectMeta.Labels == nil {
		gs.ObjectMeta.Labels = make(map[string]string, 1)
	}

	gs.ObjectMeta.Labels[FleetNameLabel] = gsSet.ObjectMeta.Labels[FleetNameLabel]
	return gs
}

func init() {
	SchemeBuilder.Register(&GameServerSet{}, &GameServerSetList{})
}
