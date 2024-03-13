/*
Copyright 2024.

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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KeplerReconciler struct {
	Ki *v1alpha1.KeplerInternal
	Ds *appsv1.DaemonSet
}

func (r KeplerReconciler) Reconcile(ctx context.Context, cli client.Client, s *runtime.Scheme) Result {
	redfish := r.Ki.Spec.Exporter.Redfish
	redfishBytes, err := json.Marshal(redfish)
	if err != nil {
		return Result{Action: Stop, Error: fmt.Errorf("Error occurred while marshaling Redfish spec %w", err)}
	}

	secretRef := redfish.SecretRef
	secret, err := r.getRedfishSecret(ctx, cli, secretRef)

	if err != nil {
		return Result{Action: Stop, Error: fmt.Errorf("Error occurred while getting Redfish secret %w", err)}
	}

	if secret == nil {
		return Result{
			Action: Stop,
			Error:  fmt.Errorf("Redfish secret %q configured, but not found in %q namespace", secretRef, r.Ki.Namespace()),
		}
	}
	if _, ok := secret.Data[exporter.RedfishCSV]; !ok {
		return Result{Action: Stop, Error: fmt.Errorf("Redfish secret is missing %q key", exporter.RedfishCSV)}
	}

	redfishHash := xxhash.Sum64(redfishBytes)
	exporter.MountRedfishSecretToDaemonSet(r.Ds, secret, redfishHash)
	return Updater{Owner: r.Ki, Resource: r.Ds}.Reconcile(ctx, cli, s)
}

func (r KeplerReconciler) getRedfishSecret(ctx context.Context, cli client.Client, secretName string) (*corev1.Secret, error) {
	ns := r.Ki.Spec.Exporter.Deployment.Namespace
	redfishSecret := corev1.Secret{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: ns, Name: secretName}, &redfishSecret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &redfishSecret, nil
}

type KeplerConfigMapReconciler struct {
	Ki  *v1alpha1.KeplerInternal
	Cfm *corev1.ConfigMap
}

func (r KeplerConfigMapReconciler) Reconcile(ctx context.Context, cli client.Client, s *runtime.Scheme) Result {
	rf := r.Ki.Spec.Exporter.Redfish
	r.Cfm.Data["REDFISH_PROBE_INTERVAL_IN_SECONDS"] = strconv.FormatFloat(rf.ProbeInterval.Duration.Seconds(), 'f', 0, 64)
	r.Cfm.Data["REDFISH_SKIP_SSL_VERIFY"] = strconv.FormatBool(rf.SkipSSLVerify)
	return Updater{Owner: r.Ki, Resource: r.Cfm}.Reconcile(ctx, cli, s)
}
