package prometheus

import "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider"

var (
	resourceMetricMap = map[string]metricIdentifier{}
	objectMetricMap   = map[string]metricIdentifier{}
	podsMetricMap     = map[string]metricIdentifier{}
)

type metricIdentifier struct {
	name  string
	scale int
}

func (mi *metricIdentifier) GetName() string { return mi.name }
func (mi *metricIdentifier) GetScale() int   { return mi.scale }

type Prometheus struct{}

func (p *Prometheus) Send(metricName string, timestamp int64, point float64, tags []string, opts map[string]interface{}) error {
	return nil
}

func (p *Prometheus) Fetch(metricName string, timestamp int64, tags []string, opts map[string]interface{}) (float64, error) {
	return 0.0, nil
}

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

func (p *Prometheus) AddSumAggregator(metricName string) string {
	return metricName
}
