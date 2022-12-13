package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	keplersystemv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestEnsureDaemon(t *testing.T) {
	ctx := context.Background()
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("kepler", types.NamespacedName{Name: "kepler-operator", Namespace: "kepler"})

	scheme := runtime.NewScheme()
	_ = keplersystemv1alpha1.AddToScheme(scheme)
	client := NewClient()

	keplerReconciler := &KeplerReconciler{
		Client: client,
		Scheme: scheme,
		Log:    logger,
	}

	keplerInstance := &keplersystemv1alpha1.Kepler{
		ObjectMeta: v1.ObjectMeta{
			Name:      "kepler-operator",
			Namespace: "kepler",
		},
		Spec: keplersystemv1alpha1.KeplerSpec{
			Collector: &keplersystemv1alpha1.CollectorSpec{
				Image: "quay.io/sustainable_computing_io/kepler:latest",
			},
		},
	}

	r := collectorReconciler{
		Ctx:              ctx,
		Instance:         keplerInstance,
		KeplerReconciler: *keplerReconciler,
		serviceAccount: &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kepler-operator",
				Namespace: "kepler",
			},
		},
	}

	res, err := r.ensureDaemonSet(logger)
	//basic check
	assert.Equal(t, true, res)
	if err != nil {
		t.Error("DaemonSet has failed which should not happen")
	}
	// check if CreateOrUpdate Object has properly set up fields (checking some critical fields)

	// check SetControllerReference has been set

	results, ok := client.NameSpacedNameToObject[KeplerKey{Name: "kepler-operator-exporter", Namespace: "kepler", ObjectType: "DaemonSet"}]
	if !ok {
		t.Fatal("Daemonset has not been properly created")
	}

	returnedDaemonSet, ok := results.obj.(*appsv1.DaemonSet)
	if ok {
		assert.Equal(t, "kepler-operator-exporter", returnedDaemonSet.Spec.Template.ObjectMeta.Name)
		assert.Equal(t, true, returnedDaemonSet.Spec.Template.Spec.HostNetwork)
		assert.Equal(t, r.serviceAccount.Name, returnedDaemonSet.Spec.Template.Spec.ServiceAccountName)
	} else {
		t.Error("Object is not DaemonSet")
	}
}

// func TestEnsureServiceAccount(t *testing.T) {
// 	ctx := context.Background()
// 	_ = log.FromContext(ctx)

// 	logger := log.Log.WithValues("kepler", types.NamespacedName{Name: "kepler-operator", Namespace: "kepler"})

// 	scheme := runtime.NewScheme()
// 	_ = keplersystemv1alpha1.AddToScheme(scheme)
// 	client := NewClient()
// 	// Expectations
// 	client.On("Get",
// 		mock.IsType(ctx),
// 		mock.IsType(types.NamespacedName{Namespace: "kepler", Name: "kepler-operator"}),
// 		mock.Anything,
// 	).Return(nil)

// 	client.On("Create",
// 		mock.IsType(ctx),
// 		mock.Anything,
// 	).Return(nil)

// 	client.On("Update",
// 		mock.IsType(ctx),
// 		mock.Anything,
// 	).Return(nil)
// 	keplerReconciler := &KeplerReconciler{
// 		Client: client,
// 		Scheme: scheme,
// 		Log:    logger,
// 	}

// 	keplerInstance := &keplersystemv1alpha1.Kepler{
// 		ObjectMeta: v1.ObjectMeta{
// 			Name:      "kepler-operator",
// 			Namespace: "kepler",
// 		},
// 		Spec: keplersystemv1alpha1.KeplerSpec{
// 			Collector: &keplersystemv1alpha1.CollectorSpec{
// 				Image: "quay.io/sustainable_computing_io/kepler:latest",
// 			},
// 		},
// 	}

// 	_, err := CollectorReconciler(ctx, keplerInstance, keplerReconciler, logger)
// 	require.NoError(t, err)
// 	client.AssertExpectations(t)
// }
