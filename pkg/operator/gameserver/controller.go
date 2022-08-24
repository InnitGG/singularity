package gameserver

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	singularityv1 "innit.gg/singularity/pkg/apis/singularity/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a GameServer object
type Reconciler struct {
	client.Client
	Recorder record.EventRecorder
	Log      logr.Logger
}

//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServers/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("reconcile")

	// Retrieve the GameServer resource from the cluster, ignoring if it was deleted
	gs := &singularityv1.GameServer{}
	if err := r.Get(ctx, req.NamespacedName, gs); err != nil {
		l.Info("reconcile: resource deleted", "gs", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 	if gs, err = c.syncGameServerDeletionTimestamp(ctx, gs); err != nil {
	//		return err
	//	}
	//	if gs, err = c.syncGameServerPortAllocationState(ctx, gs); err != nil {
	//		return err
	//	}
	//	if gs, err = c.syncGameServerCreatingState(ctx, gs); err != nil {
	//		return err
	//	}
	//	if gs, err = c.syncGameServerStartingState(ctx, gs); err != nil {
	//		return err
	//	}
	//	if gs, err = c.syncGameServerRequestReadyState(ctx, gs); err != nil {
	//		return err
	//	}
	//	if gs, err = c.syncDevelopmentGameServer(ctx, gs); err != nil {
	//		return err
	//	}
	//	if err := c.syncGameServerShutdownState(ctx, gs); err != nil {
	//		return err
	//	}

	if err := r.reconcileGameServerDeletion(ctx, gs); err != nil {
		return ctrl.Result{}, err
	}

	switch gs.Status.State {
	case singularityv1.GameServerStateCreating:
		if gs.ObjectMeta.DeletionTimestamp.IsZero() {
			if err := r.reconcileGameServerCreation(ctx, gs); err != nil {
				return ctrl.Result{}, err
			}
		}
		break
	case singularityv1.GameServerStateStarting:
		break
	case singularityv1.GameServerStateShutdown:
		break
	case "":
		if err := r.reconcileGameServerState(ctx, gs); err != nil {
			return ctrl.Result{}, err
		}
		break
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&singularityv1.GameServer{}).
		Owns(&singularityv1.GameServerInstance{}).
		WithLogConstructor(func(req *reconcile.Request) logr.Logger {
			if req != nil {
				return r.Log.WithValues("req", req)
			}
			return r.Log
		}).
		Complete(r)
}

func (r *Reconciler) reconcileGameServerDeletion(ctx context.Context, gs *singularityv1.GameServer) error {
	if gs.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	l := log.FromContext(ctx)
	l.Info("reconcile: deletion timestamp")

	pod, err := r.getGameServerPod(ctx, gs)
	if pod != nil {
		// We only need to delete the Pod once
		if pod.ObjectMeta.DeletionTimestamp.IsZero() {
			if err = r.Delete(ctx, pod); err != nil {
				return err
			}
			r.Recorder.Eventf(gs, v1.EventTypeNormal, string(gs.Status.State), "Deleting Pod %s", pod.ObjectMeta.Name)
		}

		return nil
	}

	// TODO: Remove finalizers

	return nil
}

func (r *Reconciler) reconcileGameServerCreation(ctx context.Context, gs *singularityv1.GameServer) error {
	_, err := r.getGameServerPod(ctx, gs)
	if k8serrors.IsNotFound(err) {
		// Only create resources if the backing Pod doesn't exist
		// TODO: Perhaps check if Role, ServiceAccount, and RoleBinding also exist?
		if err = r.createGameServerResources(ctx, gs); err != nil {
			return err
		}
	}

	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = singularityv1.GameServerStateStarting
	if err = r.Status().Update(ctx, gsCopy); err != nil {
		return errors.Wrapf(err, "error updating GameServer %s to Starting state", gs.Name)
	}
	return nil
}

func (r *Reconciler) reconcileGameServerState(ctx context.Context, gs *singularityv1.GameServer) error {
	// TODO: Is this the correct way to default state?
	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = singularityv1.GameServerStateCreating

	if err := r.Status().Update(ctx, gsCopy); err != nil {
		return errors.Wrapf(err, "error updating GameServer %s to Creating state", gs.Name)
	}

	return nil
}

// getGameServerPod returns the Pod associated with the GameServer
func (r *Reconciler) getGameServerPod(ctx context.Context, gs *singularityv1.GameServer) (*v1.Pod, error) {
	var pod v1.Pod
	key := client.ObjectKey{
		Namespace: gs.ObjectMeta.Namespace,
		Name:      gs.ObjectMeta.Name,
	}
	if err := r.Get(ctx, key, &pod); err != nil {
		// The Pod is not found
		return nil, err
	}

	// Check if the Pod is actually controlled by this GameServer
	if !metav1.IsControlledBy(&pod, gs) {
		return nil, k8serrors.NewNotFound(v1.Resource("pod"), gs.ObjectMeta.Name)
	}

	return &pod, nil
}

func (r *Reconciler) createGameServerResources(ctx context.Context, gs *singularityv1.GameServer) error {
	l := log.FromContext(ctx)

	role := gs.Role()
	serviceAccount := gs.ServiceAccount()
	roleBinding := gs.RoleBinding()
	pod := gs.Pod()

	l.Info("reconcile: creating role")
	err := r.Create(ctx, role)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		// TODO: Record event
		l.Error(err, "reconcile: error creating role", "role", role)
		return errors.Wrapf(err, "error creating Role for GameServer %s", gs.ObjectMeta.Name)
	}

	r.Recorder.Event(gs, v1.EventTypeNormal, string(gs.Status.State), fmt.Sprintf("Role %s created", role.ObjectMeta.Name))

	l.Info("reconcile: creating serviceaccount")
	err = r.Create(ctx, serviceAccount)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		// TODO: Record event
		l.Error(err, "reconcile: error creating serviceaccount", "serviceaccount", serviceAccount)
		return errors.Wrapf(err, "error creating ServiceAccount for GameServer %s", gs.ObjectMeta.Name)
	}

	r.Recorder.Event(gs, v1.EventTypeNormal, string(gs.Status.State), fmt.Sprintf("ServiceAccount %s created", serviceAccount.ObjectMeta.Name))

	l.Info("reconcile: creating rolebinding")
	err = r.Create(ctx, roleBinding)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		// TODO: Record event
		l.Error(err, "reconcile: error creating rolebinding", "rolebinding", roleBinding)
		return errors.Wrapf(err, "error creating RoleBinding for GameServer %s", gs.ObjectMeta.Name)
	}

	r.Recorder.Event(gs, v1.EventTypeNormal, string(gs.Status.State), fmt.Sprintf("RoleBinding %s created", roleBinding.ObjectMeta.Name))

	l.Info("reconcile: creating pod")
	err = r.Create(ctx, pod)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		// TODO: Record event
		l.Error(err, "reconcile: error creating pod", "pod", pod)
		return errors.Wrapf(err, "error creating Pod for GameServer %s", gs.ObjectMeta.Name)
	}

	r.Recorder.Event(gs, v1.EventTypeNormal, string(gs.Status.State), fmt.Sprintf("Pod %s created", pod.ObjectMeta.Name))

	// TODO: network policy

	return nil
}
