package fleet

import (
	"context"
	"github.com/pkg/errors"
	singularityv1 "innit.gg/singularity/pkg/apis/singularity/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// https://github.com/googleforgames/agones/blob/8d01f2ce9c34ffadfdf22ab2fb3b1bafae7e6389/pkg/fleets/controller.go#L415
func (r *Reconciler) handleRollingUpdateDeployment(ctx context.Context, fleet *singularityv1.Fleet, active *singularityv1.GameServerSet, rest []*singularityv1.GameServerSet) (int32, error) {
	// First, start by rolling out update for the current active GameServerSet
	replicas, err := r.handleRollingUpdateActive(fleet, active, rest)
	if err != nil {
		return 0, err
	}
	if err := r.handleRollingUpdateRest(ctx, fleet, active, rest); err != nil {
		return 0, err
	}

	return replicas, nil
}

// https://github.com/googleforgames/agones/blob/8d01f2ce9c34ffadfdf22ab2fb3b1bafae7e6389/pkg/fleets/controller.go#L428
func (r *Reconciler) handleRollingUpdateActive(fleet *singularityv1.Fleet, active *singularityv1.GameServerSet, rest []*singularityv1.GameServerSet) (int32, error) {
	desiredReplicas := active.Spec.Replicas

	// Leave room for Allocated GameServers in old GameServerSets.
	allocatedReplicas := singularityv1.CountStatusAllocatedReplicas(rest)

	// If the state doesn't match the desired replicas, ignore.
	// This means we're in the middle of a rolling update, and we should wait.
	if active.Spec.Replicas != active.Status.Replicas {
		return desiredReplicas, nil
	}

	// If there are no desired replicas, ignore.
	// The dangling GameServerSet will be removed at a later stage.
	if fleet.Spec.Replicas == 0 {
		return 0, nil
	}

	// If desired replicas in the active GameServerSet is greater or equal to the Fleet's desired replicas,
	// then we don't need to continue anymore.
	if active.Spec.Replicas >= (fleet.Spec.Replicas - allocatedReplicas) {
		return fleet.Spec.Replicas - allocatedReplicas, nil
	}

	// Determine how many more GameServers than the desired replicas is acceptable during a rolling update.
	sr, err := intstr.GetScaledValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxSurge, int(fleet.Spec.Replicas), true)
	if err != nil {
		return 0, errors.Wrapf(err, "error parsing MaxSurge value: %s", fleet.Spec.Strategy.RollingUpdate.MaxSurge)
	}
	surge := int32(sr)

	desiredReplicas = fleet.UpperBoundReplicas(desiredReplicas + surge)
	total := singularityv1.CountStatusReplicas(rest) + desiredReplicas

	// Make sure that we don't exceed the max surge.
	maxSurge := fleet.Spec.Replicas + surge
	if total > maxSurge {
		desiredReplicas = fleet.LowerBoundReplicas(desiredReplicas - (total - maxSurge))
	}

	// Take allocated GameServers into consideration.
	// Ensure the total active GameServers will not exceed the desired amount.
	if desiredReplicas+allocatedReplicas > fleet.Spec.Replicas {
		desiredReplicas = fleet.LowerBoundReplicas(fleet.Spec.Replicas - allocatedReplicas)
	}

	return desiredReplicas, nil
}

// https://github.com/googleforgames/agones/blob/8d01f2ce9c34ffadfdf22ab2fb3b1bafae7e6389/pkg/fleets/controller.go#L514
// https://github.com/kubernetes/kubernetes/blob/3aafe756986232ee9208681ee22b38f5c19424a2/pkg/controller/deployment/rolling.go#L87
func (r *Reconciler) handleRollingUpdateRest(ctx context.Context, fleet *singularityv1.Fleet, active *singularityv1.GameServerSet, rest []*singularityv1.GameServerSet) error {
	// https://github.com/kubernetes/kubernetes/blob/3ffdfbe286ebcea5d75617da6accaf67f815e0cf/staging/src/k8s.io/kubectl/pkg/util/deployment/deployment.go#L238
	ur, err := intstr.GetScaledValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxUnavailable, int(fleet.Spec.Replicas), false)
	if err != nil {
		return errors.Wrapf(err, "error parsing MaxUnavailable value: %s", fleet.Spec.Strategy.RollingUpdate.MaxUnavailable)
	}
	unavailable := int32(ur)

	if unavailable == 0 {
		unavailable = 1
	}

	// MaxUnavailable should not exceed the desired replicas.
	if unavailable > fleet.Spec.Replicas {
		unavailable = fleet.Spec.Replicas
	}

	// Check if we can scale down.
	gsSets := rest
	gsSets = append(gsSets, active)
	minAvailable := fleet.Spec.Replicas - unavailable

	desiredReplicas := singularityv1.CountSpecReplicas(gsSets)
	unavailableGSCount := active.Spec.Replicas - active.Status.ReadyReplicas - active.Status.AllocatedReplicas
	maxScaleDown := desiredReplicas - minAvailable - unavailableGSCount

	// We don't have the room to scale down.
	if maxScaleDown <= 0 {
		return nil
	}

	// TODO: clean up unhealthy replicas
	// TODO: scale down

	return nil
}