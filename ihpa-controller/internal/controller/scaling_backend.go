package controller

import (
	"fmt"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ScalingBackend builds the resource that owns the final scaling of the target workload.
type ScalingBackend interface {
	GroupVersionKind() schema.GroupVersionKind
	Build() (client.Object, error)
}

var (
	hpaGVK          = schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}
	scaledObjectGVK = schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"}
)

// hpaBackend builds the HorizontalPodAutoscaler which has the forecasted metrics.
type hpaBackend struct {
	generator IntelligentHorizontalPodAutoscalerGenerator
}

func (hpaBackend) GroupVersionKind() schema.GroupVersionKind { return hpaGVK }

func (b hpaBackend) Build() (client.Object, error) {
	return b.generator.HorizontalPodAutoscalerResource()
}

// kedaBackend builds the KEDA ScaledObject which has the forecast triggers.
type kedaBackend struct {
	generator IntelligentHorizontalPodAutoscalerGenerator
}

func (kedaBackend) GroupVersionKind() schema.GroupVersionKind { return scaledObjectGVK }

func (b kedaBackend) Build() (client.Object, error) {
	return b.generator.ScaledObjectResource()
}

// unusedBackendGVK returns the kind of the resource that the other backend makes.
func unusedBackendGVK(used schema.GroupVersionKind) schema.GroupVersionKind {
	if used == hpaGVK {
		return scaledObjectGVK
	}
	return hpaGVK
}

// newScalingBackend returns the backend selected by scaleBackend.type of the IHPA.
// An empty type is HPA, which keeps the behavior before KEDA integration.
func newScalingBackend(ihpa *ihpav1beta2.IntelligentHorizontalPodAutoscaler, g IntelligentHorizontalPodAutoscalerGenerator) (ScalingBackend, error) {
	switch t := ihpa.Spec.ScaleBackend.Type; t {
	case ihpav1beta2.ScaleBackendHPA, "":
		return hpaBackend{generator: g}, nil
	case ihpav1beta2.ScaleBackendKEDA:
		return kedaBackend{generator: g}, nil
	default:
		return nil, fmt.Errorf("scaleBackend type %q is not supported", t)
	}
}
