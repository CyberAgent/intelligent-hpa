package controller

import (
	"context"
	"fmt"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// datadogCredentialEnv selects references without copying any credential values.
// As in the runtime resolver, the last source containing a key wins independently
// for APIKey and APPKey. Missing keys retain an earlier source or inline fallback.
func datadogCredentialEnv(ctx context.Context, c client.Client, namespace string, dd *ihpav1beta2.DatadogProviderSource) ([]corev1.EnvVar, error) {
	if dd == nil {
		return nil, nil
	}
	selected := make(map[string]corev1.EnvVar, 2)
	keys := []string{"APIKey", "APPKey"}
	for _, source := range dd.KeysFrom {
		if ref := source.SecretRef; ref != nil {
			secret := &corev1.Secret{}
			if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, secret); err != nil {
				if ref.Optional == nil || !*ref.Optional || !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("failed to select Datadog Secret keys: %w", err)
				}
			} else {
				for _, key := range keys {
					if _, exists := secret.Data[key]; exists {
						selected[key] = corev1.EnvVar{Name: key, ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: ref.LocalObjectReference, Key: key, Optional: ref.Optional},
						}}
					}
				}
			}
		}
		if ref := source.ConfigMapRef; ref != nil {
			cm := &corev1.ConfigMap{}
			if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, cm); err != nil {
				if ref.Optional == nil || !*ref.Optional || !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("failed to select Datadog ConfigMap keys: %w", err)
				}
			} else {
				for _, key := range keys {
					if _, exists := cm.Data[key]; exists {
						selected[key] = corev1.EnvVar{Name: key, ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: ref.LocalObjectReference, Key: key, Optional: ref.Optional},
						}}
					}
				}
			}
		}
	}
	env := make([]corev1.EnvVar, 0, len(selected))
	for _, key := range keys {
		if ref, exists := selected[key]; exists {
			env = append(env, ref)
		}
	}
	return env, nil
}
