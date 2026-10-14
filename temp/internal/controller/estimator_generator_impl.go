package controllers

import (
	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type estimatorGeneratorImpl struct {
	// est is a struct of target est
	est    *ihpav1beta2.Estimator
	scheme *runtime.Scheme
}

func NewEstimatorGenerator(
	est *ihpav1beta2.Estimator,
	r *EstimatorReconciler,
) (EstimatorGenerator, error) {
	return &estimatorGeneratorImpl{est: est, scheme: r.Scheme}, nil
}

// ConfigMapResource generate empty ConfigMap for storing forecast data given by fittingjob.
func (g *estimatorGeneratorImpl) ConfigMapResource() (*corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.est.Spec.DataConfigMap.Name,
			Namespace: g.est.GetNamespace(),
		},
	}

	if err := ctrlutil.SetControllerReference(g.est, &cm, g.scheme); err != nil {
		return nil, err
	}

	return &cm, nil
}
