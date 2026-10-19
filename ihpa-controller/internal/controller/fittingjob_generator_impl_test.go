package controller

import (
	"encoding/json"
	"reflect"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testFittingJobSample(t *testing.T) (sample1, sample2 *fittingJobGeneratorImpl) {
	const trueValue2 = "true"
	const envvalValue2 = "envval"
	const arg2Value2 = "arg2"
	const zoneValue2 = "zone"
	const yyyValue = "yyy"
	const xxxValue = "xxx"
	const volume1Value2 = "volume1"
	const testValue = "test"
	const secret1Value = "secret1"
	const sample2Value3 = "sample2"
	const sample1Value3 = "sample1"
	const sa1Value2 = "sa1"
	const masterValue2 = "master"
	const ihpaAkeCyberagentCoJpV1beta2Value = "ihpa.ake.cyberagent.co.jp/v1beta2"
	const gpuValue2 = "gpu"
	const env1Value2 = "env1"
	const defaultValue3 = "default"
	const dailyValue = "daily"
	const cyberagentossIntelligentHpaFittingjobV1Value = "cyberagentoss/intelligent-hpa-fittingjob:v1"
	const command1Value2 = "command1"
	const cm1Value2 = "cm1"
	const autoValue = "auto"
	const arg1Value2 = "arg1"
	const appValue2 = "app"
	const noScheduleValue = "NoSchedule"
	const equalValue = "Equal"
	const id9fc642f3Bb9d404dAfd5D04f1d6149adValue2 = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
	t.Helper()
	sample1 = &fittingJobGeneratorImpl{
		fj: &ihpav1beta2.FittingJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sample1Value3,
				Namespace: defaultValue3,
				UID:       id9fc642f3Bb9d404dAfd5D04f1d6149adValue2,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "FittingJob",
				APIVersion: ihpaAkeCyberagentCoJpV1beta2Value,
			},
			Spec: ihpav1beta2.FittingJobSpec{
				Seasonality: autoValue,
				ExecuteOn:   4,
				ChangePointDetectionConfig: ihpav1beta2.ChangePointDetectionConfig{
					PercentageThreshold: 50,
					WindowSize:          100,
					TrajectoryRows:      50,
					TrajectoryFeatures:  5,
					TestRows:            50,
					TestFeatures:        5,
					Lag:                 288,
				},
				CustomConfig: `{"custom_a":1,"custom_b":{"hello":"world"}}`,
				DataConfigMap: corev1.LocalObjectReference{
					Name: "data-configmap",
				},
				JobPatchSpec: ihpav1beta2.JobPatchSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "pull-secret"},
					},
					Image: "fittingjob-image:v1",
				},
				TargetMetric: autoscalingv2.MetricIdentifier{
					Name: "metric.name",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label1": "value1",
							"label2": "value2",
						},
					},
				},
				Provider: ihpav1beta2.MetricProvider{
					Name: "anyprovider",
					ProviderSource: ihpav1beta2.ProviderSource{
						Datadog: &ihpav1beta2.DatadogProviderSource{
							APIKey: xxxValue,
							APPKey: yyyValue,
						},
					},
				},
			},
		},
	}

	sample2 = &fittingJobGeneratorImpl{
		fj: &ihpav1beta2.FittingJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sample2Value3,
				Namespace: testValue,
				UID:       id9fc642f3Bb9d404dAfd5D04f1d6149adValue2,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "FittingJob",
				APIVersion: ihpaAkeCyberagentCoJpV1beta2Value,
			},
			Spec: ihpav1beta2.FittingJobSpec{
				Seasonality: dailyValue,
				ExecuteOn:   5,
				DataConfigMap: corev1.LocalObjectReference{
					Name: "data-configmap2",
				},
				JobPatchSpec: ihpav1beta2.JobPatchSpec{
					// related to Job
					ActiveDeadlineSeconds: func(i int64) *int64 { return &i }(300),
					BackoffLimit:          func(i int32) *int32 { return &i }(5),
					Completions:           func(i int32) *int32 { return &i }(10),
					// related to Pod
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: secret1Value}},
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 50,
									PodAffinityTerm: corev1.PodAffinityTerm{
										TopologyKey: zoneValue2,
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{appValue2: "a"},
										},
									},
								},
							},
						},
					},
					NodeSelector: map[string]string{gpuValue2: trueValue2},
					Tolerations: []corev1.Toleration{
						{
							Key:      masterValue2,
							Operator: equalValue,
							Value:    trueValue2,
							Effect:   noScheduleValue,
						},
					},
					ServiceAccountName: sa1Value2,
					Volumes: []corev1.Volume{
						{
							Name: volume1Value2,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{SecretName: secret1Value},
							},
						},
					},
					// related to Container
					Image:           cyberagentossIntelligentHpaFittingjobV1Value,
					ImagePullPolicy: corev1.PullPolicy("Always"),
					Args:            []string{arg1Value2, arg2Value2},
					Command:         []string{command1Value2},
					Env: []corev1.EnvVar{
						{Name: env1Value2, Value: envvalValue2},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: cm1Value2},
							},
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: *resource.NewScaledQuantity(1000, resource.Milli),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: *resource.NewScaledQuantity(500, resource.Milli),
						},
					},
				},
				TargetMetric: autoscalingv2.MetricIdentifier{
					Name: "metric.name",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"l": "v",
						},
					},
				},
				Provider: ihpav1beta2.MetricProvider{
					Name: "datadog",
					ProviderSource: ihpav1beta2.ProviderSource{
						Datadog: &ihpav1beta2.DatadogProviderSource{
							APIKey: xxxValue,
							APPKey: yyyValue,
						},
					},
				},
			},
		},
	}
	return sample1, sample2
}

func TestFittingJobConfigMapResource(t *testing.T) {
	const sample2Value2 = "sample2"
	const sample1Value2 = "sample1"
	const ihpaAkeCyberagentCoJpV1beta2 = "ihpa.ake.cyberagent.co.jp/v1beta2"
	const defaultValue2 = "default"
	const configJson = "config.json"
	const fittingJobIhpaAkeCyberagentCoJp = "FittingJob.ihpa.ake.cyberagent.co.jp"
	const id9fc642f3Bb9d404dAfd5D04f1d6149adValue = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
	sample1, sample2 := testFittingJobSample(t)
	tests := []struct {
		base               *fittingJobGeneratorImpl
		expectedObjectMeta metav1.ObjectMeta
		expectedData       map[string]string
	}{
		{
			base: sample1,
			expectedObjectMeta: metav1.ObjectMeta{
				Name:      "sample1-config",
				Namespace: defaultValue2,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         ihpaAkeCyberagentCoJpV1beta2,
						Controller:         func(b bool) *bool { return &b }(true),
						BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
						Kind:               fittingJobIhpaAkeCyberagentCoJp,
						Name:               sample1Value2,
						UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue,
					},
				},
			},
			expectedData: map[string]string{
				configJson: `
				{
					"provider":{
						"datadog":{
							"apikey":"xxx",
							"appkey":"yyy"
						}
					},
					"targetMetricsName":"sum:metric.name",
					"targetTags":{
						"label1":"value1",
						"label2":"value2"
					},
					"seasonality":"auto",
					"dataConfigMapName":"data-configmap",
					"dataConfigMapNamespace":"default",
					"changePointDetection":{
						"percentageThreshold":50,
						"windowSize":100,
						"trajectoryRows":50,
						"trajectoryFeatures":5,
						"testRows":50,
						"testFeatures":5,
						"lag":288
					},
					"customConfig":"{\"custom_a\":1,\"custom_b\":{\"hello\":\"world\"}}"
				}`,
			},
		},
		{
			base: sample2,
			expectedObjectMeta: metav1.ObjectMeta{
				Name:      "sample2-config",
				Namespace: "test",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         ihpaAkeCyberagentCoJpV1beta2,
						Controller:         func(b bool) *bool { return &b }(true),
						BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
						Kind:               fittingJobIhpaAkeCyberagentCoJp,
						Name:               sample2Value2,
						UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue,
					},
				},
			},
			expectedData: map[string]string{
				configJson: `
				{
					"provider":{
						"datadog":{
							"apikey":"xxx",
							"appkey":"yyy"
						}
					},
					"targetMetricsName":"sum:metric.name",
					"targetTags":{
						"l":"v"
					},
					"seasonality":"daily",
					"dataConfigMapName":"data-configmap2",
					"dataConfigMapNamespace":"test",
					"changePointDetection":{},
					"customConfig":""
				}`,
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.base.ConfigMapResource()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got.ObjectMeta, tt.expectedObjectMeta) {
			t.Fatalf("configmap objectmeta is not match (got=%v, exp=%v)", got, tt.expectedObjectMeta)
		}

		configName := configJson
		var gotConfig, expConfig any
		if err := json.Unmarshal([]byte(got.Data[configName]), &gotConfig); err != nil {
			t.Fatalf("got data cannot be unmarshal (got=%v): %v", got.Data[configName], err)
		}
		if err := json.Unmarshal([]byte(tt.expectedData[configName]), &expConfig); err != nil {
			t.Fatalf("exp data cannot be unmarshal (exp=%v): %v", tt.expectedData[configName], err)
		}

		if !reflect.DeepEqual(gotConfig, expConfig) {
			t.Fatalf("config is not match (got=%#v, exp=%#v)", gotConfig, expConfig)
		}
	}
}

func TestFittingJobCronJobResource(t *testing.T) {
	const trueValue = "true"
	const envvalValue = "envval"
	const arg2Value = "arg2"
	const zoneValue = "zone"
	const volume1Value = "volume1"
	const sample2Value = "sample2"
	const sample1Value = "sample1"
	const sa1Value = "sa1"
	const masterValue = "master"
	const gpuValue = "gpu"
	const fittingjobConfig = "fittingjob-config"
	const env1Value = "env1"
	const defaultValue = "default"
	const cyberagentossIntelligentHpaFittingjobV1 = "cyberagentoss/intelligent-hpa-fittingjob:v1"
	const command1Value = "command1"
	const cm1Value = "cm1"
	const arg1Value = "arg1"
	const appValue = "app"
	const noSchedule = "NoSchedule"
	const equal = "Equal"
	const id9fc642f3Bb9d404dAfd5D04f1d6149ad = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
	sample1, sample2 := testFittingJobSample(t)
	tests := []struct {
		base     *fittingJobGeneratorImpl
		expected *batchv1beta1.CronJob
	}{
		{
			base: sample1,
			expected: &batchv1beta1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sample1Value,
					Namespace: defaultValue,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta2",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "FittingJob.ihpa.ake.cyberagent.co.jp",
							Name:               sample1Value,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
						},
					},
				},
				Spec: batchv1beta1.CronJobSpec{
					Schedule: "* 4 * * *",
					JobTemplate: batchv1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy: "OnFailure",
									ImagePullSecrets: []corev1.LocalObjectReference{
										{Name: "pull-secret"},
									},
									Containers: []corev1.Container{
										{
											Name:  "fittingjob",
											Image: "fittingjob-image:v1",
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      fittingjobConfig,
													MountPath: "/fittingjob-config",
												},
											},
										},
									},
									Volumes: []corev1.Volume{
										{
											Name: fittingjobConfig,
											VolumeSource: corev1.VolumeSource{
												ConfigMap: &corev1.ConfigMapVolumeSource{
													LocalObjectReference: corev1.LocalObjectReference{Name: "sample1-config"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			base: sample2,
			expected: &batchv1beta1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sample2Value,
					Namespace: "test",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta2",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "FittingJob.ihpa.ake.cyberagent.co.jp",
							Name:               sample2Value,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
						},
					},
				},
				Spec: batchv1beta1.CronJobSpec{
					Schedule: "* 5 * * *",
					JobTemplate: batchv1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							ActiveDeadlineSeconds: func(i int64) *int64 { return &i }(300),
							BackoffLimit:          func(i int32) *int32 { return &i }(5),
							Completions:           func(i int32) *int32 { return &i }(10),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy:      "OnFailure",
									ServiceAccountName: sa1Value,
									ImagePullSecrets: []corev1.LocalObjectReference{
										{Name: "secret1"},
									},
									Affinity: &corev1.Affinity{
										PodAntiAffinity: &corev1.PodAntiAffinity{
											PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
												{
													Weight: 50,
													PodAffinityTerm: corev1.PodAffinityTerm{
														TopologyKey: zoneValue,
														LabelSelector: &metav1.LabelSelector{
															MatchLabels: map[string]string{appValue: "a"},
														},
													},
												},
											},
										},
									},
									NodeSelector: map[string]string{gpuValue: trueValue},
									Tolerations: []corev1.Toleration{
										{
											Key:      masterValue,
											Operator: equal,
											Value:    trueValue,
											Effect:   noSchedule,
										},
									},
									Volumes: []corev1.Volume{
										{
											Name: volume1Value,
											VolumeSource: corev1.VolumeSource{
												Secret: &corev1.SecretVolumeSource{SecretName: "secret1"},
											},
										},
										{
											Name: fittingjobConfig,
											VolumeSource: corev1.VolumeSource{
												ConfigMap: &corev1.ConfigMapVolumeSource{
													LocalObjectReference: corev1.LocalObjectReference{Name: "sample2-config"},
												},
											},
										},
									},
									Containers: []corev1.Container{
										{
											Name:            "fittingjob",
											Image:           cyberagentossIntelligentHpaFittingjobV1,
											ImagePullPolicy: corev1.PullPolicy("Always"),
											Args:            []string{arg1Value, arg2Value},
											Command:         []string{command1Value},
											Env: []corev1.EnvVar{
												{Name: env1Value, Value: envvalValue},
											},
											EnvFrom: []corev1.EnvFromSource{
												{
													ConfigMapRef: &corev1.ConfigMapEnvSource{
														LocalObjectReference: corev1.LocalObjectReference{Name: cm1Value},
													},
												},
											},
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													corev1.ResourceCPU: *resource.NewScaledQuantity(1000, resource.Milli),
												},
												Requests: corev1.ResourceList{
													corev1.ResourceCPU: *resource.NewScaledQuantity(500, resource.Milli),
												},
											},
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      volume1Value,
													MountPath: "/volume1",
												},
												{
													Name:      fittingjobConfig,
													MountPath: "/fittingjob-config",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.base.CronJobResource()
		if err != nil {
			t.Fatal(err)
		}

		got.Spec.Schedule, err = testReplaceCronMinuteSectionToAny(t, got.Spec.Schedule)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("cronjob is not match (got=%#v, exp=%#v)", got, tt.expected)
		}
	}
}
