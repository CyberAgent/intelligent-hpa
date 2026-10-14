package controllers

import (
	"reflect"
	"testing"

	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = ihpav1beta2.AddToScheme(scheme)
	return scheme
}

func testEstimatorGeneratorSample(t *testing.T) *estimatorGeneratorImpl {
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
		scheme: newTestScheme(),
	}
	return sample1
}

func TestEstimatorConfigMapResource(t *testing.T) {
	sample := testEstimatorGeneratorSample(t)

	got, err := sample.ConfigMapResource()
	if err != nil {
		t.Fatalf("ConfigMapResource() returned error: %v", err)
	}

	if got.Name != "ihpa-sample1-cpu" {
		t.Errorf("configmap name mismatch: got=%s, exp=ihpa-sample1-cpu", got.Name)
	}
	if got.Namespace != "default" {
		t.Errorf("configmap namespace mismatch: got=%s, exp=default", got.Namespace)
	}

	// Verify owner reference is set
	if len(got.OwnerReferences) != 1 {
		t.Fatalf("expected 1 owner reference, got %d", len(got.OwnerReferences))
	}
	ownerRef := got.OwnerReferences[0]
	if ownerRef.Name != "ihpa-sample1-cpu" {
		t.Errorf("owner reference name mismatch: got=%s, exp=ihpa-sample1-cpu", ownerRef.Name)
	}
	if ownerRef.Kind != "Estimator" {
		t.Errorf("owner reference kind mismatch: got=%s, exp=Estimator", ownerRef.Kind)
	}
	if ownerRef.Controller == nil || !*ownerRef.Controller {
		t.Errorf("expected owner reference controller to be true")
	}
}

func TestEstimatorGeneratorWithDifferentNamespace(t *testing.T) {
	est := &v1beta2.Estimator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ihpa-test-memory",
			Namespace: "production",
		},
		Spec: v1beta2.EstimatorSpec{
			Mode:           "none",
			GapMinutes:     5,
			MetricName:     "memory-metric",
			MetricTags:     []string{"env:prod"},
			BaseMetricName: "base-memory",
			BaseMetricTags: []string{"env:prod"},
			Provider: ihpav1beta2.MetricProvider{
				Name: "prometheus",
				ProviderSource: ihpav1beta2.ProviderSource{
					Prometheus: &ihpav1beta2.PrometheusProviderSource{
						URL:            "http://prometheus:9090",
						PushgatewayURL: "http://pushgateway:9091",
					},
				},
			},
			DataConfigMap: corev1.LocalObjectReference{
				Name: "ihpa-test-memory",
			},
		},
	}
	g := &estimatorGeneratorImpl{est: est, scheme: newTestScheme()}

	got, err := g.ConfigMapResource()
	if err != nil {
		t.Fatalf("ConfigMapResource() returned error: %v", err)
	}

	expected := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ihpa-test-memory",
			Namespace: "production",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "",
					Kind:               "Estimator",
					Name:               "ihpa-test-memory",
					Controller:         func(b bool) *bool { return &b }(true),
					BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
				},
			},
		},
	}

	if got.Name != expected.Name {
		t.Errorf("name mismatch: got=%s, exp=%s", got.Name, expected.Name)
	}
	if got.Namespace != expected.Namespace {
		t.Errorf("namespace mismatch: got=%s, exp=%s", got.Namespace, expected.Namespace)
	}
	if len(got.OwnerReferences) != 1 {
		t.Fatalf("expected 1 owner reference, got %d", len(got.OwnerReferences))
	}
	if !reflect.DeepEqual(got.OwnerReferences[0].Name, expected.OwnerReferences[0].Name) {
		t.Errorf("owner reference name mismatch: got=%s, exp=%s", got.OwnerReferences[0].Name, expected.OwnerReferences[0].Name)
	}
}
