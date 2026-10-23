package controller

import (
	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type IntelligentHorizontalPodAutoscalerGenerator interface {
	// HorizontalPodAutoscalerResource generate a HPA (autoscaling/v2) struct
	// that added some forecast metric fields.
	HorizontalPodAutoscalerResource() (*autoscalingv2.HorizontalPodAutoscaler, error)
	// ScaledObjectResource generate a KEDA ScaledObject (keda.sh/v1alpha1) struct
	// whose triggers read the forecasted metrics.
	ScaledObjectResource() (*kedav1alpha1.ScaledObject, error)
	// FittingJobResources generate an array of FittingJob struct
	// that generated every metric fields.
	FittingJobResources() ([]*ihpav1beta2.FittingJob, error)
	// EstimatorResources generate an array of Estimator struct
	EstimatorResources() ([]*ihpav1beta2.Estimator, error)
	// RBACResources generate some resource for accessing to ConfigMap from FittingJob.
	RBACResources() (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding, error)
	// StatusConfigMapResource generate a ConfigMap that contains status information
	// about the reconciled IHPA resources (HPA, FittingJobs, Estimators).
	StatusConfigMapResource() (*corev1.ConfigMap, error)
}
