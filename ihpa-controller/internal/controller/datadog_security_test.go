package controller

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDatadogReferencesStayReferences(t *testing.T) {
	sample, _ := testIHPAGeneratorSample(t)
	sample.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.ScaleTargetRef.Kind = ""
	sample.ihpa.Spec.MetricProvider.Datadog = &ihpav1beta2.DatadogProviderSource{
		KeysFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "keys"}}}},
	}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keys", Namespace: "default"}, Data: map[string][]byte{"APIKey": []byte("secret-api"), "APPKey": []byte("secret-app"), "OtherPassword": []byte("unrelated-password")}},
	).Build()
	generator, err := NewIntelligentHorizontalPodAutoscalerGenerator(sample.ihpa, &IntelligentHorizontalPodAutoscalerReconciler{Client: c}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fjs, err := generator.FittingJobResources()
	if err != nil {
		t.Fatal(err)
	}
	ests, err := generator.EstimatorResources()
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range []any{sample.ihpa, fjs, ests} {
		data, _ := json.Marshal(object)
		if strings.Contains(string(data), "secret-api") || strings.Contains(string(data), "secret-app") {
			t.Fatal("resolved credential persisted")
		}
	}
	for _, fj := range fjs {
		fj.Spec.Env = []corev1.EnvVar{{Name: "APIKey", Value: "overridden"}}
		g, err := NewFittingJobGenerator(context.Background(), c, fj)
		if err != nil {
			t.Fatal(err)
		}
		cm, err := g.ConfigMapResource()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(cm.Data["config.json"], "secret-api") {
			t.Fatal("secret in ConfigMap")
		}
		cj, err := g.CronJobResource()
		if err != nil {
			t.Fatal(err)
		}
		if fj.Spec.Env[0].Value != "overridden" {
			t.Fatal("job generation modified the source CR environment")
		}
		container := cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
		if len(container.EnvFrom) != 0 {
			t.Fatal("credential source exposed through envFrom")
		}
		if len(container.Env) != 2 {
			t.Fatal("job must receive exactly two credential key references")
		}
		for _, env := range container.Env {
			if env.Value != "" || env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil ||
				env.ValueFrom.SecretKeyRef.Name != "keys" || env.ValueFrom.SecretKeyRef.Key != env.Name {
				t.Fatal("job lost explicit Secret key reference")
			}
		}
	}
	private := ests[0].Spec.Provider.DeepCopy()
	if err := resolveDatadogKeysFromEnv(context.Background(), c, "default", private.Datadog); err != nil {
		t.Fatal(err)
	}
	if private.Datadog.APIKey != "secret-api" || ests[0].Spec.Provider.Datadog.APIKey != "" {
		t.Fatal("runtime credential isolation failed")
	}
}
