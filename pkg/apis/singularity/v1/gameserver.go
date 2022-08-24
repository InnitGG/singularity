package v1

import (
	"innit.gg/singularity/pkg/apis"
	"innit.gg/singularity/pkg/apis/singularity"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
)

const (
	// GameServerTypeGame describes a game server which utilizes the allocation system
	GameServerTypeGame GameServerType = "Game"
	// GameServerTypeEphemeral describes a game server which is stateless
	GameServerTypeEphemeral GameServerType = "Ephemeral"
	// GameServerTypeStatic describes a game server which is manually controlled by the user
	GameServerTypeStatic GameServerType = "Static"

	// GameServerStateCreating indicates that the Pod is not yet created
	GameServerStateCreating GameServerState = "Creating"
	// GameServerStateStarting indicates that the Pod is created, but not yet scheduled
	GameServerStateStarting GameServerState = "Starting"
	// GameServerStateScheduled indicates that the Pod is scheduled in the cluster, basically belonging to a Node
	GameServerStateScheduled GameServerState = "Scheduled"
	// GameServerStateReady indicates that the server is ready to accept player (and optionally Allocated)
	GameServerStateReady GameServerState = "Ready"
	// GameServerStateAllocated indicates that the server has been allocated and shall not be removed
	GameServerStateAllocated GameServerState = "Allocated"
	// GameServerStateDrain indicates the server is no longer accepting new players, and is waiting for existing
	// instances to be shut down.
	GameServerStateDrain GameServerState = "Drain"
	// GameServerStateShutdown indicates that the server has shutdown and everything has to be removed from the cluster
	GameServerStateShutdown GameServerState = "Shutdown"
	// GameServerStateError indicates that something irrecoverable occurred
	GameServerStateError GameServerState = "Error"
	// GameServerStateUnhealthy indicates that the server failed its health checks
	GameServerStateUnhealthy GameServerState = "Unhealthy"

	// GameServerRole is the GameServer label value for singularity.RoleLabel
	GameServerRole = "gameserver"
	// GameServerNameLabel is the name of GameServer which owns resources like v1.Pod
	GameServerNameLabel = singularity.GroupName + "/fleet"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Desired",type=string,JSONPath=`.spec.instances`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
	Type             GameServerType             `json:"type"`
	Scheduling       apis.SchedulingStrategy    `json:"scheduling"`
	DrainStrategy    GameServerDrainStrategy    `json:"drainStrategy"`
	Ports            []GameServerPort           `json:"ports"`
	Instances        int32                      `json:"instances"`
	InstanceTemplate GameServerInstanceTemplate `json:"instanceTemplate"`
	Template         v1.PodTemplateSpec         `json:"template"`
}

type GameServerType string
type GameServerState string

// GameServerStatus defines the observed state of GameServer
type GameServerStatus struct {
	State GameServerState `json:"state"`
}

type GameServerDrainStrategy struct {
	Timeout            int32 `json:"timeout,omitempty"`
	Instances          int32 `json:"instances"`
	ReadyInstances     int32 `json:"readyInstances"`
	AllocatedInstances int32 `json:"allocatedInstances"`
}

type GameServerPort struct {
	Name          string          `json:"name"`
	PortPolicy    apis.PortPolicy `json:"portPolicy"`
	ContainerPort string          `json:"containerPort"`
}

// IsDeletable returns whether the server is currently allocated/reserved and is not already in the
// process of being deleted
func (gs *GameServer) IsDeletable() bool {
	if gs.Status.State == GameServerStateAllocated {
		return !gs.ObjectMeta.DeletionTimestamp.IsZero()
	}

	return true
}

// IsBeingDeleted returns true if the server is in the process of being deleted.
func (gs *GameServer) IsBeingDeleted() bool {
	return !gs.ObjectMeta.DeletionTimestamp.IsZero() || gs.Status.State == GameServerStateShutdown
}

// Pod creates a Pod according to the template specified in the GameServer resource
func (gs *GameServer) Pod() *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: *gs.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gs.Spec.Template.Spec.DeepCopy(),
	}

	gs.configurePodMeta(pod)

	// TODO: hostPort allocation

	return pod
}

func (gs *GameServer) ServiceAccount() *v1.ServiceAccount {
	ref := metav1.NewControllerRef(gs, GroupVersion.WithKind("GameServer"))

	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gs.Name,
			Namespace: gs.Namespace,
			Labels: map[string]string{
				GameServerNameLabel: gs.ObjectMeta.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
	}
}

func (gs *GameServer) Role() *rbacv1.Role {
	ref := metav1.NewControllerRef(gs, GroupVersion.WithKind("GameServer"))

	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gs.Name,
			Namespace: gs.Namespace,
			Labels: map[string]string{
				GameServerNameLabel: gs.ObjectMeta.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
		// Only allow access to its own GameServer and Pod resources
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:           []string{"get", "update", "list", "watch"},
				APIGroups:       []string{GroupVersion.String()},
				Resources:       []string{"gameservers", "pods"},
				ResourceNames:   []string{gs.ObjectMeta.Name},
				NonResourceURLs: nil,
			},
		},
	}
}

func (gs *GameServer) RoleBinding() *rbacv1.RoleBinding {
	ref := metav1.NewControllerRef(gs, GroupVersion.WithKind("GameServer"))

	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gs.ObjectMeta.Name,
			Namespace: gs.ObjectMeta.Namespace,
			Labels: map[string]string{
				GameServerNameLabel: gs.ObjectMeta.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: gs.ObjectMeta.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: gs.ObjectMeta.Name,
		},
	}
}

// SortDescending returns GameServers sorted by newest created
func SortDescending(list []*GameServer) []*GameServer {
	sort.Slice(list, func(i, j int) bool {
		a := list[i]
		b := list[j]

		return a.ObjectMeta.CreationTimestamp.Before(&b.ObjectMeta.CreationTimestamp)
	})

	return list
}

func (gs *GameServer) configurePodMeta(pod *v1.Pod) {
	// Name and namespace needs to match the GameServer
	pod.ObjectMeta.GenerateName = ""
	pod.ObjectMeta.Name = gs.ObjectMeta.Name
	pod.ObjectMeta.Namespace = gs.ObjectMeta.Namespace

	// Make sure that the ServiceAccount is bound
	pod.Spec.ServiceAccountName = gs.ObjectMeta.Name

	// Reset these, just in case
	pod.ObjectMeta.ResourceVersion = ""
	pod.ObjectMeta.UID = ""

	// Append labels
	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = make(map[string]string, 2)
	}
	pod.ObjectMeta.Labels[singularity.RoleLabel] = GameServerRole
	pod.ObjectMeta.Labels[GameServerNameLabel] = gs.ObjectMeta.Name

	// Append GameServer owner reference
	ref := metav1.NewControllerRef(gs, GroupVersion.WithKind("GameServer"))
	pod.ObjectMeta.OwnerReferences = append(pod.ObjectMeta.OwnerReferences, *ref)
}

func init() {
	SchemeBuilder.Register(&GameServer{}, &GameServerList{})
}
