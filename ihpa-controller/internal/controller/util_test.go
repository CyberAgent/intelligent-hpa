package controller

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	ihpav1beta1 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSanitizeForKubernetesResourceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "ihpa-nginx",
			expected: "ihpa-nginx",
		},
		{
			input:    "ihpa-nginx-nginx.net.request_per_s",
			expected: "ihpa-nginx-nginx-net-request-per-s",
		},
	}

	for _, tt := range tests {
		got := sanitizeForKubernetesResourceName(tt.input)
		if got != tt.expected {
			t.Fatalf("sanitized string is not match (got=%s, expected=%s)", got, tt.expected)
		}
	}
}

func TestCorrespondForecastedMetricName(t *testing.T) {
	const cpuValue2 = "cpu"
	prefix := "ake.ihpa.forecasted_"
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    cpuValue2,
			expected: prefix + cpuValue2,
		},
		{
			input:    "nginx.net.request_per_s",
			expected: prefix + "nginx_net_request_per_s",
		},
		{
			input:    "requests-per-second",
			expected: prefix + "requests_per_second",
		},
		{
			input:    "test.sample-metric",
			expected: prefix + "test_sample_metric",
		},
	}

	for _, tt := range tests {
		got := correspondForecastedMetricName(tt.input)
		if got != tt.expected {
			t.Fatalf("sanitized metric name is not match (got=%s, expected=%s)", got, tt.expected)
		}
	}
}

func TestReplaceStrings(t *testing.T) {
	const helloValue = "hello"
	tests := []struct {
		input    string
		m        map[string]string
		expected string
	}{
		{
			input:    helloValue,
			m:        map[string]string{"h": "b"},
			expected: "bello",
		},
		{
			input:    "_,.-=_,.-=",
			m:        map[string]string{",": "=", "_": ""},
			expected: "=.-==.-=",
		},
		{
			input:    helloValue,
			m:        map[string]string{"a": "b"},
			expected: helloValue,
		},
	}

	for _, tt := range tests {
		got := replaceStrings(tt.input, tt.m)
		if got != tt.expected {
			t.Fatalf("result is not match (got=%s, expected=%s)", got, tt.expected)
		}
	}
}

func TestExtractScopedMetricInfo(t *testing.T) {
	const testSampleMetric = "test.sample-metric"
	const requestsPerSecond = "requests-per-second"
	const cpuValue = "cpu"
	const utilization = "Utilization"
	const object = "Object"
	const averageValue = "AverageValue"
	tests := []struct {
		metric    autoscalingv2.MetricSpec
		expected  string
		expTarget *autoscalingv2.MetricTarget
	}{
		{
			metric: autoscalingv2.MetricSpec{
				Type: "Resource",
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: cpuValue,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.MetricTargetType(utilization),
						AverageUtilization: func(i int32) *int32 { return &i }(50),
					},
				},
			},
			expected: cpuValue,
			expTarget: &autoscalingv2.MetricTarget{
				Type:               autoscalingv2.MetricTargetType(utilization),
				AverageUtilization: func(i int32) *int32 { return &i }(50),
			},
		},
		{
			metric: autoscalingv2.MetricSpec{
				Type: object,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: requestsPerSecond,
					},
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.MetricTargetType(utilization),
						AverageUtilization: func(i int32) *int32 { return &i }(60),
					},
				},
			},
			expected: requestsPerSecond,
			expTarget: &autoscalingv2.MetricTarget{
				Type:               autoscalingv2.MetricTargetType(utilization),
				AverageUtilization: func(i int32) *int32 { return &i }(60),
			},
		},
		{
			metric: autoscalingv2.MetricSpec{
				Type: "Pods",
				Pods: &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "packets-per-second",
					},
					Target: autoscalingv2.MetricTarget{
						Type:  autoscalingv2.MetricTargetType("Value"),
						Value: resource.NewQuantity(100, "test1"),
					},
				},
			},
			expected: "packets-per-second",
			expTarget: &autoscalingv2.MetricTarget{
				Type:  autoscalingv2.MetricTargetType("Value"),
				Value: resource.NewQuantity(100, "test1"),
			},
		},
		{
			metric: autoscalingv2.MetricSpec{
				Type: "External",
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: testSampleMetric,
					},
					Target: autoscalingv2.MetricTarget{
						Type:         autoscalingv2.MetricTargetType(averageValue),
						AverageValue: resource.NewQuantity(20, "test2"),
					},
				},
			},
			expected: testSampleMetric,
			expTarget: &autoscalingv2.MetricTarget{
				Type:         autoscalingv2.MetricTargetType(averageValue),
				AverageValue: resource.NewQuantity(20, "test2"),
			},
		},
		{
			metric: autoscalingv2.MetricSpec{
				Type: object,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: "resource_metrics",
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.MetricTargetType(utilization),
						AverageUtilization: func(i int32) *int32 { return &i }(50),
					},
				},
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "object_metrics",
					},
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.MetricTargetType(utilization),
						AverageUtilization: func(i int32) *int32 { return &i }(60),
					},
				},
			},
			expected: "object_metrics",
			expTarget: &autoscalingv2.MetricTarget{
				Type:               autoscalingv2.MetricTargetType(utilization),
				AverageUtilization: func(i int32) *int32 { return &i }(60),
			},
		},
		{
			metric: autoscalingv2.MetricSpec{
				Type: "Unknown",
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: "resource_metrics",
				},
			},
			expected: "unknown_metric",
		},
	}

	for _, tt := range tests {
		got, gotTarget := extractScopedMetricInfo(&tt.metric)
		if got != tt.expected {
			t.Fatalf("metric name is not match (got=%s, expected=%s)", got, tt.expected)
		}
		if !reflect.DeepEqual(gotTarget, tt.expTarget) {
			t.Fatalf("metric target is not match (got=%s, expected=%s)", gotTarget, tt.expTarget)
		}
	}
}

func TestGenerateMetricUniqueFilter(t *testing.T) {
	const xxxValue2 = "xxx"
	const webValue2 = "web"
	const testValue = "test"
	const sampleValue = "sample"
	const kubeSystemUid = "kube_system_uid"
	const kubeNamespace = "kube_namespace"
	const defaultValue2 = "default"
	tests := []struct {
		kubeSystemUID string
		namespace     string
		kind          string
		name          string
		expected      map[string]string
	}{
		{
			kubeSystemUID: "14646643-26f2-4614-8cf8-a839f3185a12",
			namespace:     defaultValue2,
			kind:          "Deployment",
			name:          testValue,
			expected: map[string]string{
				kubeSystemUid:     "14646643-26f2-4614-8cf8-a839f3185a12",
				kubeNamespace:     defaultValue2,
				"kube_deployment": testValue,
			},
		},
		{
			kubeSystemUID: xxxValue2,
			namespace:     webValue2,
			kind:          "StatefulSet",
			name:          sampleValue,
			expected: map[string]string{
				kubeSystemUid:      xxxValue2,
				kubeNamespace:      webValue2,
				"kube_statefulset": sampleValue,
			},
		},
	}

	for _, tt := range tests {
		got := generateMetricUniqueFilter(tt.kubeSystemUID, tt.namespace, tt.kind, tt.name)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("tag filter is not match (got=%v, expected=%v)", got, tt.expected)
		}
	}
}

func TestTotalResourceList(t *testing.T) {
	tests := []struct {
		containers []corev1.Container
		expected   *corev1.ResourceList
	}{
		{
			containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:     *resource.NewScaledQuantity(100, resource.Milli),
							corev1.ResourceMemory:  *resource.NewScaledQuantity(300, resource.Mega),
							corev1.ResourceStorage: *resource.NewScaledQuantity(5, resource.Giga),
						},
					},
				},
			},
			expected: &corev1.ResourceList{
				corev1.ResourceCPU:     *resource.NewScaledQuantity(100, resource.Milli),
				corev1.ResourceMemory:  *resource.NewScaledQuantity(300, resource.Mega),
				corev1.ResourceStorage: *resource.NewScaledQuantity(5, resource.Giga),
			},
		},
		{
			containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:     *resource.NewScaledQuantity(100, resource.Milli),
							corev1.ResourceMemory:  *resource.NewScaledQuantity(300, resource.Mega),
							corev1.ResourceStorage: *resource.NewScaledQuantity(5, resource.Giga),
						},
					},
				},
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:     *resource.NewScaledQuantity(1, 0),
							corev1.ResourceMemory:  *resource.NewScaledQuantity(400, resource.Mega),
							corev1.ResourceStorage: *resource.NewScaledQuantity(1, resource.Giga),
						},
					},
				},
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:     *resource.NewScaledQuantity(300, resource.Milli),
							corev1.ResourceMemory:  *resource.NewScaledQuantity(2, resource.Giga),
							corev1.ResourceStorage: *resource.NewScaledQuantity(500, resource.Mega),
						},
					},
				},
			},
			expected: &corev1.ResourceList{
				corev1.ResourceCPU:     *resource.NewScaledQuantity(1400, resource.Milli),
				corev1.ResourceMemory:  *resource.NewScaledQuantity(2700, resource.Mega),
				corev1.ResourceStorage: *resource.NewScaledQuantity(6500, resource.Mega),
			},
		},
		{
			containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: *resource.NewScaledQuantity(100, resource.Milli),
						},
					},
				},
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: *resource.NewScaledQuantity(300, resource.Mega),
						},
					},
				},
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: *resource.NewScaledQuantity(5, resource.Giga),
						},
					},
				},
			},
			expected: &corev1.ResourceList{
				corev1.ResourceCPU:     *resource.NewScaledQuantity(100, resource.Milli),
				corev1.ResourceMemory:  *resource.NewScaledQuantity(300, resource.Mega),
				corev1.ResourceStorage: *resource.NewScaledQuantity(5, resource.Giga),
			},
		},
		{
			containers: []corev1.Container{},
			expected:   &corev1.ResourceList{},
		},
	}

	for _, tt := range tests {
		got := totalResourceList(tt.containers)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("total resource list is not match (got=%v, expected=%v)", got, tt.expected)
		}
	}
}

func TestAddOwnerReference(t *testing.T) {
	const xxxValue = "xxx"
	const webValue = "web"
	const ihpaAkeCyberagentCoJpV1beta1 = "ihpa.ake.cyberagent.co.jp/v1beta1"
	const ihpaAkeCyberagentCoJpV1 = "ihpa.ake.cyberagent.co.jp/v1"
	const defaultValue = "default"
	const cm1Value = "cm1"
	tests := []struct {
		owner     *ihpav1beta1.IntelligentHorizontalPodAutoscaler
		dependent metav1.Object
		expected  metav1.Object
	}{
		{
			owner: &ihpav1beta1.IntelligentHorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa1",
					Namespace: defaultValue,
					UID:       xxxValue,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "IntelligentHorizontalPodAutoscaler",
					APIVersion: ihpaAkeCyberagentCoJpV1beta1,
				},
			},
			dependent: metav1.Object(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cm1Value,
					Namespace: defaultValue,
					UID:       xxxValue,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
			}),
			expected: metav1.Object(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cm1Value,
					Namespace: defaultValue,
					UID:       xxxValue,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1beta1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "ihpa1",
							UID:                xxxValue,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
			}),
		},
		{
			owner: &ihpav1beta1.IntelligentHorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa2",
					Namespace: webValue,
					UID:       "mmm",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "IntelligentHorizontalPodAutoscaler",
					APIVersion: ihpaAkeCyberagentCoJpV1,
				},
			},
			dependent: metav1.Object(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hpa",
					Namespace: webValue,
					UID:       "nnn",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
			}),
			expected: metav1.Object(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hpa",
					Namespace: webValue,
					UID:       "nnn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "ihpa2",
							UID:                "mmm",
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
			}),
		},
	}

	for _, tt := range tests {
		addOwnerReference(&(tt.owner.TypeMeta), &(tt.owner.ObjectMeta), tt.dependent)
		if !reflect.DeepEqual(tt.dependent, tt.expected) {
			t.Fatalf("owner reference is not match (dependent=%v, expected=%v)", tt.dependent, tt.expected)
		}
	}
}

func TestRandomMinuteCronFormat(t *testing.T) {
	tests := []struct {
		hour     int
		expected string
	}{
		{
			hour:     0,
			expected: "* 0 * * *",
		},
		{
			hour:     100,
			expected: "* 4 * * *",
		},
		{
			hour:     10,
			expected: "* 10 * * *",
		},
		{
			hour:     -10,
			expected: "* 0 * * *",
		},
	}

	for _, tt := range tests {
		got := randomMinuteCronFormat(tt.hour)

		var err error
		got, err = testReplaceCronMinuteSectionToAny(t, got)
		if err != nil {
			t.Fatal(err)
		}

		if got != tt.expected {
			t.Fatalf("cron format is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}

func testReplaceCronMinuteSectionToAny(t *testing.T, cronFormat string) (string, error) {
	t.Helper()
	// remove random minute section
	if _, err := strconv.Atoi(strings.SplitN(cronFormat, " ", 2)[0]); err != nil {
		return "", fmt.Errorf("cron minute section is not valid (not applied random): %s", cronFormat)
	}
	// overwrite minute section by *
	return "* " + strings.SplitN(cronFormat, " ", 2)[1], nil
}
