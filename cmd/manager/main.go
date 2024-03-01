/*
Copyright 2022.

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

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	securityv1 "github.com/openshift/api/security/v1"

	keplersystemv1alpha1 "github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/estimator"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/modelserver"
	"github.com/sustainable.computing.io/kepler-operator/pkg/controllers"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(keplersystemv1alpha1.AddToScheme(scheme))
	utilruntime.Must(securityv1.AddToScheme(scheme))
	utilruntime.Must(monv1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

type stringList []string

func (f *stringList) String() string {
	return "multiple values"
}

func (s *stringList) Set(value string) error {
	values := strings.Split(value, ",")
	*s = append(*s, values...)
	return nil
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var openshift bool
	var probeAddr string
	var additionalNamespaces stringList

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.StringVar(&controllers.KeplerDeploymentNS, "deployment-namespace", controllers.KeplerDeploymentNS,
		"Namespace where kepler and its components are deployed.")

	flag.CommandLine.Var(flag.Value(&additionalNamespaces), "watch-namespaces",
		"Namespaces other than deployment-namespace where kepler-internal may be deployed.")

	flag.BoolVar(&openshift, "openshift", false,
		"Indicate if the operator is running on an OpenShift cluster.")

	// NOTE: RELATED_IMAGE_KEPLER can be set as env or flag, flag takes precedence over env
	keplerImage := os.Getenv("RELATED_IMAGE_KEPLER")
	flag.StringVar(&controllers.Config.Image, "kepler.image", keplerImage, "kepler image")

	flag.StringVar(&controllers.InternalConfig.ModelServerImage, "estimator.image", estimator.StableImage, "kepler estimator image")
	flag.StringVar(&controllers.InternalConfig.EstimatorImage, "model-server.image", modelserver.StableImage, "kepler model server image")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if openshift {
		controllers.Config.Cluster = k8s.OpenShift
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		// TODO: add new introduced namespace from KeplerInternal.Spec.Deployment.Namespace
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			cacheNs := map[string]cache.Config{
				controllers.KeplerDeploymentNS: {},
			}
			if openshift {
				cacheNs[exporter.DashboardNs] = cache.Config{}
			}
			for _, ns := range additionalNamespaces {
				cacheNs[ns] = cache.Config{}
			}
			opts.DefaultNamespaces = cacheNs
			return cache.New(config, opts)
		},

		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0d9cbc82.sustainable.computing.io",

		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.KeplerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "kepler")
		os.Exit(1)
	}
	if err = (&controllers.KeplerInternalReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "kepler-internal")
		os.Exit(1)
	}

	// Setup webhooks
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = setupWebhooks(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupWebhooks(mgr ctrl.Manager) error {
	if err := (&keplersystemv1alpha1.Kepler{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook: %v", err)
	}
	return nil
}
