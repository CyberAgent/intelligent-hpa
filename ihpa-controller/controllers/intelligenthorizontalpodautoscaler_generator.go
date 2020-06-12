package controllers

import (
	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type IntelligentHorizontalPodAutoscalerGenerator interface {
	// HorizontalPodAutoscalerResource generate a HPA (v2beta2.autoscaling) struct
	// that added some forecast metric fields.
	HorizontalPodAutoscalerResource() (*autoscalingv2beta2.HorizontalPodAutoscaler, error)
	// FittingJobResources generate an array of FittingJob struct
	// that generated every metric fields.
	FittingJobResources() ([]*ihpav1beta2.FittingJob, error)
	// EstimatorResources generate an array of Estimator struct
	EstimatorResources() ([]*ihpav1beta2.Estimator, error)
	// RBACResources generate some resource for accessing to ConfigMap from FittingJob.
	RBACResources() (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding, error)
}
