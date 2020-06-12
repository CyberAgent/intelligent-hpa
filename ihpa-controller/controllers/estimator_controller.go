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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	mpconfig "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/config"
)

// EstimatorReconciler reconciles a Estimator object
type EstimatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	opeCh        chan<- *EstimateOperation
	estimatorChs map[string]chan<- []byte
}

// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=estimators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=estimators/status,verbs=get;update;patch

func (r *EstimatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("estimator", req.NamespacedName)

	var est ihpav1beta2.Estimator
	if err := r.Get(ctx, req.NamespacedName, &est); err != nil {
		// clean up
		if apierrors.IsNotFound(err) {
			log.V(LogicMessageLogLevel).Info("cleanup estimator", "target", req.String())
			delete(r.estimatorChs, req.String())
			r.opeCh <- &EstimateOperation{
				Operator: EstimateRemove,
				Target:   EstimateTarget{ID: req.String()},
			}

		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	g, err := NewEstimatorGenerator(&est, r)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create estimator generator: %w", err)
	}

	// * create configmap resource for receive forecasted data
	cmResource, err := g.ConfigMapResource()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate configmap resource: %w", err)
	}
	cm := &corev1.ConfigMap{}
	// if the configmap already exists, skip create/update
	// because this is changed by fittingjob.
	if err := r.Get(ctx, types.NamespacedName{Namespace: cmResource.GetNamespace(), Name: cmResource.GetName()}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(ResourceMessageLogLevel).Info("initialize configmap", "name", cmResource.GetName())
			cm = cmResource.DeepCopy()
			if err := r.Create(ctx, cm); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to create configmap: %w", err)
			}
			log.V(ResourceMessageLogLevel).Info("successed to create configmap for forecasted data", "name", cm.GetName())
		} else {
			return ctrl.Result{}, err
		}
	} else {
		log.V(ResourceMessageLogLevel).Info("configmap is already exists", "name", cm.GetName())
	}

	// * start estimate
	if _, ok := r.estimatorChs[req.String()]; !ok {
		// capacity is set to 5 but usually pick the data by estimator immediately
		dataCh := make(chan []byte, 5)

		log.V(LogicMessageLogLevel).Info("estimator added", "id", req.String(), "mode", est.Spec.Mode,
			"gapMinutes", est.Spec.GapMinutes, "metricName", est.Spec.MetricName, "metricTags", est.Spec.MetricTags,
			"baseMetricName", est.Spec.BaseMetricName, "baseMetricTags", est.Spec.BaseMetricTags)
		r.opeCh <- &EstimateOperation{
			Operator: EstimateAdd,
			Target: EstimateTarget{
				ID:             req.String(),
				EstimateMode:   est.Spec.Mode,
				GapMinutes:     int(est.Spec.GapMinutes),
				DataCh:         dataCh,
				MetricName:     est.Spec.MetricName,
				MetricTags:     est.Spec.MetricTags,
				BaseMetricName: est.Spec.BaseMetricName,
				BaseMetricTags: est.Spec.BaseMetricTags,
				MetricProvider: mpconfig.ConvertMetricProvider(est.Spec.Provider.DeepCopy()).ActiveProvider(),
			},
		}
		r.estimatorChs[req.String()] = dataCh
	} else {
		log.V(LogicMessageLogLevel).Info("estimator updated", "id", req.String(), "mode", est.Spec.Mode,
			"gapMinutes", est.Spec.GapMinutes, "metricName", est.Spec.MetricName, "metricTags", est.Spec.MetricTags,
			"baseMetricName", est.Spec.BaseMetricName, "baseMetricTags", est.Spec.BaseMetricTags)
		r.opeCh <- &EstimateOperation{
			Operator: EstimateUpdate,
			Target: EstimateTarget{
				ID:             req.String(),
				EstimateMode:   est.Spec.Mode,
				GapMinutes:     int(est.Spec.GapMinutes),
				MetricName:     est.Spec.MetricName,
				MetricTags:     est.Spec.MetricTags,
				BaseMetricName: est.Spec.BaseMetricName,
				BaseMetricTags: est.Spec.BaseMetricTags,
				MetricProvider: mpconfig.ConvertMetricProvider(est.Spec.Provider.DeepCopy()).ActiveProvider(),
			},
		}
	}

	// * check data and send the data to estimator
	key := est.Spec.BaseMetricName
	var in interface{}
	if d, ok := cm.BinaryData[key]; ok {
		in = d
	}
	if d, ok := cm.Data[key]; ok {
		in = d
	}
	var data []byte
	switch v := in.(type) {
	case string:
		data = []byte(v)
	}
	if data != nil {
		log.V(LogicMessageLogLevel).Info("new data stored", "name", req.String(), "key", key)
		dataCh := r.estimatorChs[req.String()]
		dataCh <- data
	}

	return ctrl.Result{}, nil
}

func (r *EstimatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log := r.Log.WithName("Initializer")

	r.estimatorChs = make(map[string]chan<- []byte)

	opeCh := make(chan *EstimateOperation)
	r.opeCh = opeCh
	go estimatorHandler(opeCh, r.Log.WithName("Estimator"))
	log.V(LogicMessageLogLevel).Info("start estimate handler")

	return ctrl.NewControllerManagedBy(mgr).
		For(&ihpav1beta2.Estimator{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
