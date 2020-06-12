package controllers

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KubernetesProvider = "ake"
	MetricPath         = KubernetesProvider + ".ihpa"
)

// sanitizeForKubernetesResourceName sanitizes string for kubernetes resource name
func sanitizeForKubernetesResourceName(s string) string {
	sanitizeMap := map[string]string{".": "-", "_": "-"}
	return replaceStrings(s, sanitizeMap)
}

// correspondForecastedMetricName sanitizes metricName for storing metric provider.
func correspondForecastedMetricName(metricName string) string {
	sanitizeMap := map[string]string{".": "_", "-": "_"}
	return MetricPath + ".forecasted_" + replaceStrings(metricName, sanitizeMap)
}

// replaceString replaces m.key to m.value in string s.
func replaceStrings(s string, m map[string]string) string {
	for before, after := range m {
		s = strings.Replace(s, before, after, -1)
	}
	return s
}

// extractScopedMetricInfo extracts name and target information of the metric.
func extractScopedMetricInfo(metric *autoscalingv2beta2.MetricSpec) (string, *autoscalingv2beta2.MetricTarget) {
	var name string
	var target *autoscalingv2beta2.MetricTarget
	switch metric.Type {
	case "Resource":
		name = metric.Resource.Name.String()
		target = metric.Resource.Target.DeepCopy()
	case "Object":
		name = metric.Object.Metric.Name
		target = metric.Object.Target.DeepCopy()
	case "Pods":
		name = metric.Pods.Metric.Name
		target = metric.Pods.Target.DeepCopy()
	case "External":
		name = metric.External.Metric.Name
		target = metric.External.Target.DeepCopy()
	default:
		name = "unknown_metric"
		target = nil
	}
	return name, target
}

// generateMetricUniqueFilter return unique resource map.
func generateMetricUniqueFilter(kubeSystemUID, namespace, kind, name string) map[string]string {
	return map[string]string{
		"kube_system_uid":               kubeSystemUID,
		"kube_namespace":                namespace,
		"kube_" + strings.ToLower(kind): name,
	}
}

// totalResourceList sum up all requests of all containers.
func totalResourceList(containers []corev1.Container) *corev1.ResourceList {
	resourceLists := make([]corev1.ResourceList, 0, len(containers))
	for _, c := range containers {
		if rl := c.Resources.Requests; rl != nil {
			resourceLists = append(resourceLists, rl)
		}
	}

	return sumUpResourceLists(resourceLists)
}

// sumUpResourceLists sum up all resourceLists.
func sumUpResourceLists(resourceLists []corev1.ResourceList) *corev1.ResourceList {
	baserl := corev1.ResourceList{}

	for _, rl := range resourceLists {
		for k, v := range rl {
			if baseValue, ok := baserl[k]; ok {
				baseValue.Add(v)
				baserl[k] = baseValue
			} else {
				baserl[k] = v
			}
		}
	}

	return &baserl
}

// addOwnerReference add owner reference to dependent.
func addOwnerReference(ownerTypeMeta *metav1.TypeMeta, ownerObjectMeta *metav1.ObjectMeta, dependent metav1.Object) {
	if ownerTypeMeta == nil || ownerObjectMeta == nil || dependent == nil {
		return
	}
	ownerReferences := []metav1.OwnerReference{
		{
			APIVersion:         ownerTypeMeta.GetObjectKind().GroupVersionKind().GroupVersion().String(),
			Controller:         func(b bool) *bool { return &b }(true),
			BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
			Kind:               ownerTypeMeta.GetObjectKind().GroupVersionKind().GroupKind().String(),
			Name:               ownerObjectMeta.GetName(),
			UID:                ownerObjectMeta.GetUID(),
		},
	}
	dependent.SetOwnerReferences(ownerReferences)
}

// randomMinuteCronFormat return cron format that is set specified hour
// (it means daily execution) and randomized minute.
func randomMinuteCronFormat(hour int) string {
	if hour < 0 {
		hour = 0
	}
	hour %= 24
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d %d * * *", rand.Intn(60), hour)
}
