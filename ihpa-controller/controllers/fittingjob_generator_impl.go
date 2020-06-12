package controllers

import (
	"encoding/json"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	mpconfig "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/config"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fittingJobGeneratorImpl struct {
	fj *ihpav1beta2.FittingJob
}

func NewFittingJobGenerator(fj *ihpav1beta2.FittingJob) (FittingJobGenerator, error) {
	return &fittingJobGeneratorImpl{fj: fj}, nil
}

// ConfigMapResource generates an instance of ConfigMap (v1)
func (g *fittingJobGeneratorImpl) ConfigMapResource() (*corev1.ConfigMap, error) {
	mpConfig := mpconfig.ConvertMetricProvider(&g.fj.Spec.Provider)

	fittingJobConfig := &FittingJobConfig{
		MetricProvider:             *mpConfig,
		TargetMetricsName:          mpConfig.ActiveProvider().AddSumAggregator(g.fj.Spec.TargetMetric.Name),
		TargetTags:                 g.fj.Spec.TargetMetric.Selector.MatchLabels,
		Seasonality:                g.fj.Spec.Seasonality,
		ChangePointDetectionConfig: g.fj.Spec.ChangePointDetectionConfig,
		CustomConfig:               g.fj.Spec.CustomConfig,
		DataConfigMapName:          g.fj.Spec.DataConfigMap.Name,
		DataConfigMapNamespace:     g.fj.GetNamespace(),
	}
	configData, err := json.Marshal(fittingJobConfig)
	if err != nil {
		return nil, err
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.configMapName(),
			Namespace: g.fj.GetNamespace(),
		},
		Data: map[string]string{
			"config.json": string(configData),
		},
	}

	addOwnerReference(&(g.fj.TypeMeta), &(g.fj.ObjectMeta), &configMap)

	return &configMap, nil
}

// CronJobResource generates an instance of CronJob (v1beta1)
func (g *fittingJobGeneratorImpl) CronJobResource() (*batchv1beta1.CronJob, error) {
	jobSpec := *g.fj.Spec.JobPatchSpec.GenerateJobSpec()

	volume := corev1.Volume{
		Name: "fittingjob-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: g.configMapName()},
			},
		},
	}
	volumeMount := corev1.VolumeMount{Name: volume.Name, MountPath: "/" + volume.Name}
	jobSpec.Template.Spec.Volumes = append(jobSpec.Template.Spec.Volumes, volume)

	jobSpec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure

	fittingContainer := jobSpec.Template.Spec.Containers[0]
	fittingContainer.Name = "fittingjob"
	if fittingContainer.Image == "" {
		fittingContainer.Image = DefaultImage
	}
	fittingContainer.VolumeMounts = append(fittingContainer.VolumeMounts, volumeMount)
	jobSpec.Template.Spec.Containers[0] = fittingContainer

	cj := batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.fj.GetName(),
			Namespace: g.fj.GetNamespace(),
		},
		Spec: batchv1beta1.CronJobSpec{
			JobTemplate: batchv1beta1.JobTemplateSpec{Spec: jobSpec},
			Schedule:    randomMinuteCronFormat(int(g.fj.Spec.ExecuteOn)),
		},
	}

	addOwnerReference(&(g.fj.TypeMeta), &(g.fj.ObjectMeta), &cj)

	return &cj, nil
}

func (g *fittingJobGeneratorImpl) configMapName() string {
	return g.fj.GetName() + "-config"
}
