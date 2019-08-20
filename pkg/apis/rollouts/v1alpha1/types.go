package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=rollouts,shortName=ro
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.HPAReplicas,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Desired",type="integer",JSONPath=".spec.replicas",description="Number of desired pods"
// +kubebuilder:printcolumn:name="Current",type="integer",JSONPath=".status.replicas",description="Total number of non-terminated pods targeted by this rollout"
// +kubebuilder:printcolumn:name="Up-to-date",type="integer",JSONPath=".status.updatedReplicas",description="Total number of non-terminated pods targeted by this rollout that have the desired template spec"
// +kubebuilder:printcolumn:name="Available",type="integer",JSONPath=".status.availableReplicas",description="Total number of available pods (ready for at least minReadySeconds) targeted by this rollout"

// Rollout is a specification for a Rollout resource
type Rollout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RolloutSpec   `json:"spec"`
	Status RolloutStatus `json:"status,omitempty"`
}

// RolloutSpec is the spec for a Rollout resource
type RolloutSpec struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this rollout.
	// It must match the pod template's labels.
	Selector *metav1.LabelSelector `json:"selector"`
	// Template describes the pods that will be created.
	Template corev1.PodTemplateSpec `json:"template"`
	// Minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`
	// The deployment strategy to use to replace existing pods with new ones.
	// +optional
	Strategy RolloutStrategy `json:"strategy"`
	// The number of old ReplicaSets to retain. If unspecified, will retain 10 old ReplicaSets
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty"`
	// Paused pauses the rollout at its current step.
	Paused bool `json:"paused,omitempty"`
	// ProgressDeadlineSeconds The maximum time in seconds for a rollout to
	// make progress before it is considered to be failed. Argo Rollouts will
	// continue to process failed rollouts and a condition with a
	// ProgressDeadlineExceeded reason will be surfaced in the rollout status.
	// Note that progress will not be estimated during the time a rollout is paused.
	// Defaults to 600s.
	ProgressDeadlineSeconds *int32 `json:"progressDeadlineSeconds,omitempty"`
}

const (
	// DefaultRolloutUniqueLabelKey is the default key of the selector that is added
	// to existing ReplicaSets (and label key that is added to its pods) to prevent the existing ReplicaSets
	// to select new pods (and old pods being select by new ReplicaSet).
	DefaultRolloutUniqueLabelKey string = "rollouts-pod-template-hash"
	// DefaultReplicaSetScaleDownAtLabelKey is the default key attached to an old stable ReplicaSet after
	// the rollout transitioned to a new version. It contains the time when the controller can scale down the RS.
	DefaultReplicaSetScaleDownAtLabelKey = "scale-down-at"
)

// RolloutStrategy defines strategy to apply during next rollout
type RolloutStrategy struct {
	// +optional
	BlueGreenStrategy *BlueGreenStrategy `json:"blueGreen,omitempty"`
	// +optional
	CanaryStrategy *CanaryStrategy `json:"canary,omitempty"`
}

// BlueGreenStrategy defines parameters for Blue Green deployment
type BlueGreenStrategy struct {
	// Name of the service that the rollout modifies as the active service.
	ActiveService string `json:"activeService,omitempty"`
	// Name of the service that the rollout modifies as the preview service.
	// +optional
	PreviewService string `json:"previewService,omitempty"`
	// PreviewReplica the number of replicas to run under the preview service before the switchover. Once the rollout is
	// resumed the new replicaset will be full scaled up before the switch occurs
	// +optional
	PreviewReplicaCount *int32 `json:"previewReplicaCount,omitempty"`
	// AutoPromotionEnabled indicates if the rollout should automatically promote the new ReplicaSet
	// to the active service or enter a paused state. If not specified, the default value is true.
	// +optional
	AutoPromotionEnabled *bool `json:"autoPromotionEnabled,omitempty"`
	// AutoPromotionSeconds automatically promotes the current ReplicaSet to active after the
	// specified pause delay in seconds after the ReplicaSet becomes ready.
	// If omitted, the Rollout enters and remains in a paused state until manually resumed by
	// resetting spec.Paused to false.
	// +optional
	AutoPromotionSeconds *int32 `json:"autoPromotionSeconds,omitempty"`
	// ScaleDownDelaySeconds adds a delay before scaling down the previous replicaset.
	// If omitted, the Rollout waits 30 seconds before scaling down the previous ReplicaSet.
	// A minimum of 30 seconds is recommended to ensure IP table propagation across the nodes in
	// a cluster. See https://github.com/argoproj/argo-rollouts/issues/19#issuecomment-476329960 for
	// more information
	// +optional
	ScaleDownDelaySeconds *int32 `json:"scaleDownDelaySeconds,omitempty"`
}

// CanaryStrategy defines parameters for a Replica Based Canary
type CanaryStrategy struct {
	// CanaryService holds the name of a service which selects pods with canary version and don't select any pods with stable version.
	// +optional
	CanaryService string `json:"canaryService,omitempty"`
	// Steps define the order of phases to execute the canary deployment
	// +optional
	Steps []CanaryStep `json:"steps,omitempty"`
	// MaxUnavailable The maximum number of pods that can be unavailable during the update.
	// Value can be an absolute number (ex: 5) or a percentage of total pods at the start of update (ex: 10%).
	// Absolute number is calculated from percentage by rounding down.
	// This can not be 0 if MaxSurge is 0.
	// By default, a fixed value of 1 is used.
	// Example: when this is set to 30%, the old RC can be scaled down by 30%
	// immediately when the rolling update starts. Once new pods are ready, old RC
	// can be scaled down further, followed by scaling up the new RC, ensuring
	// that at least 70% of original number of pods are available at all times
	// during the update.
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`

	// MaxSurge The maximum number of pods that can be scheduled above the original number of
	// pods.
	// Value can be an absolute number (ex: 5) or a percentage of total pods at
	// the start of the update (ex: 10%). This can not be 0 if MaxUnavailable is 0.
	// Absolute number is calculated from percentage by rounding up.
	// By default, a value of 1 is used.
	// Example: when this is set to 30%, the new RC can be scaled up by 30%
	// immediately when the rolling update starts. Once old pods have been killed,
	// new RC can be scaled up further, ensuring that total number of pods running
	// at any time during the update is atmost 130% of original pods.
	// +optional
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty"`
}

// CanaryStep defines a step of a canary deployment.
type CanaryStep struct {
	// SetWeight sets what percentage of the newRS should receive
	SetWeight *int32 `json:"setWeight,omitempty"`
	// Pause freezes the rollout by setting spec.Paused to true.
	// A Rollout will resume when spec.Paused is reset to false.
	// +optional
	Pause *RolloutPause `json:"pause,omitempty"`
}

// RolloutPause defines a pause stage for a rollout
type RolloutPause struct {
	// Duration the amount of time to wait before moving to the next step.
	// +optional
	Duration *int32 `json:"duration,omitempty"`
}

// RolloutStatus is the status for a Rollout resource
type RolloutStatus struct {
	// CurrentPodHash the hash of the current pod template
	// +optional
	CurrentPodHash string `json:"currentPodHash,omitempty"`
	// CurrentStepHash the hash of the current list of steps for the current strategy. This is used to detect when the
	// list of current steps change
	// +optional
	CurrentStepHash string `json:"currentStepHash,omitempty"`
	// Total number of non-terminated pods targeted by this rollout (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty"`
	// Total number of non-terminated pods targeted by this rollout that have the desired template spec.
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`
	// Total number of ready pods targeted by this rollout.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// Total number of available pods (ready for at least minReadySeconds) targeted by this rollout.
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
	// CurrentStepIndex defines the current step of the rollout is on. If the current step index is null, the
	// controller will execute the rollout.
	// +optional
	CurrentStepIndex *int32 `json:"currentStepIndex,omitempty"`
	// PauseStartTime this field is set when the rollout is in a pause step and indicates the time the wait started at
	// +optional
	PauseStartTime *metav1.Time `json:"pauseStartTime,omitempty"`
	// Count of hash collisions for the Rollout. The Rollout controller uses this
	// field as a collision avoidance mechanism when it needs to create the name for the
	// newest ReplicaSet.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty"`
	// The generation observed by the rollout controller by taking a hash of the spec.
	// +optional
	ObservedGeneration string `json:"observedGeneration,omitempty"`
	// Conditions a list of conditions a rollout can have.
	// +optional
	Conditions []RolloutCondition `json:"conditions,omitempty"`
	// Canary describes the state of the canary rollout
	// +optional
	Canary CanaryStatus `json:"canary,omitempty"`
	// BlueGreen describes the state of the bluegreen rollout
	// +optional
	BlueGreen BlueGreenStatus `json:"blueGreen,omitempty"`
	// HPAReplicas the number of non-terminated replicas that are receiving active traffic
	// +optional
	HPAReplicas int32 `json:"HPAReplicas,omitempty"`
	// Selector that identifies the pods that are receiving active traffic
	// +optional
	Selector string `json:"selector,omitempty"`
}

// BlueGreenStatus status fields that only pertain to the blueGreen rollout
type BlueGreenStatus struct {
	// PreviewSelector indicates which replicas set the preview service is serving traffic to
	// +optional
	PreviewSelector string `json:"previewSelector,omitempty"`
	// ActiveSelector indicates which replicas set the active service is serving traffic to
	// +optional
	ActiveSelector string `json:"activeSelector,omitempty"`
	// PreviousActiveSelector indicates the last selector that the active service used. This is used to know which replicaset
	// to avoid scaling down for the scale down delay
	// +optional
	PreviousActiveSelector string `json:"previousActiveSelector,omitempty"`
	// ScaleDownDelayStartTime indicates the start of the scaleDownDelay
	// +optional
	ScaleDownDelayStartTime *metav1.Time `json:"scaleDownDelayStartTime,omitempty"`
	// ScaleUpPreviewCheckPoint indicates that the Replicaset receiving traffic from the preview service is ready to be scaled up after the rollout is unpaused
	// +optional
	ScaleUpPreviewCheckPoint bool `json:"scaleUpPreviewCheckPoint,omitempty"`
}

// CanaryStatus status fields that only pertain to the canary rollout
type CanaryStatus struct {
	// StableRS indicates the last replicaset that walked through all the canary steps or was the only replicaset
	// +optional
	StableRS string `json:"stableRS,omitempty"`
}

// RolloutConditionType defines the conditions of Rollout
type RolloutConditionType string

// These are valid conditions of a rollout.
const (
	// InvalidSpec means the rollout has an invalid spec and will not progress until
	// the spec is fixed.
	InvalidSpec RolloutConditionType = "InvalidSpec"
	// RolloutAvailable means the rollout is available, ie. the active service is pointing at a
	// replicaset with the required replicas up and running for at least minReadySeconds.
	RolloutAvailable RolloutConditionType = "Available"
	// RolloutProgressing means the rollout is progressing. Progress for a rollout is
	// considered when a new replica set is created or adopted, when pods scale
	// up or old pods scale down, or when the services are updated. Progress is not estimated
	// for paused rollouts.
	RolloutProgressing RolloutConditionType = "Progressing"
	// RolloutReplicaFailure ReplicaFailure is added in a deployment when one of its pods
	// fails to be created or deleted.
	RolloutReplicaFailure RolloutConditionType = "ReplicaFailure"
)

// RolloutCondition describes the state of a rollout at a certain point.
type RolloutCondition struct {
	// Type of deployment condition.
	Type RolloutConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RolloutList is a list of Rollout resources
type RolloutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Rollout `json:"items"`
}

// Experiment is a specification for a Rollout resource
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Experiment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExperimentSpec   `json:"spec"`
	Status ExperimentStatus `json:"status,omitempty"`
}

// ExperimentSpec is the spec for a Experiment resource
type ExperimentSpec struct {
	// Templates A list of PodSpecs that define the ReplicaSets that should be run during an experiment.
	Templates []TemplateSpec `json:"templates"`
	// Duration the amount of time for the experiment to run. If not listed, the experiment will run for an
	// indefinite amount of time
	// +optional
	Duration *int32 `json:"duration,omitempty"`
	// ProgressDeadlineSeconds The maximum time in seconds for a experiment to
	// make progress before it is considered to be failed. Argo Rollouts will
	// continue to process failed experiments and a condition with a
	// ProgressDeadlineExceeded reason will be surfaced in the experiment status.
	// Note that progress will not be estimated during the time a experiment is paused.
	// Defaults to 600s.
	// +optional
	ProgressDeadlineSeconds *int32 `json:"progressDeadlineSeconds,omitempty"`
}

type TemplateSpec struct {
	// Name of the template used to identity replicaset running for this experiment
	Name string `json:"name"`
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`
	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this experiment.
	// It must match the pod template's labels. Each selector must be unique to the other selectors in the other templates
	Selector *metav1.LabelSelector `json:"selector"`
	// Template describes the pods that will be created.
	Template corev1.PodTemplateSpec `json:"template"`
}

// TemplateStatus is the status of a specific template of an Experiment
type TemplateStatus struct {
	// Name of the template used to identity which hash to compare to the hash
	Name string `json:"name"`
	// Total number of non-terminated pods targeted by this experiment (their labels match the selector).
	Replicas int32 `json:"replicas"`
	// Total number of non-terminated pods targeted by this experiment that have the desired template spec.
	UpdatedReplicas int32 `json:"updatedReplicas"`
	// Total number of ready pods targeted by this experiment.
	ReadyReplicas int32 `json:"readyReplicas"`
	// Total number of available pods (ready for at least minReadySeconds) targeted by this experiment.
	AvailableReplicas int32 `json:"availableReplicas"`
	// CollisionCount count of hash collisions for the Experiment. The Experiment controller uses this
	// field as a collision avoidance mechanism when it needs to create the name for the
	// newest ReplicaSet.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

// ExperimentStatus is the status for a Experiment resource
type ExperimentStatus struct {
	// TemplateStatuses the hash of the list of environment spec that is used to prevent changes in spec.
	// +optional
	TemplateStatuses []TemplateStatus `json:"templateStatuses,omitempty"`
	// The generation observed by the experiment controller by taking a hash of the spec.
	// +optional
	ObservedGeneration string `json:"observedGeneration,omitempty"`
	// Running indicates if the experiment has started. If the experiment is not running, the controller will
	// scale down all RS. If the running field isn't set, that means that the experiment hasn't started yet.
	// +optional
	Running *bool `json:"running,omitempty"`
	// AvailableAt the time when all the templates become healthy and the experiment should start tracking the time to
	// run for the duration of specificed in the spec.
	// +optional
	AvailableAt *metav1.Time `json:"availableAt,omitempty"`
	// Conditions a list of conditions a experiment can have.
	// +optional
	Conditions []ExperimentCondition `json:"conditions,omitempty"`
}

// ExperimentConditionType defines the conditions of Experiment
type ExperimentConditionType string

// These are valid conditions of a experiment.
const (
	// InvalidExperimentSpec means the experiment has an invalid spec and will not progress until
	// the spec is fixed.
	InvalidExperimentSpec ExperimentConditionType = "InvalidSpec"
	// ExperimentConcluded means the experiment is available, ie. the active service is pointing at a
	// replicaset with the required replicas up and running for at least minReadySeconds.
	ExperimentCompleted ExperimentConditionType = "Completed"
	// ExperimentProgressing means the experiment is progressing. Progress for a experiment is
	// considered when a new replica set is created or adopted, when pods scale
	// up or old pods scale down, or when the services are updated. Progress is not estimated
	// for paused experiment.
	ExperimentProgressing ExperimentConditionType = "Progressing"
	// ExperimentRunning means that an experiment has reached the desired state and is running for the duration
	// specified in the spec
	ExperimentRunning ExperimentConditionType = "Running"
	// ExperimentReplicaFailure ReplicaFailure is added in a experiment when one of its pods
	// fails to be created or deleted.
	ExperimentReplicaFailure ExperimentConditionType = "ReplicaFailure"
)

// ExperimentCondition describes the state of a rollout at a certain point.
type ExperimentCondition struct {
	// Type of deployment condition.
	Type ExperimentConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExperimentList is a list of Rollout resources
type ExperimentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Experiment `json:"items"`
}
