package framework

import (
	"context"
	"testing"

	"k8s.io/client-go/rest"

	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	kubernetes kubernetes.Interface
	Config     *rest.Config
	K8sClient  client.Client
	Retain     bool
}

func (f *Framework) getKubernetesClient() (kubernetes.Interface, error) {
	if f.kubernetes == nil {
		c, err := kubernetes.NewForConfig(f.Config)
		if err != nil {
			return nil, err
		}
		f.kubernetes = c
	}

	return f.kubernetes, nil
}

func (f *Framework) Evict(pod *corev1.Pod, gracePeriodSeconds int64) error {
	delOpts := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	eviction := &policyv1beta1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyv1beta1.SchemeGroupVersion.String(),
			Kind:       "Eviction",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: &delOpts,
	}

	c, err := f.getKubernetesClient()
	if err != nil {
		return err
	}
	return c.PolicyV1beta1().Evictions(pod.Namespace).Evict(context.Background(), eviction)
}

func (f *Framework) CleanUp(t *testing.T, cleanupFunc func()) {
	t.Cleanup(func() {
		testSucceeded := !t.Failed()
		if testSucceeded || !f.Retain {
			cleanupFunc()
		}
	})
}
