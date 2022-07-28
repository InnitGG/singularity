package v1alpha1

import (
	"innit.gg/singularity/pkg/apis"
	"innit.gg/singularity/pkg/apis/singularity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// FleetNameLabel is the name of Fleet which owns resources like GameServerSet and GameServer
	FleetNameLabel = singularity.GroupName + "/fleet"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.readyReplicas

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

// FleetSpec defines the desired state of Fleet
type FleetSpec struct {
	Replicas   uint32                  `json:"replicas"`
	Scheduling apis.SchedulingStrategy `json:"scheduling"`
	Template   GameServerTemplate      `json:"template"`
}

// FleetStatus defines the observed state of Fleet
type FleetStatus struct {
	Replicas      uint32 `json:"replicas"`
	ReadyReplicas uint32 `json:"readyReplicas"`
}

// GameServerSet returns a single GameServerSet for this Fleet definition
func (f *Fleet) GameServerSet() *GameServerSet {
	gsSet := &GameServerSet{
		ObjectMeta: *f.Spec.Template.ObjectMeta.DeepCopy(),
		Spec: GameServerSetSpec{
			Template:   f.Spec.Template,
			Scheduling: f.Spec.Scheduling,
		},
	}

	// Generate a unique name for GameServerSet, ensuring there are no collisions.
	gsSet.ObjectMeta.GenerateName = f.ObjectMeta.Name + "-"
	gsSet.ObjectMeta.Namespace = f.ObjectMeta.Namespace

	ref := metav1.NewControllerRef(f, GroupVersion.WithKind("Fleet"))
	gsSet.ObjectMeta.OwnerReferences = append(gsSet.ObjectMeta.OwnerReferences, *ref)

	// Append Fleet name
	if gsSet.ObjectMeta.Labels == nil {
		gsSet.ObjectMeta.Labels = make(map[string]string, 1)
	}

	gsSet.ObjectMeta.Labels[FleetNameLabel] = f.ObjectMeta.Name

	return gsSet
}

func init() {
	SchemeBuilder.Register(&Fleet{}, &FleetList{})
}
