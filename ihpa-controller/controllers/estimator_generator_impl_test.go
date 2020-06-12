package controllers

import (
	"testing"

	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func testEstimatorGeneratorSample(t *testing.T) *estimatorGeneratorImpl {
	// TODO: test by suite_test
	t.Helper()
	sample1 := &estimatorGeneratorImpl{
		est: &v1beta2.Estimator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ihpa-sample1-cpu",
				Namespace: "default",
			},
			Spec: v1beta2.EstimatorSpec{
				Mode:           "adjust",
				GapMinutes:     10,
				MetricName:     "test",
				MetricTags:     []string{"a", "b"},
				BaseMetricName: "base",
				BaseMetricTags: []string{"c", "d"},
				Provider: ihpav1beta2.MetricProvider{
					Name: "sample-provider",
					ProviderSource: ihpav1beta2.ProviderSource{
						Datadog: &ihpav1beta2.DatadogProviderSource{
							APIKey: "xxx",
							APPKey: "yyy",
						},
					},
				},
				DataConfigMap: corev1.LocalObjectReference{
					Name: "ihpa-sample1-cpu",
				},
			},
		},
		scheme: runtime.NewScheme(),
	}
	return sample1
}
