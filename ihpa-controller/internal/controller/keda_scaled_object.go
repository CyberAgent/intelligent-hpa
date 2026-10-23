package controller

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errUnsupportedMetricProvider = errors.New("scaleBackend KEDA supports only the prometheus metric provider")

// maxScaledObjectNameLength keeps the HPA that KEDA generates, keda-hpa-<ScaledObject name>, within 63 characters.
// KEDA's admission webhook rejects a longer name because the name is copied into a label value of the ScaledObject and the HPA.
// A hash-shortened name is not used because the name of the HPA is not shortened either, and a name that differs from ihpa-<name> is hard to trace back.
const maxScaledObjectNameLength = 63 - len("keda-hpa-")

var errScaledObjectNameTooLong = errors.New("name of the IntelligentHorizontalPodAutoscaler is too long for scaleBackend KEDA")

// ScaledObjectResource returns a ScaledObject whose triggers read the forecasted metrics,
// followed by the triggers of scaleBackend.keda.
func (g *ihpaGeneratorImpl) ScaledObjectResource() (*kedav1alpha1.ScaledObject, error) {
	if name := g.autoscalerName(); len(name) > maxScaledObjectNameLength {
		return nil, fmt.Errorf("%w: ScaledObject %q is %d characters long, must be no more than %d (the name of the IntelligentHorizontalPodAutoscaler must be no more than %d)",
			errScaledObjectNameTooLong, name, len(name), maxScaledObjectNameLength, maxScaledObjectNameLength-len("ihpa-"))
	}

	template := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec
	keda := g.ihpa.Spec.ScaleBackend.KEDA
	if keda == nil {
		keda = &ihpav1beta2.KEDAScaleBackend{}
	}

	triggers := make([]kedav1alpha1.ScaleTriggers, 0, len(template.Metrics)+len(keda.Triggers))
	for i := range template.Metrics {
		trigger, err := g.forecastTrigger(template.Metrics[i].MetricSpec())
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, *trigger)
	}
	for i := range keda.Triggers {
		triggers = append(triggers, *keda.Triggers[i].DeepCopy())
	}

	so := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.autoscalerName(),
			Namespace: g.ihpa.GetNamespace(),
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				APIVersion: template.ScaleTargetRef.APIVersion,
				Kind:       template.ScaleTargetRef.Kind,
				Name:       template.ScaleTargetRef.Name,
			},
			PollingInterval:       keda.PollingInterval,
			CooldownPeriod:        keda.CooldownPeriod,
			InitialCooldownPeriod: keda.InitialCooldownPeriod,
			IdleReplicaCount:      keda.IdleReplicaCount,
			MinReplicaCount:       firstNonNil(keda.MinReplicaCount, template.MinReplicas),
			MaxReplicaCount:       firstNonNil(keda.MaxReplicaCount, &template.MaxReplicas),
			Fallback:              keda.Fallback.DeepCopy(),
			Advanced:              keda.Advanced.DeepCopy(),
			Triggers:              triggers,
		},
	}

	addOwnerReference(&(g.ihpa.TypeMeta), &(g.ihpa.ObjectMeta), so)

	return so, nil
}

// forecastTrigger returns the trigger that reads the forecasted metric of the metric.
// Prometheus is the only metric provider that KEDA mode supports.
func (g *ihpaGeneratorImpl) forecastTrigger(metric *autoscalingv2.MetricSpec) (*kedav1alpha1.ScaleTriggers, error) {
	prometheus := g.ihpa.Spec.MetricProvider.Prometheus
	if prometheus == nil {
		return nil, errUnsupportedMetricProvider
	}

	ft, err := g.forecastTarget(metric)
	if err != nil {
		return nil, err
	}

	return &kedav1alpha1.ScaleTriggers{
		Type:       "prometheus",
		MetricType: autoscalingv2.AverageValueMetricType,
		Metadata: map[string]string{
			"serverAddress": prometheus.URL,
			"query":         prometheusQuery(ft),
			// The default of true turns an empty series into an inactive trigger,
			// which scales the workload in to the minimum even if the forecast is just not written yet.
			"ignoreNullValues": "false",
			// Quantity.String() may return a suffixed form such as "100m", which KEDA cannot parse as a number.
			"threshold": ft.AverageValue.AsDec().String(),
		},
	}, nil
}

// prometheusQuery selects the series of the forecasted metric as it is pushed to Pushgateway.
func prometheusQuery(ft *forecastTarget) string {
	keys := slices.Sorted(maps.Keys(ft.Labels))
	selectors := make([]string, 0, len(keys))
	for _, k := range keys {
		selectors = append(selectors, fmt.Sprintf("%s=%q", k, ft.Labels[k]))
	}

	return fmt.Sprintf("%s{%s}", strings.ReplaceAll(ft.MetricName, ".", "_"), strings.Join(selectors, ","))
}

// firstNonNil returns a copy of the first non-nil pointer.
func firstNonNil[T any](ps ...*T) *T {
	for _, p := range ps {
		if p != nil {
			v := *p
			return &v
		}
	}
	return nil
}
