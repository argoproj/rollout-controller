package datadog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/utils/evaluate"
	metricutil "github.com/argoproj/argo-rollouts/utils/metric"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/jsonpath"
)

var unixNow = func() int64 { return time.Now().Unix() }

const (
	//ProviderType indicates the provider is datadog
	ProviderType            = "Datadog"
	DatadogTokensSecretName = "datadog"
)

// Provider contains all the required components to run a Datadog query
// Implements the Provider Interface
type Provider struct {
	logCtx log.Entry
	apiKey string
	appKey string
}

// Type incidates provider is a Datadog provider
func (p *Provider) Type() string {
	return ProviderType
}

func (p *Provider) Run(run *v1alpha1.AnalysisRun, metric v1alpha1.Metric) v1alpha1.Measurement {
	startTime := metav1.Now()

	// Measurement to pass back
	measurement := v1alpha1.Measurement{
		StartedAt: &startTime,
	}

	endpoint := "https://api.datadoghq.com/api/v1/query"
	if metric.Provider.Datadog.Address != "" {
		endpoint = metric.Provider.Datadog.Address + "/api/v1/query"
	}

	url, _ := url.Parse(endpoint)

	now := unixNow()
	var interval int64 = 300
	if metric.Provider.Datadog.Interval != "" {
		expDuration, err := metric.Provider.Datadog.Interval.Duration()
		if err != nil {
			return metricutil.MarkMeasurementError(measurement, err)
		}
		// Convert to seconds as DataDog expects unix timestamp
		interval = int64(expDuration.Seconds())
	}

	q := url.Query()
	q.Set("query", metric.Provider.Datadog.Query)
	q.Set("from", strconv.FormatInt(now-interval, 10))
	q.Set("to", strconv.FormatInt(now, 10))
	url.RawQuery = q.Encode()

	request := &http.Request{Method: "GET"}
	request.URL = url
	request.Header = make(http.Header)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("DD-API-KEY", p.apiKey)
	request.Header.Set("DD-APPLICATION-KEY", p.appKey)

	// Send Request
	httpClient := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	response, err := httpClient.Do(request)

	if err != nil {
		return metricutil.MarkMeasurementError(measurement, err)
	}

	value, status, err := p.parseResponse(metric, response)
	if err != nil {
		return metricutil.MarkMeasurementError(measurement, err)
	}

	measurement.Value = value
	measurement.Phase = status
	finishedTime := metav1.Now()
	measurement.FinishedAt = &finishedTime

	return measurement
}

func (p *Provider) parseResponse(metric v1alpha1.Metric, response *http.Response) (string, v1alpha1.AnalysisPhase, error) {
	var data interface{}

	jsonParser := jsonpath.New("metrics")
	jsonParser.Parse("{.series[0].pointlist[-1:][1]}")

	bodyBytes, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("Received no bytes in response: %v", err)
	}

	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusUnauthorized {
		return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("received authentication error response code: %v %s", response.StatusCode, string(bodyBytes))
	} else if response.StatusCode != http.StatusOK {
		return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("received non 2xx response code: %v %s", response.StatusCode, string(bodyBytes))
	}

	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("Could not parse JSON body: %v", err)
	}

	results, err := jsonParser.FindResults(data)
	if err != nil {
		return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("Could not find JSONPath in body: %s", err)
	}

	if len(results) < 1 && len(results[0]) < 1 {
		return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("Datadog returned no value")
	}

	var out float64 = results[0][0].Interface().(float64)

	status := evaluate.EvaluateResult(out, metric, p.logCtx)
	return strconv.FormatFloat(out, 'f', -1, 64), status, nil
}

// Resume should not be used the Datadog provider since all the work should occur in the Run method
func (p *Provider) Resume(run *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	p.logCtx.Warn("Datadog provider should not execute the Resume method")
	return measurement
}

// Terminate should not be used the Datadog provider since all the work should occur in the Run method
func (p *Provider) Terminate(run *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	p.logCtx.Warn("Datadog provider should not execute the Terminate method")
	return measurement
}

// GarbageCollect is a no-op for the Datadog provider
func (p *Provider) GarbageCollect(run *v1alpha1.AnalysisRun, metric v1alpha1.Metric, limit int) error {
	return nil
}

func NewDatadogProvider(logCtx log.Entry, kubeclientset kubernetes.Interface) (*Provider, error) {
	ns := Namespace()
	secret, err := kubeclientset.CoreV1().Secrets(ns).Get(DatadogTokensSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	apiKey := string(secret.Data["api-key"])
	appKey := string(secret.Data["app-key"])

	if apiKey != "" && appKey != "" {
		return &Provider{
			logCtx: logCtx,
			apiKey: apiKey,
			appKey: appKey,
		}, nil
	} else {
		return nil, errors.New("API or App token not found")
	}

}

func Namespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns
	}
	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return "argo-rollouts"
}
