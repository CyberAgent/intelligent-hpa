package controller

import (
	"context"
	"reflect"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func newReconcilerForTest(t *testing.T, objs ...client.Object) *IntelligentHorizontalPodAutoscalerReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := ihpav1beta2.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	return &IntelligentHorizontalPodAutoscalerReconciler{
		Client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		Log:           zap.New(zap.UseDevMode(true)),
		Scheme:        scheme,
		fittingJobMap: make(map[string]map[string]struct{}),
	}
}

func testReconcileObjects() (*corev1.Namespace, *appsv1.Deployment) {
	kubeSystem := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: "46f9e396-d3c4-4103-a807-49054f47bbfb"},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "nginx",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
					}},
				},
			},
		},
	}
	return kubeSystem, deployment
}

func testReconcileIHPA(metricNames ...corev1.ResourceName) *ihpav1beta2.IntelligentHorizontalPodAutoscaler {
	metrics := make([]ihpav1beta2.ExtendedMetricSpec, 0, len(metricNames))
	for _, name := range metricNames {
		metrics = append(metrics, ihpav1beta2.ExtendedMetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: name,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: new(int32(50)),
				},
			},
			FittingJobPatchSpec: ihpav1beta2.FittingJobPatchSpec{Seasonality: "weekly", ExecuteOn: 4},
		})
	}

	return &ihpav1beta2.IntelligentHorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default", UID: "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"},
		TypeMeta: metav1.TypeMeta{
			Kind:       "IntelligentHorizontalPodAutoscaler",
			APIVersion: "ihpa.ake.cyberagent.co.jp/v1beta2",
		},
		Spec: ihpav1beta2.IntelligentHorizontalPodAutoscalerSpec{
			HorizontalPodAutoscalerTemplate: ihpav1beta2.ExtendedHorizontalPodAutoscalerTemplateSpec{
				Spec: ihpav1beta2.ExtendedHorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "nginx"},
					MinReplicas:    new(int32(1)),
					MaxReplicas:    5,
					Metrics:        metrics,
				},
			},
			EstimatorPatchSpec: ihpav1beta2.EstimatorPatchSpec{Mode: "adjust", GapMinutes: 5},
			MetricProvider: ihpav1beta2.MetricProvider{
				Name:           "sample-provider",
				ProviderSource: ihpav1beta2.ProviderSource{Datadog: &ihpav1beta2.DatadogProviderSource{APIKey: "xxx", APPKey: "yyy"}},
			},
		},
	}
}

func reconcileForTest(t *testing.T, r *IntelligentHorizontalPodAutoscalerReconciler) {
	t.Helper()

	if _, err := reconcileResultForTest(r); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
}

func getHPAForTest(t *testing.T, r *IntelligentHorizontalPodAutoscalerReconciler) *autoscalingv2.HorizontalPodAutoscaler {
	t.Helper()

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "ihpa-nginx"}, hpa); err != nil {
		t.Fatalf("failed to get hpa: %v", err)
	}
	return hpa
}

func testReconcileKEDAIHPA() *ihpav1beta2.IntelligentHorizontalPodAutoscaler {
	ihpa := testReconcileIHPA(corev1.ResourceCPU)
	ihpa.Spec.MetricProvider = ihpav1beta2.MetricProvider{
		Name:           "prometheus",
		ProviderSource: ihpav1beta2.ProviderSource{Prometheus: &ihpav1beta2.PrometheusProviderSource{URL: "http://prometheus:9090", PushgatewayURL: "http://pushgateway:9091"}},
	}
	ihpa.Spec.ScaleBackend = ihpav1beta2.ScaleBackend{Type: ihpav1beta2.ScaleBackendKEDA}
	return ihpa
}

func getScaledObjectForTest(r *IntelligentHorizontalPodAutoscalerReconciler) (*unstructured.Unstructured, error) {
	so := &unstructured.Unstructured{}
	so.SetGroupVersionKind(scaledObjectGVK)
	err := r.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "ihpa-nginx"}, so)
	return so, err
}

func reconcileResultForTest(r *IntelligentHorizontalPodAutoscalerReconciler) (ctrl.Result, error) {
	return r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "nginx"}})
}

func scaledObjectMaxReplicasForTest(t *testing.T, r *IntelligentHorizontalPodAutoscalerReconciler) int64 {
	t.Helper()

	so, err := getScaledObjectForTest(r)
	if err != nil {
		t.Fatalf("failed to get scaledobject: %v", err)
	}
	got, _, err := unstructured.NestedInt64(so.Object, "spec", "maxReplicaCount")
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestReconcile_HPAMode(t *testing.T) {
	t.Parallel()

	t.Run("success: creates all resources", func(t *testing.T) {
		t.Parallel()

		kubeSystem, deployment := testReconcileObjects()
		r := newReconcilerForTest(t, kubeSystem, deployment, testReconcileIHPA(corev1.ResourceCPU))

		reconcileForTest(t, r)

		ctx := context.Background()
		getters := map[string]client.Object{
			"ihpa-nginx":        &corev1.ServiceAccount{},
			"ihpa-nginx-status": &corev1.ConfigMap{},
		}
		for name, obj := range getters {
			if err := r.Get(ctx, types.NamespacedName{Namespace: "default", Name: name}, obj); err != nil {
				t.Errorf("failed to get %T %s: %v", obj, name, err)
			}
		}
		if err := r.Get(ctx, types.NamespacedName{Namespace: "default", Name: "ihpa-nginx"}, &rbacv1.Role{}); err != nil {
			t.Errorf("failed to get role: %v", err)
		}
		if err := r.Get(ctx, types.NamespacedName{Namespace: "default", Name: "ihpa-nginx"}, &rbacv1.RoleBinding{}); err != nil {
			t.Errorf("failed to get rolebinding: %v", err)
		}
		getHPAForTest(t, r)

		var fjs ihpav1beta2.FittingJobList
		if err := r.List(ctx, &fjs); err != nil || len(fjs.Items) != 1 {
			t.Errorf("fittingjobs = %d (err %v), want 1", len(fjs.Items), err)
		}
		var ests ihpav1beta2.EstimatorList
		if err := r.List(ctx, &ests); err != nil || len(ests.Items) != 1 {
			t.Errorf("estimators = %d (err %v), want 1", len(ests.Items), err)
		}
	})

	t.Run("success: updates the spec of the HPA", func(t *testing.T) {
		t.Parallel()

		kubeSystem, deployment := testReconcileObjects()
		r := newReconcilerForTest(t, kubeSystem, deployment, testReconcileIHPA(corev1.ResourceCPU))
		reconcileForTest(t, r)

		ihpa := &ihpav1beta2.IntelligentHorizontalPodAutoscaler{}
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "nginx"}, ihpa); err != nil {
			t.Fatal(err)
		}
		ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.MaxReplicas = 10
		if err := r.Update(context.Background(), ihpa); err != nil {
			t.Fatal(err)
		}
		reconcileForTest(t, r)

		if got := getHPAForTest(t, r).Spec.MaxReplicas; got != 10 {
			t.Errorf("hpa maxReplicas = %d, want 10", got)
		}
	})

	t.Run("success: ignores the KEDA options", func(t *testing.T) {
		t.Parallel()

		kubeSystem, deployment := testReconcileObjects()
		plain := newReconcilerForTest(t, kubeSystem.DeepCopy(), deployment.DeepCopy(), testReconcileIHPA(corev1.ResourceCPU))
		reconcileForTest(t, plain)

		withKEDA := testReconcileIHPA(corev1.ResourceCPU)
		withKEDA.Spec.ScaleBackend = ihpav1beta2.ScaleBackend{
			Type: ihpav1beta2.ScaleBackendHPA,
			KEDA: &ihpav1beta2.KEDAScaleBackend{PollingInterval: new(int32(30))},
		}
		r := newReconcilerForTest(t, kubeSystem, deployment, withKEDA)
		reconcileForTest(t, r)

		want := getHPAForTest(t, plain).Spec
		if got := getHPAForTest(t, r).Spec; !reflect.DeepEqual(got, want) {
			t.Errorf("hpa spec = %v, want %v", got, want)
		}
	})
}

func TestReconcile_KEDAMode(t *testing.T) {
	t.Parallel()

	t.Run("success: creates the ScaledObject instead of the HPA", func(t *testing.T) {
		t.Parallel()

		kubeSystem, deployment := testReconcileObjects()
		r := newReconcilerForTest(t, kubeSystem, deployment, testReconcileKEDAIHPA())

		reconcileForTest(t, r)

		so, err := getScaledObjectForTest(r)
		if err != nil {
			t.Fatalf("failed to get scaledobject: %v", err)
		}
		if got, _, _ := unstructured.NestedString(so.Object, "spec", "scaleTargetRef", "name"); got != "nginx" {
			t.Errorf("scaledobject scaleTargetRef.name = %q, want nginx", got)
		}
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "ihpa-nginx"}, &autoscalingv2.HorizontalPodAutoscaler{}); !apierrors.IsNotFound(err) {
			t.Errorf("get hpa error = %v, want NotFound", err)
		}
	})

	t.Run("success: updates the spec of the ScaledObject", func(t *testing.T) {
		t.Parallel()

		kubeSystem, deployment := testReconcileObjects()
		r := newReconcilerForTest(t, kubeSystem, deployment, testReconcileKEDAIHPA())
		reconcileForTest(t, r)

		ihpa := &ihpav1beta2.IntelligentHorizontalPodAutoscaler{}
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "nginx"}, ihpa); err != nil {
			t.Fatal(err)
		}
		ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.MaxReplicas = 10
		if err := r.Update(context.Background(), ihpa); err != nil {
			t.Fatal(err)
		}
		reconcileForTest(t, r)

		if got := scaledObjectMaxReplicasForTest(t, r); got != 10 {
			t.Errorf("scaledobject maxReplicaCount = %d, want 10", got)
		}
	})
}

func TestReconcile_ForecastPipeline(t *testing.T) {
	t.Parallel()

	t.Run("success: deletes the FittingJob and Estimator of a removed metric", func(t *testing.T) {
		t.Parallel()

		kubeSystem, deployment := testReconcileObjects()
		r := newReconcilerForTest(t, kubeSystem, deployment, testReconcileIHPA(corev1.ResourceCPU, corev1.ResourceMemory))
		reconcileForTest(t, r)

		ctx := context.Background()
		var fjs ihpav1beta2.FittingJobList
		if err := r.List(ctx, &fjs); err != nil || len(fjs.Items) != 2 {
			t.Fatalf("fittingjobs = %d (err %v), want 2", len(fjs.Items), err)
		}

		ihpa := &ihpav1beta2.IntelligentHorizontalPodAutoscaler{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: "default", Name: "nginx"}, ihpa); err != nil {
			t.Fatal(err)
		}
		ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics = ihpa.Spec.HorizontalPodAutoscalerTemplate.Spec.Metrics[:1]
		if err := r.Update(ctx, ihpa); err != nil {
			t.Fatal(err)
		}
		reconcileForTest(t, r)

		if err := r.List(ctx, &fjs); err != nil || len(fjs.Items) != 1 {
			t.Errorf("fittingjobs = %d (err %v), want 1", len(fjs.Items), err)
		}
		var ests ihpav1beta2.EstimatorList
		if err := r.List(ctx, &ests); err != nil || len(ests.Items) != 1 {
			t.Errorf("estimators = %d (err %v), want 1", len(ests.Items), err)
		}
	})

	t.Run("success: forgets the FittingJobs of a deleted IHPA", func(t *testing.T) {
		t.Parallel()

		kubeSystem, deployment := testReconcileObjects()
		ihpa := testReconcileIHPA(corev1.ResourceCPU)
		r := newReconcilerForTest(t, kubeSystem, deployment, ihpa)
		reconcileForTest(t, r)
		if len(r.fittingJobMap["default/nginx"]) != 1 {
			t.Fatalf("fittingJobMap = %v, want 1 entry for default/nginx", r.fittingJobMap)
		}

		if err := r.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "nginx"}, ihpa); err != nil {
			t.Fatal(err)
		}
		if err := r.Delete(context.Background(), ihpa); err != nil {
			t.Fatal(err)
		}
		reconcileForTest(t, r)

		if _, ok := r.fittingJobMap["default/nginx"]; ok {
			t.Errorf("fittingJobMap = %v, want no entry for default/nginx", r.fittingJobMap)
		}
	})
}
