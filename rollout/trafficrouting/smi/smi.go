package smi

import (

	"context"
	"fmt"
	patchtypes "k8s.io/apimachinery/pkg/types"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"

	logutil "github.com/argoproj/argo-rollouts/utils/log"
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	smiv1alpha1 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha1"
	smiclientset "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/argoproj/argo-rollouts/utils/diff"
)

const (
	// Type holds this controller type
	Type = "SMI"
)

// ReconcilerConfig describes static configuration data for the SMI reconciler
type ReconcilerConfig struct {
	Rollout        *v1alpha1.Rollout
	Client         smiclientset.Interface
	Recorder       record.EventRecorder
	ControllerKind schema.GroupVersionKind
	ApiVersion     string
}

// Reconciler holds required fields to reconcile SMI resources
type Reconciler struct {
	cfg ReconcilerConfig
	log *logrus.Entry
}

// NewReconciler returns a reconciler struct that brings the SMI into the desired state
func NewReconciler(cfg ReconcilerConfig) *Reconciler {
	return &Reconciler{
		cfg: cfg,
		log: logutil.WithRollout(cfg.Rollout),
	}
}

// Type indicates this reconciler is an SMI reconciler
func (r *Reconciler) Type() string {
	return Type
}

// TODO: Make code compatible with multiple TrafficSplit versions
func (r *Reconciler) Reconcile(desiredWeight int32) error {
	trafficSplitName := r.cfg.Rollout.Spec.Strategy.Canary.TrafficRouting.SMI.TrafficSplitName

	// Service weights formatted for Traffic Split spec
	canaryWeight := resource.MustParse(string(desiredWeight))
	stableWeight := resource.MustParse(string(100-desiredWeight))

	// If root service not set, then set root service to be stable service
	rootSvc := r.cfg.Rollout.Spec.Strategy.Canary.TrafficRouting.SMI.RootService
	if rootSvc == "" {
		rootSvc = r.cfg.Rollout.Spec.Strategy.Canary.StableService
	}

	client := r.cfg.Client.SplitV1alpha1()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if Traffic Split exists in namespace
	trafficSplit, err := client.TrafficSplits(r.cfg.Rollout.Namespace).Get(ctx, trafficSplitName, metav1.GetOptions{})
	if err != nil  && k8serrors.IsNotFound(err) {
		msg := fmt.Sprintf("Traffic Split `%s` not found", trafficSplitName)
		r.cfg.Recorder.Event(r.cfg.Rollout, corev1.EventTypeNormal, "TrafficSplitNotFound", msg)
		// TODO: check for double-logging
	}

	trafficSplitSpec := smiv1alpha1.TrafficSplitSpec{
		Service: rootSvc,
		Backends: []smiv1alpha1.TrafficSplitBackend{
			{
				Service: r.cfg.Rollout.Spec.Strategy.Canary.CanaryService,
				Weight: &canaryWeight,
			},
			{
				Service: r.cfg.Rollout.Spec.Strategy.Canary.StableService,
				Weight: &stableWeight,
			},
		},
	}

	// Patch existing Traffic Split
	if trafficSplit != nil {
		controllerRef := metav1.GetControllerOf(trafficSplit)
		if controllerRef == nil || r.cfg.Rollout.UID != controllerRef.UID {
			return err // TODO: Create error case - RO doesn't own TS with TSname
		}
		patch, modified, err := diff.CreateTwoWayMergePatch(
			smiv1alpha1.TrafficSplit{
				Spec:       trafficSplit.Spec,
			},
			smiv1alpha1.TrafficSplit{
				Spec: trafficSplitSpec,
			},
			smiv1alpha1.TrafficSplit{},
		)
		if !modified {
			return nil
		}
		_, err = client.TrafficSplits(r.cfg.Rollout.Namespace).Patch(trafficSplitName, patchtypes.MergePatchType, patch)
		if err != nil {
			return err
		}
		return nil
	}

	// Create new Traffic Split
	trafficSplit = &smiv1alpha1.TrafficSplit{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: trafficSplitName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.cfg.Rollout, r.cfg.ControllerKind),
			},
		},
		Spec: trafficSplitSpec,
	}

	_, err = client.TrafficSplits(r.cfg.Rollout.Namespace).Create(ctx, trafficSplit, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}
