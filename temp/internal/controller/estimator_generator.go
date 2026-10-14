package controllers

import (
	corev1 "k8s.io/api/core/v1"
)

type EstimatorGenerator interface {
	// ConfigMapResource generate configmap for predicted data.
	ConfigMapResource() (*corev1.ConfigMap, error)
}
