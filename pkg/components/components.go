package components

import (
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Detail int

const (
	Full     Detail = iota
	Metadata Detail = iota
)

var (
	CommonLabels = k8s.StringMap{
		"app.kubernetes.io/managed-by": "kepler-operator",
		"app.kubernetes.io/part-of":    "kepler",
	}
)

const (
	Namespace = "openshift-kepler-operator"
)

func NewKeplerNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   Namespace,
			Labels: CommonLabels,
			//TODO: ensure in-cluster monitoring ignores this ns
		},
	}
}
