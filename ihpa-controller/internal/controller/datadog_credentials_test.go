package controller

import (
	"context"
	"reflect"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDatadogCredentialKeySelection(t *testing.T) {
	optional := true
	required := false
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "default"}, Data: map[string][]byte{
			"APIKey": []byte("secret-api"), "APPKey": []byte("secret-app"), "OtherPassword": []byte("unrelated-password"),
		}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "override", Namespace: "default"}, Data: map[string]string{"APIKey": "cm-api", "OtherSetting": "unrelated"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: "default"}},
	).Build()
	secretSource := corev1.EnvFromSource{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "shared"}}}
	missingOptional := corev1.EnvFromSource{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing"}, Optional: &optional}}
	for _, tt := range []struct {
		name      string
		namespace string
		sources   []corev1.EnvFromSource
		want      []corev1.EnvVar
		wantError bool
	}{
		{
			name: "per-key precedence excludes unrelated entries", namespace: "default",
			sources: []corev1.EnvFromSource{secretSource, {Prefix: "ignored-prefix", ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "override"}, Optional: &required}}, missingOptional},
			want: []corev1.EnvVar{
				{Name: "APIKey", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "override"}, Key: "APIKey", Optional: &required}}},
				{Name: "APPKey", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "shared"}, Key: "APPKey"}}},
			},
		},
		{name: "optional missing source preserves inline fallback", namespace: "default", sources: []corev1.EnvFromSource{missingOptional}, want: []corev1.EnvVar{}},
		{name: "missing keys preserve inline fallback", namespace: "default", sources: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "empty"}}}}, want: []corev1.EnvVar{}},
		{name: "required missing source fails", namespace: "default", sources: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing"}}}}, wantError: true},
		{name: "cannot select another namespace", namespace: "other", sources: []corev1.EnvFromSource{secretSource}, wantError: true},
		{
			name: "optional selected references stay optional", namespace: "default",
			sources: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "override"}, Optional: &optional}}},
			want:    []corev1.EnvVar{{Name: "APIKey", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "override"}, Key: "APIKey", Optional: &optional}}}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dd := &ihpav1beta2.DatadogProviderSource{APIKey: "inline-api", APPKey: "inline-app", KeysFrom: tt.sources}
			got, err := datadogCredentialEnv(context.Background(), c, tt.namespace, dd)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tt.wantError)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("references = %#v, want %#v", got, tt.want)
			}
			if dd.APIKey != "inline-api" || dd.APPKey != "inline-app" {
				t.Fatal("credential selection mutated the provider")
			}
		})
	}
}
