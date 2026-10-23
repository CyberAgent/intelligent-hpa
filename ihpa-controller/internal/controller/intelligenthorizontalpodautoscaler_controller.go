/*
Copyright 2020 SIA Platform Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// unusedBackendDeletionPollInterval is the interval to check whether the resource of the other backend is gone.
	unusedBackendDeletionPollInterval = 5 * time.Second

	annotationPrefix        = "ihpa.ake.cyberagent.co.jp"
	fittingJobIDAnnotation  = annotationPrefix + "/fittingjob-id"
	fittingJobIDsAnnotation = annotationPrefix + "/fittingjob-ids"

	ResourceMessageLogLevel = 1
	LogicMessageLogLevel    = 1
)

// IntelligentHorizontalPodAutoscalerReconciler reconciles a IntelligentHorizontalPodAutoscaler object
type IntelligentHorizontalPodAutoscalerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	// fittingJobMap is used for cleanup all fittingjob when ihpa resource is deleted.
	// Any fittingjob can be deleted when some ihpa metric field is deleted because of comparison fittingJobIDsAnnotation,
	// but if ihpa is deleted, this controller cannot delete fittingjob because controller cannot refer to the annotation.
	fittingJobMap map[string]map[string]struct{}
}

// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=intelligenthorizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=intelligenthorizontalpodautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *IntelligentHorizontalPodAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("intelligenthorizontalpodautoscaler", req.NamespacedName)
	ctx = ctrl.LoggerInto(ctx, log)
	ihpaNamespacedName := req.String()

	var ihpa ihpav1beta2.IntelligentHorizontalPodAutoscaler
	if err := r.Get(ctx, req.NamespacedName, &ihpa); err != nil {
		log.V(ResourceMessageLogLevel).Info("failed to fetch IntelligentHorizontalPodAutoscaler", "error_message", err)
		if apierrors.IsNotFound(err) {
			log.V(LogicMessageLogLevel).Info("cleanup estimator", "target", ihpaNamespacedName)
			delete(r.fittingJobMap, ihpaNamespacedName)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if _, ok := r.fittingJobMap[ihpaNamespacedName]; !ok {
		r.fittingJobMap[ihpaNamespacedName] = make(map[string]struct{})
	}

	g, err := NewIntelligentHorizontalPodAutoscalerGenerator(&ihpa, r, ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create ihpa manager: %w", err)
	}

	if result, err := r.reconcileAutoscaler(ctx, g, &ihpa); err != nil || !result.IsZero() {
		return result, err
	}
	if err := r.reconcileForecastPipeline(ctx, g, &ihpa, ihpaNamespacedName); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileInventory(ctx, g); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcileAutoscaler creates or updates the resource that scales the target workload,
// after deleting the resource of the other backend because two autoscalers must not control the same workload.
// It asks to be called again while the resource of the other backend is being deleted.
func (r *IntelligentHorizontalPodAutoscalerReconciler) reconcileAutoscaler(ctx context.Context, g IntelligentHorizontalPodAutoscalerGenerator, ihpa *ihpav1beta2.IntelligentHorizontalPodAutoscaler) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	backend, err := newScalingBackend(ihpa, g)
	if err != nil {
		return ctrl.Result{}, err
	}
	gvk := backend.GroupVersionKind()

	autoscaler, err := backend.Build()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate %s resource: %w", gvk.Kind, err)
	}

	// Both backends are applied as unstructured by their GroupVersionKind, so that one code path serves them.
	// Typed handling per backend would duplicate the get/create/update/delete logic.
	// The KEDA types are still used to generate the ScaledObject.
	// (K8s 1.25+ removed autoscaling/v2beta2, so we use autoscaling/v2)
	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(autoscaler)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to convert %s to unstructured: %w", gvk.Kind, err)
	}
	desired := &unstructured.Unstructured{Object: objMap}
	desired.SetGroupVersionKind(gvk)

	// The target kind is looked up before deleting the unused backend,
	// because the workload must not lose its autoscaler when the target backend cannot be used, such as KEDA not being installed.
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(gvk)
	getErr := r.Get(ctx, client.ObjectKeyFromObject(desired), current)
	if getErr != nil && !apierrors.IsNotFound(getErr) {
		return ctrl.Result{}, fmt.Errorf("failed to get %s: %w", gvk.Kind, getErr)
	}

	deleting, err := r.deleteControlledResource(ctx, ihpa, unusedBackendGVK(gvk), client.ObjectKeyFromObject(desired))
	if err != nil {
		return ctrl.Result{}, err
	}
	if deleting {
		return ctrl.Result{RequeueAfter: unusedBackendDeletionPollInterval}, nil
	}

	if apierrors.IsNotFound(getErr) {
		log.V(ResourceMessageLogLevel).Info("initialize autoscaler", "kind", gvk.Kind, "name", desired.GetName())
		if err := r.Create(ctx, desired); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create %s: %w", gvk.Kind, err)
		}
		current = desired
	} else {
		// Updating the spec cannot cancel the deletion, e.g. when a switch of the backend is reverted at once,
		// so wait for the deletion and create it again.
		if current.GetDeletionTimestamp() != nil {
			log.V(LogicMessageLogLevel).Info("wait for the autoscaler being deleted", "kind", gvk.Kind, "name", current.GetName())
			return ctrl.Result{RequeueAfter: unusedBackendDeletionPollInterval}, nil
		}
		current.Object["spec"] = desired.Object["spec"]
		if err := r.Update(ctx, current); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update %s: %w", gvk.Kind, err)
		}
	}
	log.V(ResourceMessageLogLevel).Info("successed to create/update autoscaler", "kind", gvk.Kind, "name", current.GetName())
	return ctrl.Result{}, nil
}

// deleteControlledResource deletes the resource if the IHPA controls it, and reports whether it is still being deleted.
// A resource of the kind that the cluster does not have, such as a ScaledObject without KEDA, is regarded as deleted.
// A resource that the IHPA does not control is left as it is, because it belongs to someone else.
func (r *IntelligentHorizontalPodAutoscalerReconciler) deleteControlledResource(ctx context.Context, ihpa *ihpav1beta2.IntelligentHorizontalPodAutoscaler, gvk schema.GroupVersionKind, key types.NamespacedName) (bool, error) {
	log := ctrl.LoggerFrom(ctx)
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	if err := r.Get(ctx, key, obj); apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get %s: %w", gvk.Kind, err)
	}

	if !metav1.IsControlledBy(obj, ihpa) {
		log.V(LogicMessageLogLevel).Info("skip deleting the resource which the IHPA does not control", "kind", gvk.Kind, "name", key.Name)
		return false, nil
	}
	if obj.GetDeletionTimestamp() == nil {
		// The precondition binds the deletion to the checked resource,
		// so a replacement of the same name created by someone else in the meantime is not deleted.
		if err := r.Delete(ctx, obj, client.Preconditions{UID: new(obj.GetUID())}); client.IgnoreNotFound(err) != nil {
			return false, fmt.Errorf("failed to delete %s: %w", gvk.Kind, err)
		}
		log.V(ResourceMessageLogLevel).Info("successed to delete the resource of the other backend", "kind", gvk.Kind, "name", key.Name)
	}

	// A ScaledObject is not deleted at once because KEDA has a finalizer on it.
	// Its deletion is polled by RequeueAfter instead of watched, because registering Owns(ScaledObject)
	// makes the manager fail to start on a cluster without KEDA, which must keep working in HPA mode.
	if err := r.Get(ctx, key, obj); apierrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get %s: %w", gvk.Kind, err)
	}
	return true, nil
}

// reconcileForecastPipeline creates or updates the resources that generate and send the forecasted metrics,
// and deletes the ones of metrics which are removed from the IHPA.
//
//nolint:gocyclo // This is a move of the former Reconcile as it was, so splitting it further is left out of the KEDA support.
func (r *IntelligentHorizontalPodAutoscalerReconciler) reconcileForecastPipeline(ctx context.Context, g IntelligentHorizontalPodAutoscalerGenerator, ihpa *ihpav1beta2.IntelligentHorizontalPodAutoscaler, ihpaNamespacedName string) error {
	log := ctrl.LoggerFrom(ctx)
	// * create rbac resources
	saResource, roleResource, roleBindingResource, err := g.RBACResources()
	if err != nil {
		return fmt.Errorf("failed to generate rbac resource: %w", err)
	}
	sa := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: saResource.GetNamespace(), Name: saResource.GetName()}, sa); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize serviceAccount", "name", saResource.GetName())
		sa = saResource.DeepCopy()
		if err := r.Create(ctx, sa); err != nil {
			return fmt.Errorf("failed to create serviceAccount: %w", err)
		}
	}
	role := &rbacv1.Role{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: roleResource.GetNamespace(), Name: roleResource.GetName()}, role); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize role", "name", roleResource.GetName())
		role = roleResource.DeepCopy()
		if err := r.Create(ctx, role); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	}
	roleBinding := &rbacv1.RoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: roleBindingResource.GetNamespace(), Name: roleBindingResource.GetName()}, roleBinding); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize roleBinding", "name", roleBindingResource.GetName())
		roleBinding = roleBindingResource.DeepCopy()
		if err := r.Create(ctx, roleBinding); err != nil {
			return fmt.Errorf("failed to create roleBinding: %w", err)
		}
	}

	// * create fittingjob resources
	fittingJobResources, err := g.FittingJobResources()
	if err != nil {
		return fmt.Errorf("failed to generate fittingjob resources: %w", err)
	}
	for _, fjResource := range fittingJobResources {
		fj := &ihpav1beta2.FittingJob{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: fjResource.GetNamespace(), Name: fjResource.GetName()}, fj); apierrors.IsNotFound(err) {
			log.V(ResourceMessageLogLevel).Info("initialize fittingjob", "name", fjResource.GetName())
			fj = fjResource.DeepCopy()
			if err := r.Create(ctx, fj); err != nil {
				return fmt.Errorf("failed to create fittingjob: %w", err)
			}
		} else {
			fj.Spec = fjResource.DeepCopy().Spec
			if err := r.Update(ctx, fj); err != nil {
				return fmt.Errorf("failed to update fittingjob: %w", err)
			}
		}
		log.V(ResourceMessageLogLevel).Info("successed to create/update fittingjob", "kind", fj.GetObjectKind().GroupVersionKind(), "name", fj.GetName())
	}

	// * create estimator resources
	estimatorResources, err := g.EstimatorResources()
	if err != nil {
		return fmt.Errorf("failed to generate estimator resources: %w", err)
	}
	for _, estResource := range estimatorResources {
		est := &ihpav1beta2.Estimator{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: estResource.GetNamespace(), Name: estResource.GetName()}, est); apierrors.IsNotFound(err) {
			log.V(ResourceMessageLogLevel).Info("initialize estimator", "name", estResource.GetName())
			est = estResource.DeepCopy()
			if err := r.Create(ctx, est); err != nil {
				return fmt.Errorf("failed to create estimator: %w", err)
			}
		} else {
			est.Spec = estResource.DeepCopy().Spec
			if err := r.Update(ctx, est); err != nil {
				return fmt.Errorf("failed to update estimator: %w", err)
			}
		}
		log.V(ResourceMessageLogLevel).Info("successed to create/update estimator", "kind", est.GetObjectKind().GroupVersionKind(), "name", est.GetName())
	}

	// * get all fittingjob id related to this ihpa
	currentIds := make(map[string]struct{}, len(fittingJobResources))
	for _, fj := range fittingJobResources {
		id, ok := fj.GetAnnotations()[fittingJobIDAnnotation]
		if !ok {
			continue
		}
		currentIds[id] = struct{}{}
		r.fittingJobMap[ihpaNamespacedName][id] = struct{}{}
	}

	annotations := ihpa.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// * delete fittingjob and estimator
	if fjIdsStr, ok := annotations[fittingJobIDsAnnotation]; ok {
		previousIds := strings.Split(fjIdsStr, ",")
		deleteIds := make(map[string]struct{}, len(previousIds))
		for _, id := range previousIds {
			if _, ok := currentIds[id]; !ok {
				deleteIds[id] = struct{}{}
			}
		}

		var fjs ihpav1beta2.FittingJobList
		if err := r.List(ctx, &fjs); err != nil {
			return fmt.Errorf("failed to get list of fittingjobs: %w", err)
		}
		for _, fj := range fjs.Items {
			if id, ok := fj.GetAnnotations()[fittingJobIDAnnotation]; ok {
				if _, ok := deleteIds[id]; ok {
					if err := r.Delete(ctx, &fj); err != nil {
						return fmt.Errorf("failed to delete fittingjob: %w", err)
					}
					log.V(ResourceMessageLogLevel).Info("successed to delete fitting job", "kind", fj.GetObjectKind(), "name", fj.GetName(), "id", id)
				}
			}
		}

		var ests ihpav1beta2.EstimatorList
		if err := r.List(ctx, &ests); err != nil {
			return fmt.Errorf("failed to get list of estimators: %w", err)
		}
		for _, est := range ests.Items {
			if id, ok := est.GetAnnotations()[fittingJobIDAnnotation]; ok {
				if _, ok := deleteIds[id]; ok {
					if err := r.Delete(ctx, &est); err != nil {
						return fmt.Errorf("failed to delete estimator: %w", err)
					}
					log.V(ResourceMessageLogLevel).Info("successed to delete estimator", "kind", est.GetObjectKind(), "name", est.GetName(), "id", id)
				}
			}
		}

		// unregister from estimator
		for id := range deleteIds {
			delete(r.fittingJobMap[ihpaNamespacedName], id)
		}
	}

	// * update fittingjob ids annotation
	var fjIdsStr string
	for k := range currentIds {
		fjIdsStr += k + ","
	}
	fjIdsStr = strings.TrimRight(fjIdsStr, ",")
	annotations[fittingJobIDsAnnotation] = fjIdsStr
	ihpa.SetAnnotations(annotations)
	if err := r.Update(ctx, ihpa); err != nil {
		return fmt.Errorf("failed to update ihpa resource: %w", err)
	}
	log.V(ResourceMessageLogLevel).Info("successed to apply annotation", "annotation", fittingJobIDsAnnotation, "value", fjIdsStr)
	return nil
}

// reconcileInventory keeps the ConfigMap that lets users find the generated resources, as the IHPA has no status field for them.
func (r *IntelligentHorizontalPodAutoscalerReconciler) reconcileInventory(ctx context.Context, g IntelligentHorizontalPodAutoscalerGenerator) error {
	log := ctrl.LoggerFrom(ctx)
	// * create/update status configmap
	statusCM, err := g.StatusConfigMapResource()
	if err != nil {
		return fmt.Errorf("failed to generate status configmap: %w", err)
	}
	existingCM := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: statusCM.GetNamespace(), Name: statusCM.GetName()}, existingCM); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize status configmap", "name", statusCM.GetName())
		if err := r.Create(ctx, statusCM); err != nil {
			return fmt.Errorf("failed to create status configmap: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get status configmap: %w", err)
	} else {
		existingCM.Data = statusCM.Data
		if err := r.Update(ctx, existingCM); err != nil {
			return fmt.Errorf("failed to update status configmap: %w", err)
		}
	}
	log.V(ResourceMessageLogLevel).Info("successed to create/update status configmap", "name", statusCM.GetName())
	return nil
}

func (r *IntelligentHorizontalPodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.fittingJobMap = make(map[string]map[string]struct{})

	// Owns(ScaledObject) is not registered, see deleteControlledResource.
	return ctrl.NewControllerManagedBy(mgr).
		For(&ihpav1beta2.IntelligentHorizontalPodAutoscaler{}).
		Complete(r)
}
