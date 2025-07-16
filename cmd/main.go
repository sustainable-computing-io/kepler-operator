// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
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
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	securityv1 "github.com/openshift/api/security/v1"

	keplersystemv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/version"
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
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	var additionalNamespaces stringList

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to."+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	flag.StringVar(&controller.PowerMonitorDeploymentNS, "deployment-namespace", controller.PowerMonitorDeploymentNS,
		"Namespace where power monitoring components are deployed.")

	flag.CommandLine.Var(flag.Value(&additionalNamespaces), "watch-namespaces",
		"Namespaces other than deployment-namespace where kepler-internal may be deployed.")

	flag.BoolVar(&openshift, "openshift", false,
		"Indicate if the operator is running on an OpenShift cluster.")

	// NOTE: RELATED_IMAGE_KEPLER can be set as env or flag, flag takes precedence over env
	keplerImage := os.Getenv("RELATED_IMAGE_KEPLER")
	flag.StringVar(&controller.Config.Image, "kepler.image", keplerImage, "kepler image")
	kubeRbacProxyImg := os.Getenv("RELATED_IMAGE_KUBE_RBAC_PROXY")
	flag.StringVar(&controller.Config.KubeRbacProxyImage, "kube-rbac-proxy.image", kubeRbacProxyImg, "kube rbac proxy image")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Log version info
	v := version.Info()
	setupLog.Info("Operator version info", "version",
		v.Version, "buildTime", v.BuildTime, "gitBranch",
		v.GitBranch, "gitCommit", v.GitCommit, "goVersion",
		v.GoVersion, "goOS", v.GoOS, "goArch", v.GoArch)

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	if openshift {
		controller.Config.Cluster = k8s.OpenShift
		keplersystemv1alpha1.DefaultSecurityConfig.Mode = keplersystemv1alpha1.SecurityModeRBAC
		keplersystemv1alpha1.DefaultSecurityConfig.AllowedSANames = []string{
			fmt.Sprintf("%s:%s", powermonitor.UWMNamespace, powermonitor.UWMServiceAccountName),
		}
	}

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		// TODO(user): TLSOpts is used to allow configuring the TLS config used for the server. If certificates are
		// not provided, self-signed certificates will be generated by default. This option is not recommended for
		// production environments as self-signed certificates do not offer the same level of trust and security
		// as certificates issued by a trusted Certificate Authority (CA). The primary risk is potentially allowing
		// unauthorized access to sensitive metrics data. Consider replacing with CertDir, CertName, and KeyName
		// to provide certificates, ensuring the server communicates using trusted and secure certificates.
		TLSOpts: tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsServerOptions,
		// TODO: add new introduced namespace from KeplerInternal.Spec.Deployment.Namespace
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			cacheNs := map[string]cache.Config{
				controller.PowerMonitorDeploymentNS: {},
			}
			if openshift {
				cacheNs[exporter.DashboardNs] = cache.Config{}
				cacheNs[powermonitor.UWMNamespace] = cache.Config{}
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

	if err = (&controller.KeplerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "kepler")
		os.Exit(1)
	}
	if err = (&controller.KeplerInternalReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "kepler-internal")
		os.Exit(1)
	}
	if err = (&controller.PowerMonitorReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "power-monitor")
		os.Exit(1)
	}
	if err = (&controller.PowerMonitorInternalReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "power-monitor-internal")
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
	if err := (&keplersystemv1alpha1.PowerMonitor{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook: %v", err)
	}
	return nil
}
