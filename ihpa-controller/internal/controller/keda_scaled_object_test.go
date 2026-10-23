package controller

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testKEDAGenerator(t *testing.T) *ihpaGeneratorImpl {
	t.Helper()
	sample1, _ := testIHPAGeneratorSample(t)

	ihpa := sample1.ihpa.DeepCopy()
	ihpa.APIVersion = "ihpa.ake.cyberagent.co.jp/v1beta2"
	ihpa.Spec.MetricProvider = ihpav1beta2.MetricProvider{
		Name:           "prometheus",
		ProviderSource: ihpav1beta2.ProviderSource{Prometheus: &ihpav1beta2.PrometheusProviderSource{URL: "http://prometheus-server.monitoring.svc.cluster.local:80"}},
	}
	template := &ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec
	template.MinReplicas = new(int32(1))
	template.MaxReplicas = 5
	template.Metrics = []ihpav1beta2.ExtendedMetricSpec{
		{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name:   "cpu",
				Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: new(int32(50))},
			},
		},
		{
			Type: autoscalingv2.ExternalMetricSourceType,
			External: &autoscalingv2.ExternalMetricSource{
				Metric: autoscalingv2.MetricIdentifier{Name: "nginx.net.request_per_s"},
				Target: autoscalingv2.MetricTarget{Type: autoscalingv2.AverageValueMetricType, AverageValue: resource.NewQuantity(10, resource.DecimalSI)},
			},
		},
	}
	ihpa.Spec.ScaleBackend = ihpav1beta2.ScaleBackend{Type: ihpav1beta2.ScaleBackendKEDA}

	return &ihpaGeneratorImpl{
		ihpa:                ihpa,
		kubeSystemUID:       sample1.kubeSystemUID,
		scaleTargetRequests: corev1.ResourceList{corev1.ResourceCPU: *resource.NewScaledQuantity(500, resource.Milli)},
	}
}

func TestScaledObjectResource(t *testing.T) {
	t.Parallel()

	t.Run("success: builds the ScaledObject from the IHPA", func(t *testing.T) {
		t.Parallel()

		g := testKEDAGenerator(t)
		g.ihpa.Spec.ScaleBackend.KEDA = &ihpav1beta2.KEDAScaleBackend{
			PollingInterval:       new(int32(30)),
			CooldownPeriod:        new(int32(300)),
			InitialCooldownPeriod: new(int32(60)),
			MinReplicaCount:       new(int32(2)),
			Fallback:              &kedav1alpha1.Fallback{FailureThreshold: 3, Replicas: 4},
			Triggers: []kedav1alpha1.ScaleTriggers{
				{Type: "cpu", MetricType: autoscalingv2.UtilizationMetricType, Metadata: map[string]string{"value": "50"}},
			},
		}

		got, err := g.ScaledObjectResource()
		if err != nil {
			t.Fatal(err)
		}

		const selector = `{kube_deployment="nginx",kube_namespace="default",kube_system_uid="46f9e396-d3c4-4103-a807-49054f47bbfb"}`
		expected := &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ihpa-sample1",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta2",
					Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
					Name:               "sample1",
					UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
					Controller:         new(true),
					BlockOwnerDeletion: new(true),
				}},
			},
			Spec: kedav1alpha1.ScaledObjectSpec{
				ScaleTargetRef:        &kedav1alpha1.ScaleTarget{APIVersion: "apps/v1", Kind: "Deployment", Name: "nginx"},
				PollingInterval:       new(int32(30)),
				CooldownPeriod:        new(int32(300)),
				InitialCooldownPeriod: new(int32(60)),
				MinReplicaCount:       new(int32(2)),
				MaxReplicaCount:       new(int32(5)),
				Fallback:              &kedav1alpha1.Fallback{FailureThreshold: 3, Replicas: 4},
				Triggers: []kedav1alpha1.ScaleTriggers{
					{
						Type:       "prometheus",
						MetricType: autoscalingv2.AverageValueMetricType,
						Metadata: map[string]string{
							"serverAddress":    "http://prometheus-server.monitoring.svc.cluster.local:80",
							"query":            "ake_ihpa_forecasted_container_cpu_usage_seconds_total" + selector,
							"threshold":        "250000000",
							"ignoreNullValues": "false",
						},
					},
					{
						Type:       "prometheus",
						MetricType: autoscalingv2.AverageValueMetricType,
						Metadata: map[string]string{
							"serverAddress":    "http://prometheus-server.monitoring.svc.cluster.local:80",
							"query":            "ake_ihpa_forecasted_nginx_net_request_per_s" + selector,
							"threshold":        "10",
							"ignoreNullValues": "false",
						},
					},
					{Type: "cpu", MetricType: autoscalingv2.UtilizationMetricType, Metadata: map[string]string{"value": "50"}},
				},
			},
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("ScaledObjectResource() = %#v, want %#v", got, expected)
		}
	})

	t.Run("success: does not share the options with the IHPA", func(t *testing.T) {
		t.Parallel()

		g := testKEDAGenerator(t)
		g.ihpa.Spec.ScaleBackend.KEDA = &ihpav1beta2.KEDAScaleBackend{
			Fallback: &kedav1alpha1.Fallback{FailureThreshold: 3, Replicas: 4},
			Advanced: &kedav1alpha1.AdvancedConfig{RestoreToOriginalReplicaCount: true},
			Triggers: []kedav1alpha1.ScaleTriggers{
				{Type: "cpu", MetricType: autoscalingv2.UtilizationMetricType, Metadata: map[string]string{"value": "50"}},
			},
		}

		got, err := g.ScaledObjectResource()
		if err != nil {
			t.Fatal(err)
		}
		got.Spec.Fallback.Replicas = 99
		got.Spec.Advanced.RestoreToOriginalReplicaCount = false
		got.Spec.Triggers[len(got.Spec.Triggers)-1].Metadata["value"] = "99"

		keda := g.ihpa.Spec.ScaleBackend.KEDA
		if keda.Fallback.Replicas != 4 {
			t.Errorf("fallback replicas of the IHPA = %d, want 4", keda.Fallback.Replicas)
		}
		if !keda.Advanced.RestoreToOriginalReplicaCount {
			t.Error("advanced of the IHPA is changed through the ScaledObject")
		}
		if keda.Triggers[0].Metadata["value"] != "50" {
			t.Errorf("trigger metadata of the IHPA = %q, want 50", keda.Triggers[0].Metadata["value"])
		}
	})

	t.Run("fail: the metric provider is not Prometheus", func(t *testing.T) {
		t.Parallel()

		g := testKEDAGenerator(t)
		g.ihpa.Spec.MetricProvider = ihpav1beta2.MetricProvider{
			Name:           "datadog",
			ProviderSource: ihpav1beta2.ProviderSource{Datadog: &ihpav1beta2.DatadogProviderSource{APIKey: "xxx", APPKey: "yyy"}},
		}

		if _, err := g.ScaledObjectResource(); !errors.Is(err, errUnsupportedMetricProvider) {
			t.Errorf("ScaledObjectResource() error = %v, want %v", err, errUnsupportedMetricProvider)
		}
	})
}

func TestScaledObjectResource_NameLength(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ihpaName string
		wantErr  bool
	}{
		// ihpa-<name> is 54 characters, so keda-hpa-ihpa-<name> is exactly 63 characters.
		"success: the generated HPA name is 63 characters":   {ihpaName: strings.Repeat("a", 49)},
		"fail: the generated HPA name exceeds 63 characters": {ihpaName: strings.Repeat("a", 50), wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := testKEDAGenerator(t)
			g.ihpa.Name = tt.ihpaName

			_, err := g.ScaledObjectResource()
			if got := errors.Is(err, errScaledObjectNameTooLong); got != tt.wantErr {
				t.Errorf("ScaledObjectResource() error = %v, want errScaledObjectNameTooLong: %t", err, tt.wantErr)
			}
		})
	}
}

func TestScaledObjectResource_ReplicaCounts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		minReplicas *int32
		keda        *ihpav1beta2.KEDAScaleBackend
		wantMin     *int32
		wantMax     int32
	}{
		"success: from the template":                       {minReplicas: new(int32(1)), wantMin: new(int32(1)), wantMax: 5},
		"success: keda overrides the template":             {minReplicas: new(int32(1)), keda: &ihpav1beta2.KEDAScaleBackend{MinReplicaCount: new(int32(0)), MaxReplicaCount: new(int32(8))}, wantMin: new(int32(0)), wantMax: 8},
		"success: unspecified minReplicas is left to KEDA": {wantMin: nil, wantMax: 5},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := testKEDAGenerator(t)
			g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.MinReplicas = tt.minReplicas
			g.ihpa.Spec.ScaleBackend.KEDA = tt.keda

			got, err := g.ScaledObjectResource()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Spec.MinReplicaCount, tt.wantMin) {
				t.Errorf("MinReplicaCount = %v, want %v", got.Spec.MinReplicaCount, tt.wantMin)
			}
			if got.Spec.MaxReplicaCount == nil || *got.Spec.MaxReplicaCount != tt.wantMax {
				t.Errorf("MaxReplicaCount = %v, want %d", got.Spec.MaxReplicaCount, tt.wantMax)
			}
		})
	}
}

func TestScaledObjectResource_ForecastTriggers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		metric        ihpav1beta2.ExtendedMetricSpec
		wantQueryName string
		wantThreshold string
	}{
		"success: fractional average value keeps the decimal": {
			metric: ihpav1beta2.ExtendedMetricSpec{
				Type: autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{Name: "nginx.net.request_per_s"},
					Target: autoscalingv2.MetricTarget{Type: autoscalingv2.AverageValueMetricType, AverageValue: resource.NewScaledQuantity(40, resource.Milli)},
				},
			},
			wantQueryName: "ake_ihpa_forecasted_nginx_net_request_per_s",
			wantThreshold: "0.040",
		},
		"success: memory utilization is converted by the requests": {
			metric: ihpav1beta2.ExtendedMetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name:   "memory",
					Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: new(int32(50))},
				},
			},
			wantQueryName: "ake_ihpa_forecasted_container_memory_working_set_bytes",
			wantThreshold: "134217728",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := testKEDAGenerator(t)
			g.scaleTargetRequests = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}
			g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics = []ihpav1beta2.ExtendedMetricSpec{tt.metric}

			got, err := g.ScaledObjectResource()
			if err != nil {
				t.Fatal(err)
			}

			if len(got.Spec.Triggers) != 1 {
				t.Fatalf("triggers = %d, want 1", len(got.Spec.Triggers))
			}
			metadata := got.Spec.Triggers[0].Metadata
			if !strings.HasPrefix(metadata["query"], tt.wantQueryName+"{") {
				t.Errorf("query = %q, want the metric %q", metadata["query"], tt.wantQueryName)
			}
			if metadata["threshold"] != tt.wantThreshold {
				t.Errorf("threshold = %q, want %q", metadata["threshold"], tt.wantThreshold)
			}
		})
	}
}

func TestPrometheusQuery(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		target   forecastTarget
		expected string
	}{
		"success: labels are sorted by key": {
			target:   forecastTarget{MetricName: "ake.ihpa.forecasted_x", Labels: map[string]string{"b": "2", "a": "1", "c": "3"}},
			expected: `ake_ihpa_forecasted_x{a="1",b="2",c="3"}`,
		},
		"success: a quote in a value is escaped": {
			target:   forecastTarget{MetricName: "m", Labels: map[string]string{"a": `x"y`}},
			expected: `m{a="x\"y"}`,
		},
		"success: no labels": {
			target:   forecastTarget{MetricName: "m"},
			expected: "m{}",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := prometheusQuery(&tt.target); got != tt.expected {
				t.Errorf("prometheusQuery() = %q, want %q", got, tt.expected)
			}
		})
	}
}
