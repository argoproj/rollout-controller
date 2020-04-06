package rollout

import (
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-rollouts/utils/replicaset"
)

const (
	// restartPodCheckTime prevents the Rollout from not making any progress with restarting Pods. When pods can be restarted
	// faster than the old pods can be scaled down, the parent's ReplicaSet's availableReplicas does not change. A rollout
	// uses changes to the availableReplicas of the ReplicaSet to detect when the controller should requeue and continue
	// deleting pods. In this situation, the rollout does not renqueue and wont make any more progress restarting pods until
	// the resync period passes or another change is made to the Rollout. The controller requeue Rollouts with deleted
	// Pods every 30 seconds to make sure the rollout is not stuck.
	restartPodCheckTime = 30 * time.Second
)

// RolloutPodRestarter describes the components needed for the controller to restart all the pods of
// a rollout.
type RolloutPodRestarter struct {
	client       kubernetes.Interface
	resyncPeriod time.Duration
	enqueueAfter func(obj interface{}, duration time.Duration)
}

// checkEnqueueRollout enqueues a Rollout if the Rollout's restartedAt is within the next resync
func (p RolloutPodRestarter) checkEnqueueRollout(roCtx rolloutContext) {
	r := roCtx.Rollout()
	logCtx := roCtx.Log().WithField("Reconciler", "PodRestarter")
	now := nowFn().UTC()
	if r.Spec.RestartAt == nil || now.After(r.Spec.RestartAt.Time) {
		return
	}
	nextResync := now.Add(p.resyncPeriod)
	// Only enqueue if the Restart time is before the next sync period
	if nextResync.After(r.Spec.RestartAt.Time) {
		timeRemaining := r.Spec.RestartAt.Sub(now)
		logCtx.Infof("Enqueueing Rollout in %s seconds for restart", timeRemaining.String())
		p.enqueueAfter(r, timeRemaining)
	}
}

func (p *RolloutPodRestarter) Reconcile(roCtx rolloutContext) error {
	rollout := roCtx.Rollout()
	logCtx := roCtx.Log().WithField("Reconciler", "PodRestarter")
	p.checkEnqueueRollout(roCtx)
	if !replicaset.NeedsRestart(rollout) {
		return nil
	}
	logCtx.Info("Reconcile pod restarts")
	s := NewSortReplicaSetsByPriority(roCtx)
	for _, rs := range s.allRSs {
		if rs.Status.AvailableReplicas != *rs.Spec.Replicas {
			logCtx.WithField("ReplicaSet", rs.Name).Info("cannot restart pods as not all ReplicasSets are fully available")
			return nil
		}
	}
	sort.Sort(s)
	for _, rs := range s.allRSs {
		reconciledReplicaSet, err := p.reconcilePodsInReplicaSet(roCtx, rs)
		if err != nil {
			return err
		}
		if reconciledReplicaSet {
			return nil
		}
	}
	logCtx.Info("all pods have been restarted and setting restartedAt status")
	roCtx.SetRestartedAt()
	return nil
}

func (p RolloutPodRestarter) reconcilePodsInReplicaSet(roCtx rolloutContext, rs *appsv1.ReplicaSet) (bool, error) {
	logCtx := roCtx.Log().WithField("Reconciler", "PodRestarter")
	restartedAt := roCtx.Rollout().Spec.RestartAt
	pods, err := p.client.CoreV1().Pods(rs.Namespace).List(metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(rs.Spec.Selector),
	})
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			logCtx.Info("cannot reconcile any more pods as pod with deletionTimestamp exists")
			p.enqueueAfter(roCtx.Rollout(), restartPodCheckTime)
			return true, nil
		}
	}

	for _, pod := range pods.Items {
		if restartedAt.After(pod.CreationTimestamp.Time) && pod.DeletionTimestamp == nil {
			newLogCtx := logCtx.WithField("Pod", pod.Name).WithField("CreatedAt", pod.CreationTimestamp.Format(time.RFC3339)).WithField("RestartAt", restartedAt.Format(time.RFC3339))
			newLogCtx.Info("restarting Pod that's older than restartAt Time")
			err := p.client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			return true, err
		}
	}
	return false, nil
}

func NewSortReplicaSetsByPriority(roCtx rolloutContext) SortReplicaSetsByPriority {
	newRS := roCtx.NewRS()
	newRSName := ""
	if newRS != nil {
		newRSName = newRS.Name
	}
	stableRS := roCtx.StableRS()
	stableRSName := ""
	if stableRS != nil {
		stableRSName = stableRS.Name
	}
	return SortReplicaSetsByPriority{
		allRSs:   roCtx.AllRSs(),
		newRS:    newRSName,
		stableRS: stableRSName,
	}
}

// SortReplicaSetsByPriority sorts the ReplicaSets with the following Priority:
// 1. Stable RS
// 2. New RS
// 3. Older ReplicaSets
type SortReplicaSetsByPriority struct {
	allRSs   []*appsv1.ReplicaSet
	newRS    string
	stableRS string
}

func (s SortReplicaSetsByPriority) Len() int {
	return len(s.allRSs)
}

func (s SortReplicaSetsByPriority) Swap(i, j int) {
	rs := s.allRSs[i]
	s.allRSs[i] = s.allRSs[j]
	s.allRSs[j] = rs
}

func (s SortReplicaSetsByPriority) Less(i, j int) bool {
	iRS := s.allRSs[i]
	jRS := s.allRSs[j]
	if iRS.Name == s.stableRS {
		return true
	}
	if jRS.Name == s.stableRS {
		return false
	}
	if iRS.Name == s.newRS {
		return true
	}
	if jRS.Name == s.newRS {
		return false
	}

	return iRS.CreationTimestamp.Before(&jRS.CreationTimestamp)
}
