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

// Send sends a metric to Prometheus via Pushgateway.
func (p *Prometheus) Send(metricName string, timestamp int64, point float64, tags []string, opts map[string]interface{}) error {
	if p.PushgatewayURL == "" {
		return fmt.Errorf("pushgatewayURL is not configured")
	}

	// Build the Pushgateway URL with job label
	pushURL := strings.TrimSuffix(p.PushgatewayURL, "/") + PushPath + "/job/estimator"

	// Build the metric body in Prometheus text format
	// Format: metric_name{tag1="value1",tag2="value2"} value timestamp
	labels := strings.Join(tags, ",")
	body := fmt.Sprintf("%s{%s} %f %d\n", metricName, labels, point, timestamp)

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

// Fetch queries a metric from Prometheus at the specified timestamp.
func (p *Prometheus) Fetch(metricName string, timestamp int64, tags []string, opts map[string]interface{}) (float64, error) {
	if p.URL == "" {
		return 0.0, fmt.Errorf("prometheus URL is not configured")
	}

	// Build the PromQL query: metric_name{tag1="value1",tag2="value2"}
	query := metricName
	if len(tags) > 0 {
		query = fmt.Sprintf("%s{%s}", metricName, strings.Join(tags, ","))
	}

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

// AddSumAggregator wraps the metric name with sum() for Prometheus queries.
func (p *Prometheus) AddSumAggregator(metricName string) string {
	return fmt.Sprintf("sum(%s)", metricName)
}
