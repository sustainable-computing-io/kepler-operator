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

package reconciler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Action int

const (
	Continue Action = iota
	Requeue
	Stop
)

func (a Action) String() string {
	return [...]string{"Continue", "Requeue", "Stop"}[a]
}

// Result represents the result of reconciliation. Zero value indicates reconciliation ran fine
type Result struct {
	Action Action
	Error  error
}

type Reconciler interface {
	Reconcile(context.Context, client.Client, *runtime.Scheme) Result
}

type KeplerDaemonSetReconciler struct {
	Ki v1alpha1.KeplerInternal
	Ds *appsv1.DaemonSet
}

func (r KeplerDaemonSetReconciler) Reconcile(ctx context.Context, cli client.Client, s *runtime.Scheme) Result {

	secretRef := r.Ki.Spec.Exporter.Redfish.SecretRef
	secret, err := r.getRedfishSecret(ctx, cli, secretRef)

	if err != nil {
		return Result{Action: Stop, Error: fmt.Errorf("Error occured while getting secret %w", err)}
	}
	if secret == nil {
		return Result{Action: Stop, Error: fmt.Errorf("Redfish secret configured, but secret %q not found", secretRef)}
	}
	if _, ok := secret.Data[exporter.REDFISH_CSV]; !ok {
		return Result{Action: Stop, Error: fmt.Errorf("Redfish secret does not contain \"redfish.csv\"")}
	}

	exporter.MountRedfishSecretToDaemonSet(r.Ds, secret)

	return Updater{Owner: &r.Ki, Resource: r.Ds}.Reconcile(ctx, cli, s)
}

func (r KeplerDaemonSetReconciler) getRedfishSecret(ctx context.Context, cli client.Client, secretName string) (*corev1.Secret, error) {
	ns := r.Ki.Spec.Exporter.Deployment.Namespace
	redfishSecret := corev1.Secret{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: ns, Name: secretName}, &redfishSecret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &redfishSecret, nil
}

type KeplerConfigMapReconciler struct {
	Ki  v1alpha1.KeplerInternal
	Cfm *corev1.ConfigMap
}

func (r KeplerConfigMapReconciler) Reconcile(ctx context.Context, cli client.Client, s *runtime.Scheme) Result {
	rf := r.Ki.Spec.Exporter.Redfish
	zero := metav1.Duration{}
	if rf.ProbeInterval != zero {
		r.Cfm.Data["REDFISH_PROBE_INTERVAL_IN_SECONDS"] = fmt.Sprintf("%f", rf.ProbeInterval.Duration.Seconds())
	}
	r.Cfm.Data["REDFISH_SKIP_SSL_VERIFY"] = strconv.FormatBool(rf.SkipSSLVerify)
	return Updater{Owner: &r.Ki, Resource: r.Cfm}.Reconcile(ctx, cli, s)
}
