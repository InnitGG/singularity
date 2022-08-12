package gameserverset

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	singularityv1 "innit.gg/singularity/pkg/apis/singularity/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	maxCreationParalellism         = 16
	maxGameServerCreationsPerBatch = 64

	maxDeletionParallelism         = 64
	maxGameServerDeletionsPerBatch = 64

	// maxPodPendingCount is the maximum number of pending pods per game server set
	maxPodPendingCount = 5000
)

// Reconciler reconciles a GameServerSet object
type Reconciler struct {
	client.Client
	Recorder record.EventRecorder
	Log      logr.Logger
}

//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServerSets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServerSets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServerSets/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: retry on error?

	r.Log.Info("reconcile", "req", req)

	// Retrieve the GameServerSet resource from the cluster, ignoring if it was deleted
	gsSet := &singularityv1.GameServerSet{}
	if err := r.Get(ctx, req.NamespacedName, gsSet); err != nil {
		r.Log.Info("reconcile: resource deleted", "gsSet", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	list, err := gsSet.ListGameServer(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	createCount, toDelete, isPartial := computeReconciliationAction(list, int(gsSet.Spec.Replicas))

	// The GameServerSet is marked for deletion.
	if !gsSet.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if createCount > 0 {
		if err = r.createGameServers(ctx, gsSet, createCount); err != nil {
			r.Log.Error(err, "error creating gameservers")
		}
	}

	if len(toDelete) > 0 {
		if err := r.deleteGameServers(ctx, gsSet, toDelete); err != nil {
			r.Log.Error(err, "error deleting gameservers")
		}
		// TODO
	}

	if isPartial {
		// We have more work to do, reschedule reconciliation for this GameServerSet.
		return ctrl.Result{
			Requeue: true,
		}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&singularityv1.GameServerSet{}).
		WithLogConstructor(func(req *reconcile.Request) logr.Logger {
			if req != nil {
				return r.Log.WithValues("req", req)
			}
			return r.Log
		}).
		Complete(r)
}

// computeReconciliationAction computes the action to take in the reconcilation cycle
func computeReconciliationAction(list []*singularityv1.GameServer, targetReplicaCount int) (int, []*singularityv1.GameServer, bool) {
	var upCount int     // up == Ready or will become ready
	var deleteCount int // number of gameservers to delete

	// track the number of pods that are being created at any given moment by the GameServerSet
	// so we can limit it at a throughput that Kubernetes can handle
	var podPendingCount int // podPending == "up" but don't have a Pod running yet

	var potentialDeletions []*singularityv1.GameServer
	var toDelete []*singularityv1.GameServer

	scheduleDeletion := func(gs *singularityv1.GameServer) {
		toDelete = append(toDelete, gs)
		deleteCount--
	}

	handleGameServerUp := func(gs *singularityv1.GameServer) {
		if upCount >= targetReplicaCount {
			deleteCount++
		} else {
			upCount++
		}

		// Track gameservers that could be potentially deleted
		potentialDeletions = append(potentialDeletions, gs)
	}

	// pass 1 - count allocated/reserved servers only, since those can't be touched
	for _, gs := range list {
		if !gs.IsDeletable() {
			upCount++
		}
	}

	// pass 2 - handle all other statuses
	for _, gs := range list {
		if !gs.IsDeletable() {
			// already handled above
			continue
		}

		// GS being deleted don't count.
		if gs.IsBeingDeleted() {
			continue
		}

		switch gs.Status.State {
		//case singularityv1.GameServerStatePortAllocation:
		//	podPendingCount++
		//	handleGameServerUp(gs)
		case singularityv1.GameServerStateCreating:
			podPendingCount++
			handleGameServerUp(gs)
		case singularityv1.GameServerStateStarting:
			podPendingCount++
			handleGameServerUp(gs)
		case singularityv1.GameServerStateScheduled:
			podPendingCount++
			handleGameServerUp(gs)
		//case singularityv1.GameServerStateRequestReady:
		//	handleGameServerUp(gs)
		case singularityv1.GameServerStateReady:
			handleGameServerUp(gs)
		//case singularityv1.GameServerStateReserved:
		//	handleGameServerUp(gs)

		// GameServerStateShutdown - already handled above
		// GameServerStateAllocated - already handled above
		case singularityv1.GameServerStateError, singularityv1.GameServerStateUnhealthy:
			scheduleDeletion(gs)
		default:
			// unrecognized state, assume it's up.
			handleGameServerUp(gs)
		}
	}

	var partialReconciliation bool
	var numServersToAdd int

	if upCount < targetReplicaCount {
		numServersToAdd = targetReplicaCount - upCount
		originalNumServersToAdd := numServersToAdd

		if numServersToAdd > maxGameServerCreationsPerBatch {
			numServersToAdd = maxGameServerCreationsPerBatch
		}

		if numServersToAdd+podPendingCount > maxPodPendingCount {
			numServersToAdd = maxPodPendingCount - podPendingCount
			if numServersToAdd < 0 {
				numServersToAdd = 0
			}
		}

		if originalNumServersToAdd != numServersToAdd {
			partialReconciliation = true
		}
	}

	if deleteCount > 0 {
		potentialDeletions = singularityv1.SortDescending(potentialDeletions)
		toDelete = append(toDelete, potentialDeletions[0:deleteCount]...)
	}

	if len(toDelete) > maxGameServerDeletionsPerBatch {
		toDelete = toDelete[0:maxGameServerDeletionsPerBatch]
		partialReconciliation = true
	}

	return numServersToAdd, toDelete, partialReconciliation
}

func (r *Reconciler) createGameServers(ctx context.Context, gsSet *singularityv1.GameServerSet, count int) error {
	r.Log.WithValues("count", count).Info("creating gameservers")

	return parallelize(newGameServersChannel(count, gsSet), maxCreationParalellism, func(gs *singularityv1.GameServer) error {
		if err := r.Create(ctx, gs); err != nil {
			return errors.Wrapf(err, "error creating gameserver for gameserverset %s", gsSet.ObjectMeta.Name)
		}

		r.Recorder.Eventf(gsSet, v1.EventTypeNormal, "SuccessfulCreate", "created gameserver: %s", gs.ObjectMeta.Name)
		return nil
	})
}

func (r *Reconciler) deleteGameServers(ctx context.Context, gsSet *singularityv1.GameServerSet, toDelete []*singularityv1.GameServer) error {
	r.Log.WithValues("count", len(toDelete)).Info("deleting gameservers from gameserverset %s", gsSet.ObjectMeta.Name)

	return parallelize(gameServerListToChannel(toDelete), maxDeletionParallelism, func(gs *singularityv1.GameServer) error {
		// We should not delete the GameServers directly, as we would like the GameServer controller to handle deletion.
		gsCopy := gs.DeepCopy()

		// TODO: Draining
		gsCopy.Status.State = singularityv1.GameServerStateShutdown
		if err := r.Update(ctx, gsCopy); err != nil {
			return errors.Wrapf(err, "error updating gameserver %s from status %s to Shutdown status", gs.ObjectMeta.Name, gs.Status.State)
		}

		r.Recorder.Eventf(gsSet, v1.EventTypeNormal, "SuccessfulDelete", "deleted gameserver in state %s: %v", gs.Status.State, gs.ObjectMeta.Name)
		return nil
	})
}
