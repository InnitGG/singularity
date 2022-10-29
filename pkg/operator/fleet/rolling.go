/*
 *     Singularity is an open-source game server orchestration framework
 *     Copyright (C) 2022 Innit Incorporated
 *
 *     This program is free software: you can redistribute it and/or modify
 *     it under the terms of the GNU Affero General Public License as published
 *     by the Free Software Foundation, either version 3 of the License, or
 *     (at your option) any later version.
 *
 *     This program is distributed in the hope that it will be useful,
 *     but WITHOUT ANY WARRANTY; without even the implied warranty of
 *     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *     GNU Affero General Public License for more details.
 *
 *     You should have received a copy of the GNU Affero General Public License
 *     along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package fleet

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	singularityv1 "innit.gg/singularity/pkg/apis/singularity/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/integer"
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

	if _, err = r.cleanupUnhealthyReplicas(ctx, rest, fleet, maxScaleDown); err != nil {
		// There could be the case when GameServerSet would be updated from another place, say Status or Spec would be updated
		// We don't want to propagate such errors further
		// And this set in sync with reconcileOldReplicaSets() Kubernetes code
		return nil
	}
	// TODO: scale down

	return nil
}

func (r *Reconciler) cleanupUnhealthyReplicas(ctx context.Context, rest []*singularityv1.GameServerSet,
	fleet *singularityv1.Fleet, maxCleanupCount int32) (int32, error) {

	// Safely scale down all old GameServerSets with unhealthy replicas.
	totalScaledDown := int32(0)
	for i, gsSet := range rest {
		if totalScaledDown >= maxCleanupCount {
			// We have scaled down enough.
			break
		}
		if gsSet.Spec.Replicas == 0 {
			// Cannot scale down this replica set.
			continue
		}
		if gsSet.Spec.Replicas == gsSet.Status.ReadyReplicas {
			// No unhealthy replicas found, no scaling required
			continue
		}

		scaledDownCount := int32(integer.IntMin(int(maxCleanupCount-totalScaledDown), int(gsSet.Spec.Replicas-gsSet.Status.ReadyReplicas)))
		newReplicasCount := gsSet.Spec.Replicas - scaledDownCount
		if newReplicasCount > gsSet.Spec.Replicas {
			return 0, fmt.Errorf("invalid scale down request for gameserverset %s: %d -> %d", gsSet.Name, gsSet.Spec.Replicas, newReplicasCount)
		}

		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Spec.Replicas = newReplicasCount
		totalScaledDown += scaledDownCount
		if err := r.Update(ctx, gsSetCopy); err != nil {
			return totalScaledDown, errors.Wrapf(err, "error updating gameserverset %s/%s", gsSetCopy.Namespace, gsSetCopy.ObjectMeta.Name)
		}

		r.Recorder.Eventf(fleet, v1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling inactive GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, gsSet.Spec.Replicas, gsSetCopy.Spec.Replicas)

		rest[i] = gsSetCopy
	}

	return totalScaledDown, nil
}
