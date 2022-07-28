package controllers

import (
	"context"
	singularityv1alpha1 "innit.gg/singularity/pkg/apis/singularity/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GameServerSetReconciler reconciles a GameServerSet object
type GameServerSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServerSets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServerSets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=singularity.innit.gg,resources=GameServerSets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GameServerSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *GameServerSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameServerSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&singularityv1alpha1.GameServerSet{}).
		Complete(r)
}
