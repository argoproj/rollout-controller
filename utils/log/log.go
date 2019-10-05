package log

import (
	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
)

const (
	// RolloutKey defines the key for the rollout field
	RolloutKey = "rollout"
	// ExperimentKey defines the key for the experiment field
	ExperimentKey = "experiment"
	// AnalysisRunKey defines the key for the analysisrun field
	AnalysisRunKey = "analysisrun"
	// ServiceKey defines the key for the service field
	ServiceKey = "service"
	// JobKey defines the key for the job field
	JobKey = "job"
	// NamespaceKey defines the key for the namespace field
	NamespaceKey = "namespace"
)

// WithRollout returns a logging context with the rollout field set
func WithRollout(rollout *v1alpha1.Rollout) *log.Entry {
	return log.WithField(RolloutKey, rollout.Name).WithField(NamespaceKey, rollout.Namespace)
}

func WithExperiment(experiment *v1alpha1.Experiment) *log.Entry {
	return log.WithField(ExperimentKey, experiment.Name).WithField(NamespaceKey, experiment.Namespace)
}

func WithAnalysisRun(ar *v1alpha1.AnalysisRun) *log.Entry {
	return log.WithField(AnalysisRunKey, ar.Name).WithField(NamespaceKey, ar.Namespace)
}
