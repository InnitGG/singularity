package v1

import (
	"context"
	"innit.gg/singularity/pkg/apis"
	"innit.gg/singularity/pkg/apis/singularity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// GameServerSetNameLabel is the name of GameServerSet which owns resources like GameServer
	GameServerSetNameLabel = singularity.GroupName + "/gameserverset"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Scheduling",type=string,JSONPath=`.spec.scheduling`
//+kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.spec.replicas`
//+kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.replicas`
//+kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
//+kubebuilder:printcolumn:name="Allocated",type=integer,JSONPath=`.status.allocatedReplicas`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
	Replicas   int32                   `json:"replicas"`
	Scheduling apis.SchedulingStrategy `json:"scheduling"`
	Template   GameServerTemplate      `json:"template"`
}

// GameServerSetStatus defines the observed state of GameServerSet
type GameServerSetStatus struct {
	Replicas           int32 `json:"replicas"`
	ReadyReplicas      int32 `json:"readyReplicas"`
	AllocatedReplicas  int32 `json:"allocatedReplicas"`
	ShutdownReplicas   int32 `json:"shutdownReplicas"`
	Instances          int32 `json:"instances"`
	ReadyInstances     int32 `json:"readyInstances"`
	AllocatedInstances int32 `json:"allocatedInstances"`
	ShutdownInstances  int32 `json:"shutdownInstances"`
}

// GameServer returns a single GameServer for this GameServerSet specification
func (gsSet *GameServerSet) GameServer() *GameServer {
	gs := &GameServer{
		ObjectMeta: *gsSet.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gsSet.Spec.Template.Spec.DeepCopy(),
	}

	gs.Spec.Scheduling = gsSet.Spec.Scheduling

	// Generate a unique name for GameServerSet, ensuring there are no collisions.
	// Also, reset the ObjectMeta.
	gs.ObjectMeta.GenerateName = gsSet.ObjectMeta.Name + "-"
	gs.ObjectMeta.Name = ""
	gs.ObjectMeta.Namespace = gsSet.ObjectMeta.Namespace
	gs.ObjectMeta.ResourceVersion = ""
	gs.ObjectMeta.UID = ""

	ref := metav1.NewControllerRef(gsSet, GroupVersion.WithKind("GameServerSet"))
	gs.ObjectMeta.OwnerReferences = append(gs.ObjectMeta.OwnerReferences, *ref)

	// Append Fleet name and GameServerSet name labels
	if gs.ObjectMeta.Labels == nil {
		gs.ObjectMeta.Labels = make(map[string]string, 2)
	}

	gs.ObjectMeta.Labels[FleetNameLabel] = gsSet.ObjectMeta.Labels[FleetNameLabel]
	gs.ObjectMeta.Labels[GameServerSetNameLabel] = gsSet.ObjectMeta.Name
	return gs
}

// ListGameServer lists all owned GameServer
func (gsSet *GameServerSet) ListGameServer(ctx context.Context, c client.Client) ([]*GameServer, error) {
	list := &GameServerList{}
	labelSelector := client.MatchingLabels{
		GameServerSetNameLabel: gsSet.ObjectMeta.Name,
	}
	if err := c.List(ctx, list, labelSelector); err != nil {
		return []*GameServer{}, err
	}

	// Make sure that the Fleet actually owns it
	var result []*GameServer
	for i := range list.Items {
		gs := &list.Items[i]
		if metav1.IsControlledBy(gs, gsSet) {
			result = append(result, gs)
		}
	}

	return result, nil
}

func init() {
	SchemeBuilder.Register(&GameServerSet{}, &GameServerSetList{})
}
