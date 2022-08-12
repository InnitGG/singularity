package v1

import (
	"context"
	"innit.gg/singularity/pkg/apis"
	"innit.gg/singularity/pkg/apis/singularity"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// FleetNameLabel is the name of Fleet which owns resources like GameServerSet and GameServer
	FleetNameLabel = singularity.GroupName + "/fleet"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.labelSelector
//+kubebuilder:printcolumn:name="Scheduling",type=string,JSONPath=`.spec.scheduling`
//+kubebuilder:printcolumn:name="Desired",type=string,JSONPath=`.spec.replicas`
//+kubebuilder:printcolumn:name="Current",type=string,JSONPath=`.status.replicas`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
	Replicas   int32                     `json:"replicas"`
	Strategy   appsv1.DeploymentStrategy `json:"strategy"`
	Scheduling apis.SchedulingStrategy   `json:"scheduling"`
	Template   GameServerTemplate        `json:"template"`
}

// FleetStatus defines the observed state of Fleet
type FleetStatus struct {
	Replicas           int32 `json:"replicas"`
	ReadyReplicas      int32 `json:"readyReplicas"`
	AllocatedReplicas  int32 `json:"allocatedReplicas"`
	Instances          int32 `json:"instances"`
	ReadyInstances     int32 `json:"readyInstances"`
	AllocatedInstances int32 `json:"allocatedInstances"`
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
	// Also, reset the ObjectMeta.
	gsSet.ObjectMeta.GenerateName = f.ObjectMeta.Name + "-"
	gsSet.ObjectMeta.Name = ""
	gsSet.ObjectMeta.Namespace = f.ObjectMeta.Namespace
	gsSet.ObjectMeta.ResourceVersion = ""
	gsSet.ObjectMeta.UID = ""

	ref := metav1.NewControllerRef(f, GroupVersion.WithKind("Fleet"))
	gsSet.ObjectMeta.OwnerReferences = append(gsSet.ObjectMeta.OwnerReferences, *ref)

	// Append Fleet name
	if gsSet.ObjectMeta.Labels == nil {
		gsSet.ObjectMeta.Labels = make(map[string]string, 1)
	}

	gsSet.ObjectMeta.Labels[FleetNameLabel] = f.ObjectMeta.Name

	return gsSet
}

// ListGameServerSet lists all owned GameServerSet
func (f *Fleet) ListGameServerSet(ctx context.Context, c client.Client) ([]*GameServerSet, error) {
	list := &GameServerSetList{}
	labelSelector := client.MatchingLabels{
		FleetNameLabel: f.ObjectMeta.Name,
	}
	if err := c.List(ctx, list, labelSelector); err != nil {
		return []*GameServerSet{}, err
	}

	// Make sure that the Fleet actually owns it
	var result []*GameServerSet
	for _, gsSet := range list.Items {
		if metav1.IsControlledBy(&gsSet, f) {
			result = append(result, &gsSet)
		}
	}

	return result, nil
}

// CountStatusReadyReplicas returns the count of GameServer with GameServerStateReady in a list of GameServerSet
func CountStatusReadyReplicas(list []*GameServerSet) int32 {
	total := int32(0)
	for _, gsSet := range list {
		if gsSet != nil {
			total += gsSet.Status.ReadyReplicas
		}
	}

	return total
}

// CountStatusAllocatedReplicas returns the count of GameServer with GameServerStateAllocated in a list of GameServerSet
func CountStatusAllocatedReplicas(list []*GameServerSet) int32 {
	total := int32(0)
	for _, gsSet := range list {
		if gsSet != nil {
			total += gsSet.Status.AllocatedReplicas
		}
	}

	return total
}

func CountStatusReplicas(list []*GameServerSet) int32 {
	total := int32(0)
	for _, gsSet := range list {
		if gsSet != nil {
			total += gsSet.Status.Replicas
		}
	}

	return total
}

func CountSpecReplicas(list []*GameServerSet) int32 {
	total := int32(0)
	for _, gsSet := range list {
		if gsSet != nil {
			total += gsSet.Spec.Replicas
		}
	}

	return total
}

// UpperBoundReplicas returns whichever is smaller, the value i, or the Fleet's desired replicas.
func (f *Fleet) UpperBoundReplicas(i int32) int32 {
	if i > f.Spec.Replicas {
		return f.Spec.Replicas
	}
	return i
}

// LowerBoundReplicas returns 0 if parameter i is less than zero
func (f *Fleet) LowerBoundReplicas(i int32) int32 {
	if i < 0 {
		return 0
	}
	return i
}

func init() {
	SchemeBuilder.Register(&Fleet{}, &FleetList{})
}
