package controller

import (
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/utils/annotations"
	"github.com/argoproj/argo-rollouts/utils/conditions"
	"github.com/argoproj/argo-rollouts/utils/defaults"
	logutil "github.com/argoproj/argo-rollouts/utils/log"
	replicasetutil "github.com/argoproj/argo-rollouts/utils/replicaset"
)

// rolloutBlueGreen implements the logic for rolling a new replica set.
func (c *Controller) rolloutBlueGreen(r *v1alpha1.Rollout, rsList []*appsv1.ReplicaSet) error {
	logCtx := logutil.WithRollout(r)
	previewSvc, activeSvc, err := c.getPreviewAndActiveServices(r)
	if err != nil {
		return err
	}
	newRS, oldRSs, err := c.getAllReplicaSetsAndSyncRevision(r, rsList, true)
	if err != nil {
		return err
	}
	allRSs := append(oldRSs, newRS)
	// Scale up, if we can.
	logCtx.Infof("Reconciling new ReplicaSet '%s'", newRS.Name)
	scaledUp, err := c.reconcileNewReplicaSet(allRSs, newRS, r)
	if err != nil {
		return err
	}
	if scaledUp {
		logCtx.Infof("Not finished reconciling new ReplicaSet '%s'", newRS.Name)
		return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, false)
	}

	if previewSvc != nil {
		logCtx.Infof("Reconciling preview service '%s'", previewSvc.Name)
		if !annotations.IsSaturated(r, newRS) {
			logutil.WithRollout(r).Infof("New RS '%s' is not fully saturated", newRS.Name)
			return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, false)
		}
		switchPreviewSvc, err := c.reconcilePreviewService(r, newRS, previewSvc, activeSvc)
		if err != nil {
			return err
		}
		if switchPreviewSvc {
			logCtx.Infof("Not finished reconciling preview service' %s'", previewSvc.Name)
			return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, true)
		}
	}

	logCtx.Info("Reconciling pause before switching active service")
	pauseBeforeSwitchActive := c.reconcileBlueGreenPause(activeSvc, r)
	if pauseBeforeSwitchActive {
		logCtx.Info("Not finished reconciling pause before switching active service")
		return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, true)
	}

	logCtx.Infof("Reconciling active service '%s'", activeSvc.Name)
	if !annotations.IsSaturated(r, newRS) {
		logutil.WithRollout(r).Infof("New RS '%s' is not fully saturated", newRS.Name)
		return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, false)
	}
	switchActiveSvc, err := c.reconcileActiveService(r, newRS, previewSvc, activeSvc)
	if err != nil {
		return err
	}
	if switchActiveSvc {
		logCtx.Infof("Not finished reconciling active service '%s'", activeSvc.Name)
		return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, false)
	}

	logCtx.Infof("Reconciling scale down delay")
	scaleDownDelayNotFinished := c.reconcileScaleDownDelay(r, oldRSs, previewSvc, activeSvc)
	if scaleDownDelayNotFinished {
		logCtx.Info("Not finished pausing for scale down delay")
		return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, false)
	}

	// Scale down, if we can.
	logCtx.Info("Reconciling old replica sets")
	scaledDown, err := c.reconcileOldReplicaSets(allRSs, controller.FilterActiveReplicaSets(oldRSs), newRS, r)
	if err != nil {
		return err
	}
	if scaledDown {
		logCtx.Info("Not finished reconciling old replica sets")
		return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, false)
	}

	logCtx.Infof("Confirming rollout is complete")
	if conditions.RolloutComplete(r, &r.Status) {
		logCtx.Info("Cleaning up old replicasets")
		if err := c.cleanupRollouts(oldRSs, r); err != nil {
			return err
		}
	}

	return c.syncRolloutStatusBlueGreen(oldRSs, newRS, previewSvc, activeSvc, r, false)
}

func (c *Controller) reconcileScaleDownDelay(rollout *v1alpha1.Rollout, oldRSs []*appsv1.ReplicaSet, previewSvc, activeSvc *corev1.Service) bool {
	oldPodsCount := replicasetutil.GetReplicaCountForReplicaSets(oldRSs)
	if oldPodsCount == 0 {
		// Already scaled down old RS
		return false
	}
	if rollout.Status.BlueGreen.ScaleDownDelayStartTime == nil {
		return true
	}
	scaleDownDelaySeconds := defaults.GetScaleDownDelaySecondsOrDefault(rollout)
	c.checkEnqueueRolloutDuringWait(rollout, *rollout.Status.BlueGreen.ScaleDownDelayStartTime, scaleDownDelaySeconds)
	pauseUntil := metav1.NewTime(rollout.Status.BlueGreen.ScaleDownDelayStartTime.Add(time.Duration(scaleDownDelaySeconds) * time.Second))
	now := metav1.Now()
	return now.Before(&pauseUntil)
}

func (c *Controller) reconcileBlueGreenPause(activeSvc *corev1.Service, rollout *v1alpha1.Rollout) bool {
	if rollout.Spec.Strategy.BlueGreenStrategy.PreviewService == "" {
		return false
	}
	if _, ok := activeSvc.Spec.Selector[v1alpha1.DefaultRolloutUniqueLabelKey]; !ok {
		return false
	}

	return rollout.Spec.Paused && rollout.Status.PauseStartTime != nil
}

// scaleDownOldReplicaSetsForBlueGreen scales down old replica sets when rollout strategy is "Blue Green".
func (c *Controller) scaleDownOldReplicaSetsForBlueGreen(allRSs []*appsv1.ReplicaSet, oldRSs []*appsv1.ReplicaSet, rollout *v1alpha1.Rollout) (int32, error) {
	availablePodCount := replicasetutil.GetAvailableReplicaCountForReplicaSets(allRSs)
	if availablePodCount <= defaults.GetRolloutReplicasOrDefault(rollout) {
		// Cannot scale down.
		return 0, nil
	}
	logutil.WithRollout(rollout).Infof("Found %d available pods, scaling down old RSes", availablePodCount)

	sort.Sort(controller.ReplicaSetsByCreationTimestamp(oldRSs))

	totalScaledDown := int32(0)
	for _, targetRS := range oldRSs {
		if *(targetRS.Spec.Replicas) == 0 {
			// cannot scale down this ReplicaSet.
			continue
		}
		scaleDownCount := *(targetRS.Spec.Replicas)
		// Scale down.
		newReplicasCount := int32(0)
		_, _, err := c.scaleReplicaSetAndRecordEvent(targetRS, newReplicasCount, rollout)
		if err != nil {
			return totalScaledDown, err
		}

		totalScaledDown += scaleDownCount
	}

	return totalScaledDown, nil
}

func (c *Controller) syncRolloutStatusBlueGreen(oldRSs []*appsv1.ReplicaSet, newRS *appsv1.ReplicaSet, previewSvc *corev1.Service, activeSvc *corev1.Service, r *v1alpha1.Rollout, addPause bool) error {
	allRSs := append(oldRSs, newRS)
	newStatus := c.calculateBaseStatus(allRSs, newRS, r)
	previewSelector, ok := c.getRolloutSelectorLabel(previewSvc)
	if !ok {
		previewSelector = ""
	}
	newStatus.BlueGreen.PreviewSelector = previewSelector
	activeSelector, ok := c.getRolloutSelectorLabel(activeSvc)
	if !ok {
		activeSelector = ""
	}
	newStatus.BlueGreen.ActiveSelector = activeSelector

	activeRS := replicasetutil.GetActiveReplicaSet(allRSs, newStatus.BlueGreen.ActiveSelector)
	if activeRS != nil {
		newStatus.HPAReplicas = replicasetutil.GetActualReplicaCountForReplicaSets([]*appsv1.ReplicaSet{activeRS})
		newStatus.Selector = metav1.FormatLabelSelector(activeRS.Spec.Selector)
	} else {
		newStatus.HPAReplicas = replicasetutil.GetActualReplicaCountForReplicaSets(allRSs)
		newStatus.Selector = metav1.FormatLabelSelector(r.Spec.Selector)
	}

	newStatus.BlueGreen.ScaleDownDelayStartTime = r.Status.BlueGreen.ScaleDownDelayStartTime
	if activeSelector == newStatus.CurrentPodHash && newStatus.BlueGreen.ScaleDownDelayStartTime == nil {
		now := metav1.Now()
		newStatus.BlueGreen.ScaleDownDelayStartTime = &now
	}
	if activeSelector != newStatus.CurrentPodHash || replicasetutil.GetReplicaCountForReplicaSets(oldRSs) == 0 {
		newStatus.BlueGreen.ScaleDownDelayStartTime = nil
	}

	pauseStartTime, paused := calculatePauseStatus(r, addPause)
	newStatus.PauseStartTime = pauseStartTime
	newStatus = c.calculateRolloutConditions(r, newStatus, allRSs, newRS)
	return c.persistRolloutStatus(r, &newStatus, &paused)
}

// Should run only on scaling events and not during the normal rollout process.
func (c *Controller) scaleBlueGreen(rollout *v1alpha1.Rollout, newRS *appsv1.ReplicaSet, oldRSs []*appsv1.ReplicaSet, previewSvc *corev1.Service, activeSvc *corev1.Service) error {
	rolloutReplicas := defaults.GetRolloutReplicasOrDefault(rollout)
	previewSelector, ok := c.getRolloutSelectorLabel(previewSvc)
	if !ok {
		previewSelector = ""
	}
	activeSelector, ok := c.getRolloutSelectorLabel(activeSvc)
	if !ok {
		activeSelector = ""
	}

	// If there is only one replica set with pods, then we should scale that up to the full count of the
	// rollout. If there is no replica set with pods, then we should scale up the newest replica set.
	if activeOrLatest := replicasetutil.FindActiveOrLatest(newRS, oldRSs); activeOrLatest != nil {
		if *(activeOrLatest.Spec.Replicas) != rolloutReplicas {
			_, _, err := c.scaleReplicaSetAndRecordEvent(activeOrLatest, rolloutReplicas, rollout)
			return err
		}
	}

	allRS := append([]*appsv1.ReplicaSet{newRS}, oldRSs...)
	activeRS := replicasetutil.GetActiveReplicaSet(allRS, rollout.Status.BlueGreen.ActiveSelector)
	if activeRS != nil {
		if *(activeRS.Spec.Replicas) != rolloutReplicas {
			_, _, err := c.scaleReplicaSetAndRecordEvent(activeRS, rolloutReplicas, rollout)
			return err
		}
	}

	if newRS != nil {
		if *(newRS.Spec.Replicas) != rolloutReplicas {
			_, _, err := c.scaleReplicaSetAndRecordEvent(newRS, rolloutReplicas, rollout)
			return err
		}
	}

	// Old replica sets should be fully scaled down if they aren't receiving traffic from the active or
	// preview service. This case handles replica set adoption during a saturated new replica set.
	for _, old := range controller.FilterActiveReplicaSets(oldRSs) {
		oldLabel, ok := old.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]
		if !ok || (oldLabel != activeSelector && oldLabel != previewSelector) {
			if _, _, err := c.scaleReplicaSetAndRecordEvent(old, 0, rollout); err != nil {
				return err
			}
		}
	}
	return nil
}
