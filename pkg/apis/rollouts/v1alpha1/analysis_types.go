package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AnalysisTemplate holds the template for performing canary analysis
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AnalysisTemplateSpec `json:"spec"`
}

// AnalysisTemplateList is a list of AnalysisTemplate resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AnalysisTemplate `json:"items"`
}

// AnalysisTemplateSpec is the specification for a AnalysisTemplate resource
type AnalysisTemplateSpec struct {
	// Metrics contains the list of metrics to query as part of an analysis run
	Metrics []AnalysisMetric `json:"metrics"`
}

// AnalysisMetric defines a metric in which to perform analysis
type AnalysisMetric struct {
	// Name is the name of the metric
	Name string `json:"name"`
	// Interval defines the interval in seconds between each metric analysis
	// If omitted, will perform the metric analysis only once
	Interval *int32 `json:"interval,omitempty"`
	// SuccessCondition is an expression which determines if a measurement is considered successful
	// Expression is a goevaluate expression. The keyword `result` is a variable reference to the
	// value of measurement. Results can be both structured data or primitive.
	// Examples:
	//   result > 10
	//   (result.requests_made * result.requests_succeeded / 100) >= 90
	//   result IN (red, yellow)
	SuccessCondition string `json:"successCondition,omitempty"`
	// FailureCondition is an expression which determines if a measurement is considered failed
	// If both success and failure conditions are specified, and the measurement does not fall into
	// either condition, the measurement is considered Inconclusive
	FailureCondition string `json:"failureCondition,omitempty"`
	// MaxFailures is the maximum number of times the measurement is allowed to fail, before the
	// entire metric is considered failed (default: 1)
	MaxFailures *int32 `json:"maxFailures,omitempty"`
	// FailFast will fail the entire analysis run prematurely
	FailFast bool `json:"failFast,omitempty"`
	// Provider configuration to the external system to use to verify the analysis
	Provider AnalysisProvider `json:"provider"`
}

// AnalysisProvider which external system to use to verify the analysis
// Only one of the fields in this struct should be non-nil
type AnalysisProvider struct {
	// PrometheusMetric specifies the prometheus metric to query
	Prometheus *PrometheusMetric `json:"prometheus,omitempty"`
}

// AnalysisStatus is the overall status of the AnalysisRun, MetricResults, or Measurement
type AnalysisStatus string

// AnalysisStatus is the overall status of the AnalysisRun, MetricResults
const (
	AnalysisStatusPending      AnalysisStatus = "Pending"
	AnalysisStatusRunning      AnalysisStatus = "Running"
	AnalysisStatusSuccessful   AnalysisStatus = "Successful"
	AnalysisStatusFailed       AnalysisStatus = "Failed"
	AnalysisStatusError        AnalysisStatus = "Error"
	AnalysisStatusInconclusive AnalysisStatus = "Inconclusive"
)

// PrometheusMetric defines the prometheus query to perform canary analysis
type PrometheusMetric struct {
	// Server is the address and port of the prometheus server
	Server string `json:"server,omitempty"`
	// Query is a raw prometheus query to perform
	Query string `json:"query,omitempty"`
}

// AnalysisRun is an instantiation of an AnalysisTemplate
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AnalysisRunSpec    `json:"spec"`
	Status            *AnalysisRunStatus `json:"status,omitempty"`
}

// AnalysisRunList is a list of AnalysisTemplate resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AnalysisRun `json:"items"`
}

// AnalysisRunSpec is the spec for a AnalysisRun resource
type AnalysisRunSpec struct {
	// AnalysisSpec holds the AnalysisSpec definition for performing analysis
	AnalysisSpec AnalysisTemplateSpec `json:"analysisSpec"`
	// Arguments hold the arguments to
	Arguments []Argument `json:"arguments,omitempty"`
	// ReplicaSets identifies the ReplicaSets in which to monitor to decide when to begin analysis
	ReplicaSets []string `json:"replicaSets,omitempty"`
}

// Argument is an argument to an AnalysisRun
type Argument struct {
	// Name is the name of the argument
	Name string `json:"name"`
	// Value is the value of the argument
	Value string `json:"value"`
}

// AnalysisRunStatus is the status for a AnalysisRun resource
type AnalysisRunStatus struct {
	// Status is the status of the analysis run
	Status AnalysisStatus `json:"status"`
	// Metrics contains the metrics collected during the run
	MetricResults map[string]MetricResult `json:"metricResults"`
}

// MetricResult contain a list of the most recent measurements for a single metric
type MetricResult struct {
	// Name is the name of the metric
	Name string `json:"name"`
	// Status is the overall aggregate status of the metric
	Status AnalysisStatus `json:"status"`
	// Measurements holds the most recent measurements collected for the metric
	Measurements []Measurement `json:"measurements"`
	// Failures counts the number of times the measurement was measured as a failure
	Failures *int32 `json:"failures"`
}

// Measurement is a point in time result value of a single metric, and the time it was measured
type Measurement struct {
	// Status is the status of this single measurement
	Status AnalysisStatus `json:"status"`
	// StartedAt is the timestamp in which this measurement started to be measured
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// FinishedAt is the timestamp in which this measurement completed and value was collected
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
	// Value is the measured value of the metric
	Value string `json:"value,omitempty"`
	// Metadata stores additional metadata about this metric result, used by the different providers
	// (e.g. kayenta run ID, job name)
	Metadata map[string]string `json:"metadata,omitempty"`
}
