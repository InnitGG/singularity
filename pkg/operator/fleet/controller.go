package fleet

import (
	"context"
	"github.com/pkg/errors"
	singularityv1 "innit.gg/singularity/pkg/apis/singularity/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// Reconciler reconciles a Fleet object
type Reconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=singularity.innit.gg,resources=fleets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=fleets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=fleets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Fleet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("reconcile", "req", req)

	// Retrieve the Fleet resource from the cluster, ignoring if it was deleted
	fleet := &singularityv1.Fleet{}
	if err := r.Get(ctx, req.NamespacedName, fleet); err != nil {
		l.Info("reconcile: resource deleted", "fleet", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Retrieve GameServerSets associated with this Fleet
	gsSetList := &singularityv1.GameServerSetList{}
	labelSelector := client.MatchingLabels{
		singularityv1.FleetNameLabel: req.Name,
	}
	if err := r.List(ctx, gsSetList, labelSelector); err != nil {
		l.Error(err, "reconcile: unable to list GameServerSet", "fleet", req.Name)

		// TODO: is this the correct way?
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 3 * time.Second,
		}, err
	}

	// Find the active GameServerSet and return the rest
	active, rest := r.filterActiveGameServerSet(fleet, gsSetList)
	if active == nil {
		l.Info("reconcile: creating GameServerSet", "fleet", req.Name)

		// If there isn't an active GameServerSet, create one.
		// However, don't apply it to the cluster yet.
		active = fleet.GameServerSet()
	}

	// Run the deployment cycle
	_, err := r.handleDeployment(ctx, fleet, active, rest)
	if err != nil {
		l.Error(err, "reconcile: deployment cycle failed", "fleet", req.Name)
		return ctrl.Result{}, err
	}

	// TODO: Delete empty GameServerSet
	// TODO: Insert the new (active) GameServerSet, if required
	// TODO: Update the active GameServerSet to match the desired replicas
	// TODO: Update Fleet status

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&singularityv1.Fleet{}).
		Complete(r)
}

// handleDeployment performs the deployment strategy
// https://github.com/googleforgames/agones/blob/8d01f2ce9c34ffadfdf22ab2fb3b1bafae7e6389/pkg/fleets/controller.go#L356
func (r *Reconciler) handleDeployment(ctx context.Context, fleet *singularityv1.Fleet, active *singularityv1.GameServerSet, rest []*singularityv1.GameServerSet) (int32, error) {
	if len(rest) == 0 {
		// There is only one GameServerSet which matches the desired state.
		// Further action is not required.
		return fleet.Spec.Replicas, nil
	}

	switch fleet.Spec.Strategy.Type {
	case appsv1.RollingUpdateDeploymentStrategyType:
		return r.handleRollingUpdateDeployment(ctx, fleet, active, rest)
	}

	return 0, errors.Errorf("unexpected deployment strategy type: %s", fleet.Spec.Strategy.Type)
}

func (r *Reconciler) filterActiveGameServerSet(fleet *singularityv1.Fleet, list *singularityv1.GameServerSetList) (*singularityv1.GameServerSet, []*singularityv1.GameServerSet) {
	var active *singularityv1.GameServerSet
	var rest []*singularityv1.GameServerSet

	for _, gsSet := range list.Items {
		// If the actual state is equal to the desired state
		if equality.Semantic.DeepEqual(gsSet.Spec.Template, fleet.Spec.Template) {
			active = &gsSet
		} else {
			rest = append(rest, &gsSet)
		}
	}

	return active, rest
}
