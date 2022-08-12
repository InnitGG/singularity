package fleet

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	singularityv1 "innit.gg/singularity/pkg/apis/singularity/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// Reconciler reconciles a Fleet object
type Reconciler struct {
	client.Client
	Recorder record.EventRecorder
	Log      logr.Logger
}

//+kubebuilder:rbac:groups=singularity.innit.gg,resources=fleets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=fleets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=fleets/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: retry on error?
	l := log.FromContext(ctx)
	l.Info("reconcile")

	// Retrieve the Fleet resource from the cluster, ignoring if it was deleted
	fleet := &singularityv1.Fleet{}
	if err := r.Get(ctx, req.NamespacedName, fleet); err != nil {
		l.Info("reconcile: resource deleted", "fleet", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If Fleet is marked for deletion, don't do anything.
	if !fleet.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Retrieve GameServerSets associated with this Fleet
	list, err := fleet.ListGameServerSet(ctx, r.Client)
	if err != nil {
		l.Error(err, "reconcile: unable to list GameServerSet", "fleet", req.Name)

		// TODO: is this the correct way?
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 3 * time.Second,
		}, err
	}

	// Find the active GameServerSet and return the rest
	active, rest := r.filterActiveGameServerSet(fleet, list)
	if active == nil {
		l.Info("reconcile: creating GameServerSet", "fleet", req.Name)

		// If there isn't an active GameServerSet, create one.
		// However, don't apply it to the cluster yet.
		active = fleet.GameServerSet()
	}

	// Run the deployment cycle
	replicas, err := r.handleDeployment(ctx, fleet, active, rest)
	if err != nil {
		l.Error(err, "reconcile: deployment cycle failed", "fleet", req.Name)
		return ctrl.Result{}, err
	}

	if err = r.deleteEmptyGameServerSets(ctx, fleet, rest); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.upsertGameServerSet(ctx, fleet, active, replicas); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.updateStatus(ctx, fleet); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&singularityv1.Fleet{}).
		Owns(&singularityv1.GameServerSet{}).
		WithLogConstructor(func(req *reconcile.Request) logr.Logger {
			if req != nil {
				return r.Log.WithValues("req", req)
			}
			return r.Log
		}).
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

func (r *Reconciler) filterActiveGameServerSet(fleet *singularityv1.Fleet, list []*singularityv1.GameServerSet) (*singularityv1.GameServerSet, []*singularityv1.GameServerSet) {
	var active *singularityv1.GameServerSet
	var rest []*singularityv1.GameServerSet

	for _, gsSet := range list {
		// If the actual state is equal to the desired state
		if equality.Semantic.DeepEqual(gsSet.Spec.Template, fleet.Spec.Template) {
			active = gsSet
		} else {
			rest = append(rest, gsSet)
		}
	}

	return active, rest
}

// deleteEmptyGameServerSets deletes all GameServerSets with 0 replicas
func (r *Reconciler) deleteEmptyGameServerSets(ctx context.Context, fleet *singularityv1.Fleet, list []*singularityv1.GameServerSet) error {
	policy := client.PropagationPolicy(metav1.DeletePropagationBackground)
	for _, gsSet := range list {
		if gsSet.Status.Replicas == 0 && gsSet.Status.ShutdownReplicas == 0 {
			if err := r.Delete(ctx, gsSet, policy); err != nil {
				return errors.Wrapf(err, "error deleting gameserverset %s", gsSet.ObjectMeta.Name)
			}

			r.Recorder.Eventf(fleet, v1.EventTypeNormal, "DeletingGameServerSet", "Deleting inactive GameServerSet %s", gsSet.ObjectMeta.Name)
		}
	}

	return nil
}

// upsertGameServerSet inserts the new GameServerSet (if required)
// and updates the active GameServerSet to match the desired state
func (r *Reconciler) upsertGameServerSet(ctx context.Context, fleet *singularityv1.Fleet, active *singularityv1.GameServerSet, replicas int32) error {
	if active.UID == "" {
		active.Spec.Replicas = replicas
		if err := r.Create(ctx, active); err != nil {
			return errors.Wrapf(err, "error creating gameserverset %s", active.ObjectMeta.Name)
		}

		gsSetCopy := active.DeepCopy()
		gsSetCopy.Status.Replicas = 0
		gsSetCopy.Status.ReadyReplicas = 0
		gsSetCopy.Status.AllocatedReplicas = 0
		gsSetCopy.Status.ShutdownReplicas = 0
		gsSetCopy.Status.Instances = 0
		gsSetCopy.Status.ReadyInstances = 0
		gsSetCopy.Status.AllocatedInstances = 0
		gsSetCopy.Status.ShutdownInstances = 0
		if err := r.Status().Update(ctx, gsSetCopy); err != nil {
			return errors.Wrapf(err, "error updating status for gameserverset %s", active.ObjectMeta.Name)
		}

		r.Recorder.Eventf(fleet, v1.EventTypeNormal, "CreatingGameServerSet", "Created GameServerSet %s", active.ObjectMeta.Name)
		return nil
	}

	if replicas != active.Spec.Replicas || active.Spec.Scheduling != fleet.Spec.Scheduling {
		gsSetCopy := active.DeepCopy()
		gsSetCopy.Spec.Replicas = replicas
		gsSetCopy.Spec.Scheduling = fleet.Spec.Scheduling
		if err := r.Update(ctx, gsSetCopy); err != nil {
			return errors.Wrapf(err, "error updating replicas for gameserverset %s", active.ObjectMeta.Name)
		}
		r.Recorder.Eventf(fleet, v1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling active GameServerSet %s from %d to %d", active.ObjectMeta.Name, active.Spec.Replicas, gsSetCopy.Spec.Replicas)
	}

	return nil
}

func (r *Reconciler) updateStatus(ctx context.Context, fleet *singularityv1.Fleet) error {
	// TODO: Log

	list, err := fleet.ListGameServerSet(ctx, r.Client)
	if err != nil {
		return err
	}

	// Agones fetches Fleet again here... is that needed?
	fleetCopy := fleet.DeepCopy()
	fleetCopy.Status.Replicas = 0
	fleetCopy.Status.ReadyReplicas = 0
	fleetCopy.Status.AllocatedReplicas = 0
	fleetCopy.Status.Instances = 0
	fleetCopy.Status.ReadyInstances = 0
	fleetCopy.Status.AllocatedInstances = 0

	for _, gsSet := range list {
		fleetCopy.Status.Replicas += gsSet.Status.Replicas
		fleetCopy.Status.ReadyReplicas += gsSet.Status.ReadyReplicas
		fleetCopy.Status.AllocatedReplicas += gsSet.Status.AllocatedReplicas
		fleetCopy.Status.Instances += gsSet.Status.Instances
		fleetCopy.Status.ReadyInstances += gsSet.Status.ReadyInstances
		fleetCopy.Status.AllocatedInstances += gsSet.Status.AllocatedInstances
	}

	// TODO: Aggregate player status

	if err = r.Status().Update(ctx, fleetCopy); err != nil {
		return errors.Wrapf(err, "error updating status")
	}

	return nil
}
