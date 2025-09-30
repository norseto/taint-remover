/*
MIT License

Copyright (c) 2023-2025 Norihiro Seto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"io"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	taintremover "github.com/norseto/taint-remover"
	nodesv1alpha1 "github.com/norseto/taint-remover/api/v1alpha1"
	"github.com/norseto/taint-remover/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

type cliConfig struct {
	metricsAddr          string
	probeAddr            string
	secureMetrics        bool
	enableLeaderElection bool
	enableHTTP2          bool
	zapOptions           zap.Options
}

type managerFacade interface {
	AddHealthzCheck(name string, check healthz.Checker) error
	AddReadyzCheck(name string, check healthz.Checker) error
	Start(ctx context.Context) error
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nodesv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func disableHTTP2(logger logr.Logger) func(*tls.Config) {
	return func(c *tls.Config) {
		logger.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}
}

func parseFlags(args []string) (cliConfig, error) {
	fs := flag.NewFlagSet("taint-remover", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	cfg := cliConfig{
		metricsAddr: ":8080",
		probeAddr:   ":8081",
		zapOptions: zap.Options{
			Development: false,
		},
	}

	fs.BoolVar(&cfg.secureMetrics, "metrics-secure", false, "If set the metrics endpoint is served securely")
	fs.BoolVar(&cfg.enableHTTP2, "enable-http2", false, "If HTTP/2 should be enabled for the metrics and webhook servers.")
	fs.StringVar(&cfg.metricsAddr, "metrics-bind-address", cfg.metricsAddr, "The address the metric endpoint binds to.")
	fs.StringVar(&cfg.probeAddr, "health-probe-bind-address", cfg.probeAddr, "The address the probe endpoint binds to.")
	fs.BoolVar(&cfg.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	cfg.zapOptions.BindFlags(fs)

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func run(args []string) int {
	cfg, err := parseFlags(args)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			setupLog.Error(err, "failed to parse flags")
		}
		return 2
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&cfg.zapOptions)))

	ctrl.Log.Info("Starting TaintRemover", "version", taintremover.RELEASE_VERSION,
		"GitVersion", taintremover.GitVersion)

	var tlsOpts []func(*tls.Config)
	if !cfg.enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2(setupLog))
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	metricsServerOptions := buildMetricsOptions(cfg.secureMetrics, tlsOpts, cfg.metricsAddr)

	restCfg, err := getConfigFn()
	if err != nil {
		setupLog.Error(err, "unable to get Kubernetes configuration")
		return 1
	}

	managerOpts := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		HealthProbeBindAddress: cfg.probeAddr,
		WebhookServer:          webhookServer,
		LeaderElection:         cfg.enableLeaderElection,
		LeaderElectionID:       "cab18bf0.peppy-ratio.dev",
	}

	facade, _, err := managerFactory(restCfg, managerOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return 1
	}

	if facade == nil {
		setupLog.Error(errors.New("manager facade is nil"), "unable to start manager")
		return 1
	}

	if controllerSetupFn != nil {
		if err := controllerSetupFn(facade); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "TaintRemover")
			return 1
		}
	}

	if err := facade.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return 1
	}
	if err := facade.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return 1
	}

	setupLog.Info("starting manager")
	handler := signalHandlerFn
	if handler == nil {
		handler = ctrl.SetupSignalHandler
	}
	ctx := handler()
	if err := facade.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return 1
	}

	return 0
}

var (
	getConfigFn       = ctrl.GetConfig
	managerFactory    = defaultManagerFactory
	controllerSetupFn = defaultControllerSetup
	signalHandlerFn   = ctrl.SetupSignalHandler
)

func defaultManagerFactory(cfg *rest.Config, opts ctrl.Options) (managerFacade, ctrl.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		return nil, nil, err
	}
	return mgr, mgr, nil
}

func defaultControllerSetup(f managerFacade) error {
	mgr, ok := any(f).(ctrl.Manager)
	if !ok {
		return errors.New("controller manager unavailable")
	}
	if mgr == nil {
		return errors.New("controller manager is nil")
	}
	return (&controller.TaintRemoverReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
}

func buildMetricsOptions(secure bool, tlsOpts []func(*tls.Config), bind string) metricsserver.Options {
	options := metricsserver.Options{
		BindAddress:   bind,
		SecureServing: secure,
		TLSOpts:       tlsOpts,
	}

	if secure {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		options.FilterProvider = filters.WithAuthenticationAndAuthorization

		// TODO(user): If CertDir, CertName, and KeyName are not specified, controller-runtime will automatically
		// generate self-signed certificates for the metrics server. While convenient for development and testing,
		// this setup is not recommended for production.
	}

	return options
}

func main() {
	if exitCode := run(os.Args[1:]); exitCode != 0 {
		os.Exit(exitCode)
	}
}
