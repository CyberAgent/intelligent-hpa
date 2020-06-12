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

	"github.com/go-logr/logr"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
)

// FittingJobReconciler reconciles a FittingJob object
type FittingJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=fittingjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=fittingjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs/status,verbs=get

func (r *FittingJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("fittingjob", req.NamespacedName)

	var fj ihpav1beta2.FittingJob
	if err := r.Get(ctx, req.NamespacedName, &fj); err != nil {
		log.V(ResourceMessageLogLevel).Info("failed to fetch FittingJob", "error_message", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	g, err := NewFittingJobGenerator(&fj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create fittingjob resource generator: %w", err)
	}

	// * create/update configmap resource for fittingjob config
	cmResource, err := g.ConfigMapResource()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate configmap resource: %w", err)
	}
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cmResource.GetNamespace(), Name: cmResource.GetName()}, cm); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize configmap", "name", cmResource.GetName())
		cm = cmResource.DeepCopy()
		if err := r.Create(ctx, cm); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create configmap: %w", err)
		}
	} else {
		cm.Data = cmResource.DeepCopy().Data
		cm.BinaryData = cmResource.DeepCopy().BinaryData
		if err := r.Update(ctx, cm); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update configmap: %w", err)
		}
	}
	log.V(ResourceMessageLogLevel).Info("successed to create/update configmap", "kind", cm.GetObjectKind().GroupVersionKind(), "name", cm.GetName())

	// * create/update cronjob resource
	cjResource, err := g.CronJobResource()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate cronjob resource: %w", err)
	}
	cj := &batchv1beta1.CronJob{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cjResource.GetNamespace(), Name: cjResource.GetName()}, cj); apierrors.IsNotFound(err) {
		log.V(ResourceMessageLogLevel).Info("initialize cronjob", "name", cjResource.GetName())
		cj = cjResource.DeepCopy()
		if err := r.Create(ctx, cj); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create cronjob: %w", err)
		}
	} else {
		cj.Spec = cjResource.DeepCopy().Spec
		if err := r.Update(ctx, cj); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update cronjob: %w", err)
		}
	}
	log.V(ResourceMessageLogLevel).Info("successed to create/update cronjob", "kind", cj.GetObjectKind().GroupVersionKind(), "name", cj.GetName())

	return ctrl.Result{}, nil
}

func (r *FittingJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ihpav1beta2.FittingJob{}).
		Complete(r)
}
