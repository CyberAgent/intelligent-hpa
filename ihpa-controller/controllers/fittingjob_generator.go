package controllers

import (
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

type FittingJobGenerator interface {
	// ConfigMapResource generate configmap for configuration of fittingjob image.
	ConfigMapResource() (*corev1.ConfigMap, error)
	// CronJobResource generate cronjob for running fittingjob image.
	CronJobResource() (*batchv1beta1.CronJob, error)
}
