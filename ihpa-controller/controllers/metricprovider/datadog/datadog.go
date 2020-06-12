package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider"
)

const (
	BaseURL               = "https://api.datadoghq.com"
	TimeseriesSeriesPath  = "/api/v1/series"
	TimeseriesQueryPath   = "/api/v1/query"
	TimeseriesMetricsPath = "/api/v1/metrics"
)

var (
	resourceMetricMap = map[string]metricIdentifier{
		"cpu":    {name: "kubernetes.cpu.usage.total", scale: -9},
		"memory": {name: "kubernetes.memory.usage", scale: 0},
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

type Datadog struct {
	APIKey string `json:"apikey"`
	APPKey string `json:"appkey"`
}

type datapoint struct {
	unixtime int64
	point    float64
}

type timeseriesItem struct {
	Metric     string          `json:"metric"`
	Points     []datapoint     `json:"-"`
	TmpPoints  [][]interface{} `json:"points"`
	MetricType string          `json:"type,omitempty"`
	Tags       []string        `json:"tags"`
}

type timeseries struct {
	TimeseriesItems []timeseriesItem `json:"series"`
}

// MarshalJSON convert array of datapoint to array of interface{} as proprocess.
func (ts *timeseriesItem) MarshalJSON() ([]byte, error) {
	ts.TmpPoints = make([][]interface{}, len(ts.Points))
	for i, dp := range ts.Points {
		ts.TmpPoints[i] = []interface{}{dp.unixtime, dp.point}
	}

	// avoid cyclic call
	type T timeseriesItem
	return json.Marshal(&struct{ T }{T: T(*ts)})
}

func (d *Datadog) Send(metricName string, timestamp int64, point float64, tags []string, opts map[string]interface{}) error {
	var metricUnitReference string
	if v, ok := opts["metricUnitReference"]; ok {
		metricUnitReference = v.(string)
	}
	return d.send(BaseURL, metricName, timestamp, point, tags, metricUnitReference)
}

func (d *Datadog) send(baseurl string, metricName string, timestamp int64, point float64, tags []string, metricUnitReference string) error {
	ts := timeseries{
		TimeseriesItems: []timeseriesItem{
			{
				Metric: metricName,
				Points: []datapoint{
					{
						unixtime: timestamp,
						point:    point,
					},
				},
				Tags: tags,
			},
		},
	}

	tsjson, err := json.Marshal(&ts)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s?api_key=%s", baseurl, TimeseriesSeriesPath, d.APIKey)

	resp, err := http.Post(url, "application/json", bytes.NewReader(tsjson))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Request error: %s (code=%d, json=%s)", string(b), resp.StatusCode, string(tsjson))
	}

	// edit unit metadata of metric
	// If don't do this, hpa interpret the metric as wrong unit.
	// For example, cpu metric stored as nanocore, 100,000,000 is 0.1 core,
	// but you don't set unit, hpa interpret as 100,000,000 core.
	if metricUnitReference != "" {
		mtype, munit, err := d.getUnit(baseurl, metricUnitReference)
		if err != nil {
			return err
		}
		if err := d.setUnit(baseurl, metricName, mtype, munit); err != nil {
			return err
		}
	}

	return nil
}

func (d *Datadog) getUnit(baseurl, metricName string) (string, string, error) {
	url := fmt.Sprintf("%s%s/%s", baseurl, TimeseriesMetricsPath, metricName)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("DD-API-KEY", d.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", d.APPKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", "", err
		}
		return "", "", fmt.Errorf("Request error: %s (code=%d, metricName=%s)", string(b), resp.StatusCode, metricName)
	}

	var j interface{}
	if err = json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return "", "", fmt.Errorf("decode failed: %w", err)
	}

	var mtype, munit string
	if v, ok := j.(map[string]interface{})["type"]; ok {
		if v != nil {
			mtype = v.(string)
		}
	}
	if v, ok := j.(map[string]interface{})["unit"]; ok {
		if v != nil {
			munit = v.(string)
		}
	}

	return mtype, munit, nil
}

func (d *Datadog) setUnit(baseurl, metricName string, metricType string, metricUnit string) error {
	url := fmt.Sprintf("%s%s/%s", baseurl, TimeseriesMetricsPath, metricName)

	body := fmt.Sprintf(`{"type":"%s","short_name":"%s","unit":"%s"}`, metricType, metricUnit, metricUnit)

	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("DD-API-KEY", d.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", d.APPKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Request error: %s (code=%d, body=%s)", string(b), resp.StatusCode, body)
	}

	return nil
}

func (d *Datadog) Fetch(metricName string, timestamp int64, tags []string, opts map[string]interface{}) (float64, error) {
	dp, err := d.fetch(BaseURL, metricName, timestamp, tags)
	if err != nil {
		return 0.0, err
	}
	return dp.point, nil
}

func (d *Datadog) fetch(baseurl, metricName string, timestamp int64, tags []string) (datapoint, error) {
	// check around 10 minutes
	// NOTE: datadog timestamp is msec scale
	margin := int64(10)
	fromts := timestamp - 60*margin
	tots := timestamp + 60*margin

	url := fmt.Sprintf("%s%s?query=%s{%s}by{host}&from=%d&to=%d",
		baseurl,
		TimeseriesQueryPath,
		metricName,
		strings.Join(tags, ","),
		fromts,
		tots,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return datapoint{}, err
	}
	req.Header.Set("DD-API-KEY", d.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", d.APPKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return datapoint{}, err
	}
	if resp.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return datapoint{}, err
		}
		return datapoint{}, fmt.Errorf("Request error: %s (code=%d, url=%s)", string(b), resp.StatusCode, url)
	}

	var j interface{}
	if err = json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return datapoint{}, err
	}

	sortedDps := mergeAllSeriesDatapointToOne(j)

	//var debugStr string
	//for _, dp := range sortedDps {
	//	t := time.Unix(dp.unixtime, 0).String()
	//	s := strconv.FormatFloat(dp.point, 'f', 3, 64)
	//	debugStr += fmt.Sprintf("{%s (%s)} ", t, s)
	//}
	//log.Printf("%s", debugStr)

	return binarySearchNearTimestamp(sortedDps, timestamp), nil
}

// binarySearchNearTimestamp search most near datapoint by specified timestamp.
// NOTE: dps must be sorted.
func binarySearchNearTimestamp(dps []datapoint, timestamp int64) datapoint {
	if len(dps) == 1 {
		return dps[0]
	}
	if dp := dps[len(dps)/2]; dp.unixtime > timestamp {
		return binarySearchNearTimestamp(dps[:len(dps)/2], timestamp)
	}
	return binarySearchNearTimestamp(dps[len(dps)/2:], timestamp)
}

// mergeAllSeriesDatapointToOne merge all series datapoints to single datapoints (sum up).
func mergeAllSeriesDatapointToOne(j interface{}) []datapoint {
	var dps []datapoint
	if series, ok := j.(map[string]interface{})["series"]; ok {
		// 20min duration query -> jq ".series[0].pointlist | length" -> 229
		dps = make([]datapoint, 0, 300*len(series.([]interface{})))

		for _, s := range series.([]interface{}) {
			if pointlist, ok := s.(map[string]interface{})["pointlist"]; ok {
				for _, p := range pointlist.([]interface{}) {
					tsPtPair, ok := p.([]interface{})
					if len(tsPtPair) < 2 || !ok {
						continue
					}
					ddts, ok1 := tsPtPair[0].(float64)
					ddpt, ok2 := tsPtPair[1].(float64)
					if !ok1 || !ok2 {
						continue
					}
					dp := datapoint{
						unixtime: int64(ddts / 1000.0),
						point:    ddpt,
					}
					dps = append(dps, dp)
				}
			}
		}
	} else {
		return nil
	}

	sort.Slice(dps, func(i, j int) bool {
		return dps[i].unixtime < dps[j].unixtime
	})

	mergedDps := make([]datapoint, 300)
	mergedDps[0].unixtime = dps[0].unixtime
	var idx int
	for _, dp := range dps {
		if mergedDps[idx].unixtime == dp.unixtime {
			mergedDps[idx].point += dp.point
		} else {
			idx++
			mergedDps[idx] = dp
		}
	}

	return mergedDps[:idx+1]
}

func (d *Datadog) ConvertResourceMetricName(metricName string, reverse bool) metricprovider.MetricIdentifier {
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

func (d *Datadog) ConvertObjectMetricName(metricName string, reverse bool) metricprovider.MetricIdentifier {
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

func (d *Datadog) ConvertPodsMetricName(metricName string, reverse bool) metricprovider.MetricIdentifier {
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

func (d *Datadog) AddSumAggregator(metricName string) string {
	return "sum:" + metricName
}
