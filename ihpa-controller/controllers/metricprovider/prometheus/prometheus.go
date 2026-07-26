package prometheus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider"
)

const (
	// QueryPath is the Prometheus query API endpoint.
	QueryPath = "/api/v1/query"
	// PushPath is the Pushgateway metrics endpoint.
	PushPath = "/metrics"
)

var (
	resourceMetricMap = map[string]metricIdentifier{
		"cpu":    {name: "container_cpu_usage_seconds_total", scale: -9},
		"memory": {name: "container_memory_working_set_bytes", scale: 0},
	}
	objectMetricMap = map[string]metricIdentifier{}
	podsMetricMap   = map[string]metricIdentifier{}
)

type metricIdentifier struct {
	name  string
	scale int
}

func (mi *metricIdentifier) GetName() string { return mi.name }
func (mi *metricIdentifier) GetScale() int   { return mi.scale }

// Prometheus implements the MetricProvider interface for Prometheus.
type Prometheus struct {
	URL            string `json:"url"`
	PushgatewayURL string `json:"pushgatewayUrl"`
	Aggregation    string `json:"aggregation,omitempty"`
}

type queryResponse struct {
	Status string `json:"status"`
	Data   queryData `json:"data"`
}

type queryData struct {
	ResultType string        `json:"resultType"`
	Result     []queryResult `json:"result"`
}

type queryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

// convertTagsToPrometheusFormat converts tags from "key:value" format
// (used internally by IHPA) to Prometheus label format "key=\"value\"".
func convertTagsToPrometheusFormat(tags []string) string {
	converted := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			converted = append(converted, fmt.Sprintf("%s=\"%s\"", parts[0], parts[1]))
		} else {
			// Already in Prometheus format or unknown format, keep as-is
			converted = append(converted, tag)
		}
	}
	return strings.Join(converted, ",")
}

// sanitizeMetricNameForPrometheus replaces dots with underscores to make
// the metric name valid in Prometheus (metric names must match [a-zA-Z_:][a-zA-Z0-9_:]*).
func sanitizeMetricNameForPrometheus(metricName string) string {
	return strings.ReplaceAll(metricName, ".", "_")
}

// Send sends a metric to Prometheus via Pushgateway.
func (p *Prometheus) Send(metricName string, timestamp int64, point float64, tags []string, opts map[string]interface{}) error {
	if p.PushgatewayURL == "" {
		return fmt.Errorf("pushgatewayURL is not configured")
	}

	// Build the Pushgateway URL with job label
	pushURL := strings.TrimSuffix(p.PushgatewayURL, "/") + PushPath + "/job/estimator"

	// Sanitize metric name for Prometheus (dots are not allowed in metric names)
	sanitizedMetricName := sanitizeMetricNameForPrometheus(metricName)

	// Build the metric body in Prometheus text format
	// Format: metric_name{tag1="value1",tag2="value2"} value
	// Note: Pushgateway does not accept timestamps in pushed metrics.
	labels := convertTagsToPrometheusFormat(tags)
	body := fmt.Sprintf("%s{%s} %f\n", sanitizedMetricName, labels, point)

	req, err := http.NewRequest(http.MethodPost, pushURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("pushgateway request error: %s (code=%d)", string(b), resp.StatusCode)
	}

	return nil
}

// buildPromQL constructs a valid PromQL query from a (possibly aggregated)
// metric name and label tags. Labels are applied to the innermost metric
// selector, correctly handling nested aggregators (e.g. "sum(rate(metric[5m]))")
// and range-vectors (e.g. "rate(metric[5m])").
func buildPromQL(metricName string, tags []string) string {
	if len(tags) == 0 {
		return metricName
	}
	labelSelector := fmt.Sprintf("{%s}", convertTagsToPrometheusFormat(tags))
	return insertLabels(metricName, labelSelector)
}

// insertLabels recursively finds the innermost metric selector and inserts
// the label selector before any range-vector suffix (e.g. [5m]).
func insertLabels(expr string, labelSelector string) string {
	// If the expression is wrapped in an aggregator (e.g. "sum(...)"),
	// recurse into the inner expression.
	if open := strings.Index(expr, "("); open >= 0 && strings.HasSuffix(expr, ")") {
		inner := expr[open+1 : len(expr)-1]
		return expr[:open+1] + insertLabels(inner, labelSelector) + ")"
	}
	// At the innermost metric selector. Insert labels before any range-vector suffix.
	if idx := strings.Index(expr, "["); idx >= 0 {
		return expr[:idx] + labelSelector + expr[idx:]
	}
	return expr + labelSelector
}

// Fetch queries a metric from Prometheus at the specified timestamp.
func (p *Prometheus) Fetch(metricName string, timestamp int64, tags []string, opts map[string]interface{}) (float64, error) {
	if p.URL == "" {
		return 0.0, fmt.Errorf("prometheus URL is not configured")
	}

	query := buildPromQL(metricName, tags)

	params := url.Values{}
	params.Set("query", query)
	params.Set("time", fmt.Sprintf("%d", timestamp))

	reqURL := strings.TrimSuffix(p.URL, "/") + QueryPath + "?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return 0.0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0.0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0.0, err
		}
		return 0.0, fmt.Errorf("prometheus query error: %s (code=%d)", string(b), resp.StatusCode)
	}

	var qr queryResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return 0.0, fmt.Errorf("failed to decode response: %w", err)
	}

	if qr.Status != "success" {
		return 0.0, fmt.Errorf("prometheus query returned status: %s", qr.Status)
	}

	if len(qr.Data.Result) == 0 {
		return 0.0, nil
	}

	// Sum up all series values
	var total float64
	for _, r := range qr.Data.Result {
		if len(r.Value) < 2 {
			continue
		}
		switch v := r.Value[1].(type) {
		case float64:
			total += v
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			total += f
		}
	}

	return total, nil
}

// ConvertResourceMetricName converts a resource metric name to Prometheus-specific name.
func (p *Prometheus) ConvertResourceMetricName(metricName string, reverse bool) metricprovider.MetricIdentifier {
	if !reverse {
		if v, ok := resourceMetricMap[metricName]; ok {
			return &v
		}
	} else {
		for k, v := range resourceMetricMap {
			if v.name == metricName {
				return &metricIdentifier{name: k, scale: v.scale}
			}
		}
	}
	return nil
}

// ConvertObjectMetricName converts an object metric name to Prometheus-specific name.
func (p *Prometheus) ConvertObjectMetricName(metricName string, reverse bool) metricprovider.MetricIdentifier {
	if !reverse {
		if v, ok := objectMetricMap[metricName]; ok {
			return &v
		}
	} else {
		for k, v := range objectMetricMap {
			if v.name == metricName {
				return &metricIdentifier{name: k, scale: v.scale}
			}
		}
	}
	return nil
}

// ConvertPodsMetricName converts a pods metric name to Prometheus-specific name.
func (p *Prometheus) ConvertPodsMetricName(metricName string, reverse bool) metricprovider.MetricIdentifier {
	if !reverse {
		if v, ok := podsMetricMap[metricName]; ok {
			return &v
		}
	} else {
		for k, v := range podsMetricMap {
			if v.name == metricName {
				return &metricIdentifier{name: k, scale: v.scale}
			}
		}
	}
	return nil
}

// AddAggregator wraps the metric name with the configured aggregation function for Prometheus queries.
func (p *Prometheus) AddAggregator(metricName string) string {
	aggregation := p.Aggregation
	if aggregation == "" {
		aggregation = "sum"
	}
	return fmt.Sprintf("%s(%s)", aggregation, metricName)
}
