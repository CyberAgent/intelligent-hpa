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

package controllers

import (
	"context"
	"fmt"
	"strings"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	"github.com/go-logr/logr"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
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

func (r *IntelligentHorizontalPodAutoscalerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("intelligenthorizontalpodautoscaler", req.NamespacedName)
	ihpaNamespacedName := req.NamespacedName.String()

	// TODO: (low) fetch datadog key from env
	// TODO: (low) determine sum/min/max/count/avg from IHPA property
	// TODO: (low) consider selector of hpa object type
	// TODO: (low) write any status to configmap

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

	// * create/update hpa resource
	hpaResource, err := g.HorizontalPodAutoscalerResource()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate hpa resource: %w", err)
	}
	hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: hpaResource.GetNamespace(), Name: hpaResource.GetName()}, hpa); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize hpa", "name", hpaResource.GetName())
		hpa = hpaResource.DeepCopy()
		if err := r.Create(ctx, hpa); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create hpa: %w", err)
		}
	} else {
		hpa.Spec = hpaResource.DeepCopy().Spec
		if err := r.Update(ctx, hpa); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update hpa: %w", err)
		}
	}
	log.V(ResourceMessageLogLevel).Info("successed to create/update hpa", "kind", hpa.GetObjectKind().GroupVersionKind(), "name", hpa.GetName())

	// * create rbac resources
	saResource, roleResource, roleBindingResource, err := g.RBACResources()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate rbac resource: %w", err)
	}
	sa := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: saResource.GetNamespace(), Name: saResource.GetName()}, sa); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize serviceAccount", "name", saResource.GetName())
		sa = saResource.DeepCopy()
		if err := r.Create(ctx, sa); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create serviceAccount: %w", err)
		}
	}
	role := &rbacv1.Role{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: roleResource.GetNamespace(), Name: roleResource.GetName()}, role); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize role", "name", roleResource.GetName())
		role = roleResource.DeepCopy()
		if err := r.Create(ctx, role); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create role: %w", err)
		}
	}
	roleBinding := &rbacv1.RoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: roleBindingResource.GetNamespace(), Name: roleBindingResource.GetName()}, roleBinding); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize roleBinding", "name", roleBindingResource.GetName())
		roleBinding = roleBindingResource.DeepCopy()
		if err := r.Create(ctx, roleBinding); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create roleBinding: %w", err)
		}
	}

	// * create fittingjob resources
	fittingJobResources, err := g.FittingJobResources()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate fittingjob resources: %w", err)
	}
	for _, fjResource := range fittingJobResources {
		fj := &ihpav1beta2.FittingJob{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: fjResource.GetNamespace(), Name: fjResource.GetName()}, fj); apierrors.IsNotFound(err) {
			log.V(ResourceMessageLogLevel).Info("initialize fittingjob", "name", fjResource.GetName())
			fj = fjResource.DeepCopy()
			if err := r.Create(ctx, fj); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to create fittingjob: %w", err)
			}
		} else {
			fj.Spec = fjResource.DeepCopy().Spec
			if err := r.Update(ctx, fj); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update fittingjob: %w", err)
			}
		}
		log.V(ResourceMessageLogLevel).Info("successed to create/update fittingjob", "kind", fj.GetObjectKind().GroupVersionKind(), "name", fj.GetName())
	}

	// * create estimator resources
	estimatorResources, err := g.EstimatorResources()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate estimator resources: %w", err)
	}
	for _, estResource := range estimatorResources {
		est := &ihpav1beta2.Estimator{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: estResource.GetNamespace(), Name: estResource.GetName()}, est); apierrors.IsNotFound(err) {
			log.V(ResourceMessageLogLevel).Info("initialize estimator", "name", estResource.GetName())
			est = estResource.DeepCopy()
			if err := r.Create(ctx, est); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to create estimator: %w", err)
			}
		} else {
			est.Spec = estResource.DeepCopy().Spec
			if err := r.Update(ctx, est); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update estimator: %w", err)
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
			return ctrl.Result{}, fmt.Errorf("failed to get list of fittingjobs: %w", err)
		}
		for _, fj := range fjs.Items {
			if id, ok := fj.GetAnnotations()[fittingJobIDAnnotation]; ok {
				if _, ok := deleteIds[id]; ok {
					if err := r.Delete(ctx, &fj); err != nil {
						return ctrl.Result{}, fmt.Errorf("failed to delete fittingjob: %w", err)
					}
					log.V(ResourceMessageLogLevel).Info("successed to delete fitting job", "kind", fj.GetObjectKind(), "name", fj.GetName(), "id", id)
				}
			}
		}

		var ests ihpav1beta2.EstimatorList
		if err := r.List(ctx, &ests); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get list of estimators: %w", err)
		}
		for _, est := range ests.Items {
			if id, ok := est.GetAnnotations()[fittingJobIDAnnotation]; ok {
				if _, ok := deleteIds[id]; ok {
					if err := r.Delete(ctx, &est); err != nil {
						return ctrl.Result{}, fmt.Errorf("failed to delete estimator: %w", err)
					}
					log.V(ResourceMessageLogLevel).Info("successed to delete estimator", "kind", est.GetObjectKind(), "name", est.GetName(), "id", id)
				}
			}
		}

		// unregister from estimator
		for id, _ := range deleteIds {
			delete(r.fittingJobMap[ihpaNamespacedName], id)
		}
	}

	// * update fittingjob ids annotation
	var fjIdsStr string
	for k, _ := range currentIds {
		fjIdsStr += k + ","
	}
	fjIdsStr = strings.TrimRight(fjIdsStr, ",")
	annotations[fittingJobIDsAnnotation] = fjIdsStr
	ihpa.SetAnnotations(annotations)
	if err := r.Update(ctx, &ihpa); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update ihpa resource: %w", err)
	}
	log.V(ResourceMessageLogLevel).Info("successed to apply annotation", fittingJobIDsAnnotation, fjIdsStr)

	return ctrl.Result{}, nil
}

func (r *IntelligentHorizontalPodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.fittingJobMap = make(map[string]map[string]struct{})

	return ctrl.NewControllerManagedBy(mgr).
		For(&ihpav1beta2.IntelligentHorizontalPodAutoscaler{}).
		Complete(r)
}
