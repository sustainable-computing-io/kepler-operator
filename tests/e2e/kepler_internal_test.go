/*
Copyright 2023.

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
package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	"github.com/sustainable.computing.io/kepler-operator/pkg/controllers"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	keplerImage = `quay.io/sustainable_computing_io/kepler:release-0.7.10`
)

func TestKeplerInternal_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)
	name := "e2e-ki"
	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controllers.KeplerDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())

	// when
	b := test.InternalBuilder{}
	ki := f.CreateInternal(name,
		b.WithNamespace(testNs),
		b.WithExporterImage(keplerImage),
		b.WithExporterPort(9108),
		b.WithCluster(Cluster),
	)

	// then the following resources will be created
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(ki.Name, testNs, &ds)
	containers := ds.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(containers))
	assert.Equal(t, 1, len(containers[0].Ports))
	assert.EqualValues(t, 9108, containers[0].Ports[0].ContainerPort)
	// test expected status
	f.AssertInternalStatus(ki.Name, test.Timeout(5*time.Minute))
}

func TestKeplerInternal_ReconciliationWithRedfish(t *testing.T) {
	f := test.NewFramework(t)
	name := "e2e-ki-redfish"
	secretName := "my-redfish-secret"
	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controllers.KeplerDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())

	// when
	b := test.InternalBuilder{}
	ki := f.CreateInternal(name,
		b.WithNamespace(testNs),
		b.WithExporterImage(keplerImage),
		b.WithExporterPort(9108),
		b.WithCluster(Cluster),
		b.WithRedfish(Cluster, secretName),
	)

	// then the following resources will be created
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertNoResourceExists(ki.Name, testNs, &ds)

	// provide time for controller to reconcile
	// NOTE: reconcile should be false since the secret is not created yet
	ki = f.WaitUntilInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionFalse)
	reconciled, _ := k8s.FindCondition(ki.Status.Exporter.Conditions, v1alpha1.Reconciled)
	assert.Equal(t, fmt.Sprintf("Redfish secret %q configured, but not found in %q namespace", secretName, testNs), reconciled.Message)

	// create redfish secret
	redfishSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: controllers.KeplerDeploymentNS,
		},
		Data: map[string][]byte{
			"redfish.csv": []byte("dummy"),
		},
	}
	err := f.Client().Create(context.TODO(), &redfishSecret)
	assert.NoError(t, err)

	// wait for DaemonSet to be created
	f.AssertResourceExists(ki.Name, testNs, &ds)

	// expect reconcile to be true after secret is created
	ki = f.WaitUntilInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)

	containers := ds.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(containers))
	exp := containers[exporter.KeplerContainerIndex]
	assert.Contains(t, exp.Command, exporter.RedfishArgs)
	assert.Contains(t, exp.VolumeMounts,
		corev1.VolumeMount{Name: "redfish-cred", MountPath: "/etc/redfish", ReadOnly: true})
	assert.Contains(t, ds.Spec.Template.Spec.Volumes,
		k8s.VolumeFromSecret("redfish-cred", redfishSecret.Name))
	assert.Contains(t, ds.Spec.Template.Annotations, exporter.RedfishSecretAnnotation)

	og := ds.Status.ObservedGeneration
	assert.Equal(t, og, int64(1))

	redfishSecret.Data["redfish.csv"] = []byte("dummy2")
	err = f.Client().Update(context.TODO(), &redfishSecret)
	assert.NoError(t, err)

	// wait for DaemonSet to restart
	ds = appsv1.DaemonSet{}
	f.WaitUntil("Daemonset to restart", func() (bool, error) {
		err := f.Client().Get(context.TODO(),
			client.ObjectKey{Namespace: controllers.KeplerDeploymentNS, Name: ki.Name}, &ds)
		if errors.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return ds.Status.ObservedGeneration == og+1, nil
	})

	// test expected status
	f.AssertInternalStatus(ki.Name)
}

func TestKeplerInternal_WithEstimator(t *testing.T) {
	f := test.NewFramework(t)
	name := "e2e-ki-with-estimator"
	// Ensure Kepler is not deployed (by any chance)
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())

	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controllers.KeplerDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())
	// when
	b := test.InternalBuilder{}
	ki := f.CreateInternal(name,
		b.WithNamespace(testNs),
		b.WithExporterImage(keplerImage),
		b.WithEstimator(),
		b.WithCluster(Cluster),
	)

	// then the following resources will be created
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(ki.Name, testNs, &ds)
	containers := ds.Spec.Template.Spec.Containers
	// deamonset must have a sidecar
	assert.Equal(t, 2, len(containers))
	// test expected status
	f.AssertInternalStatus(ki.Name, test.Timeout(5*time.Minute))
}

func TestKeplerInternal_WithModelServer(t *testing.T) {
	f := test.NewFramework(t)
	name := "e2e-ki-with-modelserver"
	// Ensure Kepler is not deployed (by any chance)
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())

	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controllers.KeplerDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())
	// when
	b := test.InternalBuilder{}
	ki := f.CreateInternal(name,
		b.WithNamespace(testNs),
		b.WithExporterImage(keplerImage),
		b.WithModelServer(),
		b.WithCluster(Cluster),
	)

	// then the following resources will be created
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(ki.Name, testNs, &ds)
	containers := ds.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(containers))
	// test expected status
	f.AssertInternalStatus(ki.Name, test.Timeout(5*time.Minute))

	// confirm model-server deployment ready
	deploy := appsv1.Deployment{}
	f.AssertResourceExists(ki.ModelServerDeploymentName(), testNs, &deploy)
	readyReplicas := deploy.Status.ReadyReplicas
	assert.Equal(t, int32(1), readyReplicas)
}

func TestKeplerInternal_WithEstimatorAndModelServer(t *testing.T) {
	f := test.NewFramework(t)
	name := "e2e-ki-est-mserver"
	// Ensure Kepler is not deployed (by any chance)
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())

	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controllers.KeplerDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())
	// when
	b := test.InternalBuilder{}
	ki := f.CreateInternal(name,
		b.WithNamespace(testNs),
		b.WithExporterImage(keplerImage),
		b.WithEstimator(),
		b.WithModelServer(),
		b.WithCluster(Cluster),
	)

	// then the following resources will be created
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(ki.Name, testNs, &ds)
	containers := ds.Spec.Template.Spec.Containers
	// deamonset must have a sidecar
	assert.Equal(t, 2, len(containers))
	// test expected status
	f.AssertInternalStatus(ki.Name, test.Timeout(5*time.Minute))

	// confirm model-server deployment ready
	deploy := appsv1.Deployment{}
	f.AssertResourceExists(ki.ModelServerDeploymentName(), testNs, &deploy)
	readyReplicas := deploy.Status.ReadyReplicas
	assert.Equal(t, int32(1), readyReplicas)
}
