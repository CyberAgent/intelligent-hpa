package controllers

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	mpconfig "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/config"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	amtypes "k8s.io/apimachinery/pkg/types"
)

type ihpaGeneratorImpl struct {
	// ihpa is a struct of target ihpa
	ihpa *ihpav1beta2.IntelligentHorizontalPodAutoscaler
	// scaleTargetRequests is total of resource map for scaleTargetRef.
	scaleTargetRequests corev1.ResourceList
	// kubeSystemUID is used for guarantee uniqueness of kubernetes cluster.
	kubeSystemUID string
}

func NewIntelligentHorizontalPodAutoscalerGenerator(
	ihpa *ihpav1beta2.IntelligentHorizontalPodAutoscaler,
	r *IntelligentHorizontalPodAutoscalerReconciler,
	ctx context.Context,
) (IntelligentHorizontalPodAutoscalerGenerator, error) {
	kubeSystemNS := &corev1.Namespace{}
	if err := r.Get(ctx, amtypes.NamespacedName{Name: "kube-system"}, kubeSystemNS); err != nil {
		return nil, err
	}

	var containers []corev1.Container
	scaleTarget := ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.ScaleTargetRef
	switch scaleTarget.Kind {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, amtypes.NamespacedName{Namespace: ihpa.GetNamespace(), Name: scaleTarget.Name}, deployment); err != nil {
			return nil, err
		}
		containers = deployment.Spec.Template.Spec.Containers
	case "StatefulSet":
		statefulset := &appsv1.StatefulSet{}
		if err := r.Get(ctx, amtypes.NamespacedName{Namespace: ihpa.GetNamespace(), Name: scaleTarget.Name}, statefulset); err != nil {
			return nil, err
		}
		containers = statefulset.Spec.Template.Spec.Containers
	case "ReplicaSet":
		replicaset := &appsv1.ReplicaSet{}
		if err := r.Get(ctx, amtypes.NamespacedName{Namespace: ihpa.GetNamespace(), Name: scaleTarget.Name}, replicaset); err != nil {
			return nil, err
		}
		containers = replicaset.Spec.Template.Spec.Containers
	}
	rl := *totalResourceList(containers)

	return &ihpaGeneratorImpl{ihpa: ihpa, scaleTargetRequests: rl, kubeSystemUID: string(kubeSystemNS.GetUID())}, nil
}

// HorizontalPodAutoscalerResource returns an HPA resource which forecasted metrics is added to.
func (g *ihpaGeneratorImpl) HorizontalPodAutoscalerResource() (*autoscalingv2beta2.HorizontalPodAutoscaler, error) {
	hpa := autoscalingv2beta2.HorizontalPodAutoscaler{}
	hpa.ObjectMeta = metav1.ObjectMeta{
		Name:      g.hpaName(),
		Namespace: g.ihpa.GetNamespace(),
	}

	extendedMetrics := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics
	metrics := make([]autoscalingv2beta2.MetricSpec, len(extendedMetrics))
	for i := range extendedMetrics {
		metrics[i] = *(extendedMetrics[i].MetricSpec())
	}
	forecastedMetrics := make([]autoscalingv2beta2.MetricSpec, len(metrics))
	for i := range metrics {
		f, err := g.generateForecastedMetricSpec(&metrics[i])
		if err != nil {
			return nil, err
		}
		forecastedMetrics[i] = *f
	}
	metrics = append(metrics, forecastedMetrics...)

	hpa.Spec = autoscalingv2beta2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: *g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.ScaleTargetRef.DeepCopy(),
		MinReplicas:    g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.MinReplicas,
		MaxReplicas:    g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.MaxReplicas,
		Metrics:        metrics,
	}

	addOwnerReference(&(g.ihpa.TypeMeta), &(g.ihpa.ObjectMeta), &hpa)

	return &hpa, nil
}

// generateForecastedMetricSpec returns external MetricSpec for forecasted value.
func (g *ihpaGeneratorImpl) generateForecastedMetricSpec(metric *autoscalingv2beta2.MetricSpec) (*autoscalingv2beta2.MetricSpec, error) {
	metricName, metricTarget := extractScopedMetricInfo(metric)
	if metricTarget == nil {
		return nil, fmt.Errorf("cannot generate correspond metric. please check integrity of the metric instance. (%v)", *metric)
	}

	metricIdentifier, err := g.convertMetricSpecToIdentifier(metric)
	if err != nil {
		return nil, err
	}
	metricIdentifier.Name = correspondForecastedMetricName(metricIdentifier.Name)
	metricIdentifier.Selector = &metav1.LabelSelector{MatchLabels: g.uniqueMetricFilters()}

	var avgValue resource.Quantity
	// Utilization of Resource metrics is special value
	// because this metric refer to metric server and
	// we need whole value of this metric for calculation of utilization.
	// Utilization is used by cpu, memory, storage and ephemeral-storage.
	if metric.Type == "Resource" && metricTarget.Type == "Utilization" {
		var avg int64

		mp := mpconfig.ConvertMetricProvider(g.ihpa.Spec.MetricProvider.DeepCopy()).ActiveProvider()
		mi := mp.ConvertResourceMetricName(metricName, false)
		if mi == nil {
			return nil, fmt.Errorf("correspond metric name is not found (convert failed): %s", metricName)
		}
		// set percentage of total containers requests
		if q, ok := g.scaleTargetRequests[metric.Resource.Name]; ok {
			// Only cpu quantity unit is milli in Kubernetes.
			// Create base quantity
			var qty *resource.Quantity
			var scaleAdjust int
			if metricName == "cpu" {
				qty = resource.NewQuantity(q.MilliValue(), resource.DecimalSI)
				// increase scale unit because cpu value is interpreted by milli scale.
				// metricprovider.MetricIdentifier.GetScale() return scale value based on non scaled value.
				// ex.) 0.05 core -> 50 mcore
				//      datadog value: 50 mcore -> 50,000,000 nanocore
				//      o: 50 (scale: -6) -> 50,000,000 (50 M)
				//      x: 50 (scale: -9) -> 50,000,000,000 (50 G)
				scaleAdjust = 3
			} else {
				// memory, storage is here
				qty = resource.NewQuantity(q.Value(), resource.DecimalSI)
			}
			// multiply utilization percentage to scaled value by specific metric scale
			// NOTE: We have to scale value according to the unit on metric provider.
			//       For example, the unit of "kubernetes.cpu.usage.total" in Datadog is nanocore (-9),
			//       so we must scale value to nanocore. HPA treat external metrics without unit.
			//       Therefore HPA shows very big number, 9,000,000,000 **nanocores** in Datadog is
			//       shown as 9,000,000,000 **cores**, even if Datadog dashboard shows that 9 millicore.
			percentage := float64(*(metricTarget.AverageUtilization)) / 100.0
			requestTotal := float64(qty.ScaledValue(resource.Scale(mi.GetScale() + scaleAdjust)))
			avg = int64(requestTotal * percentage)
			q.Set(avg)
			avgValue = q.DeepCopy()
		} else {
			return nil, fmt.Errorf("cannot set %s as utilization, not found scaleTarget...containers.Resource.Requests", metricName)
		}
		// clear utilization field
		metricTarget.AverageUtilization = nil
	} else if metric.Type == "External" {
		switch metricTarget.Type {
		case "AverageValue":
			avgValue = metricTarget.AverageValue.DeepCopy()
			metricTarget.AverageValue = nil
		case "Value":
			avgValue = metricTarget.Value.DeepCopy()
		}
	}

	// forecasted value is single data point, but it expected to sum value,
	// so we have to store the avg value to AverageValue field.
	// (AverageValue must be devided by number of replicas.)
	metricTarget.Type = "AverageValue"
	metricTarget.AverageValue = &avgValue

	externalSource := &autoscalingv2beta2.ExternalMetricSource{
		Metric: *metricIdentifier,
		Target: *metricTarget,
	}

	forecastedMetric := &autoscalingv2beta2.MetricSpec{
		Type:     "External",
		External: externalSource,
	}

	return forecastedMetric, nil
}

// FittingJobResources generate an array of FittingJob struct
// that generated every metric fields.
func (g *ihpaGeneratorImpl) FittingJobResources() ([]*ihpav1beta2.FittingJob, error) {
	metrics := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics

	errStrs := make([]string, 0, len(metrics))
	fjs := make([]*ihpav1beta2.FittingJob, len(metrics))
	for i := range metrics {
		var err error
		fjs[i], err = g.fittingJobResource(&metrics[i])
		if err != nil {
			errStrs = append(errStrs, err.Error())
		}
	}

	var err error
	if len(errStrs) != 0 {
		err = fmt.Errorf(strings.Join(errStrs, ", "))
	}

	return fjs, err
}

// fittingJobResource generate a FittingJob struct for specified metric.
func (g *ihpaGeneratorImpl) fittingJobResource(metric *ihpav1beta2.ExtendedMetricSpec) (*ihpav1beta2.FittingJob, error) {
	metricIdentifier, err := g.convertMetricSpecToIdentifier(metric.MetricSpec())
	if err != nil {
		return nil, err
	}

	fj := ihpav1beta2.FittingJob{}
	fj.ObjectMeta = metav1.ObjectMeta{
		Name:      g.ihpaMetricString(metric.MetricSpec()),
		Namespace: g.ihpa.GetNamespace(),
	}
	fj.Spec = *metric.FittingJobPatchSpec.GenerateFittingJobSpec()
	fj.Spec.TargetMetric = *metricIdentifier
	fj.Spec.DataConfigMap = corev1.LocalObjectReference{Name: g.configMapName(metric)}
	fj.Spec.Provider = g.ihpa.Spec.MetricProvider

	if fj.Spec.ServiceAccountName == "" {
		fj.Spec.ServiceAccountName = g.rbacName()
	}

	// this id is used for checking existence
	fj.Annotations = map[string]string{
		fittingJobIDAnnotation: g.uniqueMetricID(metric.MetricSpec()),
	}

	addOwnerReference(&(g.ihpa.TypeMeta), &(g.ihpa.ObjectMeta), &fj)

	return &fj, nil
}

// convertMetricSpecToIdentifier convert metric name to special name which is dedicated to metric provider.
// Currently, Resource and External are supported only.
// For example, "cpu" in Resource is convert to "kubernetes.cpu.usage.total" in Datadog.
func (g *ihpaGeneratorImpl) convertMetricSpecToIdentifier(metric *autoscalingv2beta2.MetricSpec) (*autoscalingv2beta2.MetricIdentifier, error) {
	metricIdentifier := &autoscalingv2beta2.MetricIdentifier{}
	switch metric.Type {
	case "Resource":
		mp := mpconfig.ConvertMetricProvider(g.ihpa.Spec.MetricProvider.DeepCopy()).ActiveProvider()
		mi := mp.ConvertResourceMetricName(metric.Resource.Name.String(), false)
		filters := g.uniqueMetricFilters()

		metricIdentifier.Name = mi.GetName()
		metricIdentifier.Selector = &metav1.LabelSelector{MatchLabels: filters}
	case "External":
		metricIdentifier = metric.External.Metric.DeepCopy()
	default:
		return nil, fmt.Errorf("%s metric is not supported yet.", metric.Type)
	}

	return metricIdentifier, nil
}

// EstimatorResources generate an array of Estimator struct
func (g *ihpaGeneratorImpl) EstimatorResources() ([]*ihpav1beta2.Estimator, error) {
	metrics := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics

	errStrs := make([]string, 0, len(metrics))
	ests := make([]*ihpav1beta2.Estimator, len(metrics))
	for i := range metrics {
		var err error
		ests[i], err = g.estimatorResource(&metrics[i])
		if err != nil {
			errStrs = append(errStrs, err.Error())
		}
	}

	var err error
	if len(errStrs) != 0 {
		err = fmt.Errorf(strings.Join(errStrs, ", "))
	}

	return ests, err
}

// estimatorResource generate a FittingJob struct for specified metric.
func (g *ihpaGeneratorImpl) estimatorResource(metric *ihpav1beta2.ExtendedMetricSpec) (*ihpav1beta2.Estimator, error) {
	metricIdentifier, err := g.convertMetricSpecToIdentifier(metric.MetricSpec())
	if err != nil {
		return nil, err
	}

	tagMap := g.uniqueMetricFilters()
	tags := make([]string, 0, len(tagMap))
	for k, v := range tagMap {
		tags = append(tags, k+":"+v)
	}
	sort.Strings(tags)

	// search base tags
	var baseMetricTags []string
	for _, metric := range g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics {
		if metric.Type != "External" {
			continue
		}
		name, _ := extractScopedMetricInfo(metric.MetricSpec())
		if name == metricIdentifier.Name {
			m := metric.External.Metric.Selector.MatchLabels
			baseMetricTags = make([]string, 0, len(m))
			for k, v := range m {
				baseMetricTags = append(baseMetricTags, k+":"+v)
			}
			sort.Strings(baseMetricTags)
			break
		}
	}
	if baseMetricTags == nil {
		// tags of excepting External is generated by unique filter
		baseMetricTags = tags
	}

	meta := metav1.ObjectMeta{
		Name:      g.ihpaMetricString(metric.MetricSpec()),
		Namespace: g.ihpa.GetNamespace(),
		Annotations: map[string]string{
			fittingJobIDAnnotation: g.uniqueMetricID(metric.MetricSpec()),
		},
	}

	spec := *g.ihpa.Spec.EstimatorPatchSpec.GenerateEstimatorSpec()
	spec.DataConfigMap = corev1.LocalObjectReference{Name: g.configMapName(metric)}
	spec.Provider = g.ihpa.Spec.MetricProvider
	spec.MetricName = correspondForecastedMetricName(metricIdentifier.Name)
	spec.MetricTags = tags
	spec.BaseMetricName = metricIdentifier.Name
	spec.BaseMetricTags = baseMetricTags

	est := ihpav1beta2.Estimator{
		ObjectMeta: meta,
		Spec:       spec,
	}

	addOwnerReference(&(g.ihpa.TypeMeta), &(g.ihpa.ObjectMeta), &est)

	return &est, nil
}

// RBACResources generate some resource for accessing to ConfigMap from FittingJob.
func (g *ihpaGeneratorImpl) RBACResources() (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding, error) {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.rbacName(),
			Namespace: g.ihpa.GetNamespace(),
		},
	}
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.rbacName(),
			Namespace: g.ihpa.GetNamespace(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: g.allConfigMapName(),
				Verbs:         []string{"get", "update", "patch"},
			},
		},
	}
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.rbacName(),
			Namespace: g.ihpa.GetNamespace(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     g.rbacName(),
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      g.rbacName(),
				Namespace: g.ihpa.GetNamespace(),
			},
		},
	}

	addOwnerReference(&(g.ihpa.TypeMeta), &(g.ihpa.ObjectMeta), &sa)
	addOwnerReference(&(g.ihpa.TypeMeta), &(g.ihpa.ObjectMeta), &role)
	addOwnerReference(&(g.ihpa.TypeMeta), &(g.ihpa.ObjectMeta), &roleBinding)

	return &sa, &role, &roleBinding, nil
}

// uniqueMetricID return identifier for a metric. This ID is used for
// identification each fittingjob.
func (g *ihpaGeneratorImpl) uniqueMetricID(metric *autoscalingv2beta2.MetricSpec) string {
	hash := g.uniqueMetricHash(metric)
	dst := make([]byte, hex.EncodedLen(len(hash)))
	hex.Encode(dst, hash)
	return string(dst)
}

// uniqueMetricHash return identifier for a metric as md5 hash (fixed-length(16) byte).
func (g *ihpaGeneratorImpl) uniqueMetricHash(metric *autoscalingv2beta2.MetricSpec) []byte {
	metricName, _ := extractScopedMetricInfo(metric)
	targetKind := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.ScaleTargetRef.Kind
	targetName := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.ScaleTargetRef.Name

	h := md5.New()
	io.WriteString(h, metricName)
	io.WriteString(h, correspondForecastedMetricName(metricName))
	io.WriteString(h, g.ihpa.GetNamespace())
	io.WriteString(h, targetKind)
	io.WriteString(h, targetName)
	return h.Sum(nil)
}

// uniqueMetricTags return tags as an array of string for identification
// forecasted metric.
func (g *ihpaGeneratorImpl) uniqueMetricTags() []string {
	filters := g.uniqueMetricFilters()

	tags := make([]string, 0, len(filters))
	for k, v := range filters {
		tags = append(tags, k+":"+v)
	}

	return tags
}

// uniqueMetricFilters return filters as an map of string.
func (g *ihpaGeneratorImpl) uniqueMetricFilters() map[string]string {
	targetKind := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.ScaleTargetRef.Kind
	targetName := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.ScaleTargetRef.Name

	return generateMetricUniqueFilter(
		g.kubeSystemUID,
		g.ihpa.GetNamespace(),
		targetKind,
		targetName,
	)
}

// ihpaMetricString return namespaced metric name.
// e.g.) ihpa-<ihpa_name>-<metric_name>
func (g *ihpaGeneratorImpl) ihpaMetricString(metric *autoscalingv2beta2.MetricSpec) string {
	// uniqueness of name is guaranteed by Kind, Name, MetricName
	// because CronJob is namespaced resource.
	metricName, _ := extractScopedMetricInfo(metric)
	s := fmt.Sprintf("%s-%s", g.ihpaString(), strings.ToLower(metricName))
	return sanitizeForKubernetesResourceName(s)
}

// ihpaMetricString return namespaced resource name.
// This name conflicts in same resource.
// e.g.) ihpa-<ihpa_name>
func (g *ihpaGeneratorImpl) ihpaString() string {
	s := fmt.Sprintf("ihpa-%s", strings.ToLower(g.ihpa.GetName()))
	return sanitizeForKubernetesResourceName(s)
}

func (g *ihpaGeneratorImpl) hpaName() string  { return g.ihpaString() }
func (g *ihpaGeneratorImpl) rbacName() string { return g.ihpaString() }
func (g *ihpaGeneratorImpl) configMapName(metric *ihpav1beta2.ExtendedMetricSpec) string {
	return g.ihpaMetricString(metric.MetricSpec())
}
func (g *ihpaGeneratorImpl) allConfigMapName() []string {
	metrics := g.ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics
	cmNames := make([]string, len(metrics))
	for i := range metrics {
		cmNames[i] = g.configMapName(&metrics[i])
	}
	return cmNames
}
