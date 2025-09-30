package main

import (
	"context"
	"crypto/tls"
	"errors"
	"reflect"
	"testing"

	"github.com/go-logr/logr/testr"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
)

type stubManager struct {
	addHealthzErr error
	addReadyzErr  error
	startErr      error
	started       bool
	startCtx      context.Context
}

func (s *stubManager) AddHealthzCheck(string, healthz.Checker) error {
	return s.addHealthzErr
}

func (s *stubManager) AddReadyzCheck(string, healthz.Checker) error {
	return s.addReadyzErr
}

func (s *stubManager) Start(ctx context.Context) error {
	s.started = true
	s.startCtx = ctx
	return s.startErr
}

func TestDisableHTTP2(t *testing.T) {
	logger := testr.NewWithOptions(t, testr.Options{LogTimestamp: true})
	tlsConfig := &tls.Config{}
	disableHTTP2(logger)(tlsConfig)

	if !reflect.DeepEqual(tlsConfig.NextProtos, []string{"http/1.1"}) {
		t.Fatalf("expected NextProtos to be [http/1.1], got %v", tlsConfig.NextProtos)
	}
}

func TestParseFlags(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg, err := parseFlags(nil)
		if err != nil {
			t.Fatalf("parseFlags returned error: %v", err)
		}

		if cfg.metricsAddr != ":8080" {
			t.Fatalf("expected metricsAddr :8080, got %q", cfg.metricsAddr)
		}
		if cfg.probeAddr != ":8081" {
			t.Fatalf("expected probeAddr :8081, got %q", cfg.probeAddr)
		}
		if cfg.secureMetrics {
			t.Fatalf("expected secureMetrics to be false")
		}
		if cfg.enableHTTP2 {
			t.Fatalf("expected enableHTTP2 to be false by default")
		}
		if cfg.enableLeaderElection {
			t.Fatalf("expected enableLeaderElection to be false")
		}
	})

	t.Run("custom values", func(t *testing.T) {
		args := []string{
			"--metrics-secure",
			"--enable-http2",
			"--metrics-bind-address", ":9443",
			"--health-probe-bind-address", ":9440",
			"--leader-elect",
		}
		cfg, err := parseFlags(args)
		if err != nil {
			t.Fatalf("parseFlags returned error: %v", err)
		}

		if !cfg.secureMetrics {
			t.Fatalf("expected secureMetrics true")
		}
		if !cfg.enableHTTP2 {
			t.Fatalf("expected enableHTTP2 true")
		}
		if !cfg.enableLeaderElection {
			t.Fatalf("expected enableLeaderElection true")
		}
		if cfg.metricsAddr != ":9443" {
			t.Fatalf("expected metricsAddr :9443, got %q", cfg.metricsAddr)
		}
		if cfg.probeAddr != ":9440" {
			t.Fatalf("expected probeAddr :9440, got %q", cfg.probeAddr)
		}
	})
}

func TestBuildMetricsOptions(t *testing.T) {
	t.Run("insecure metrics", func(t *testing.T) {
		tlsOption := func(*tls.Config) {}
		tlsOpts := []func(*tls.Config){tlsOption}
		bind := ":8080"

		opts := buildMetricsOptions(false, tlsOpts, bind)

		if opts.SecureServing {
			t.Fatalf("expected SecureServing to be false")
		}

		if opts.BindAddress != bind {
			t.Fatalf("expected BindAddress %q, got %q", bind, opts.BindAddress)
		}

		if len(opts.TLSOpts) != len(tlsOpts) {
			t.Fatalf("expected %d TLSOpts, got %d", len(tlsOpts), len(opts.TLSOpts))
		}

		for i := range tlsOpts {
			got := reflect.ValueOf(opts.TLSOpts[i]).Pointer()
			want := reflect.ValueOf(tlsOpts[i]).Pointer()
			if got != want {
				t.Fatalf("expected TLS option pointer %v, got %v", want, got)
			}
		}

		if opts.FilterProvider != nil {
			t.Fatalf("expected FilterProvider to be nil")
		}
	})

	t.Run("secure metrics", func(t *testing.T) {
		tlsOption := func(*tls.Config) {}
		tlsOpts := []func(*tls.Config){tlsOption}
		bind := ":8443"

		opts := buildMetricsOptions(true, tlsOpts, bind)

		if !opts.SecureServing {
			t.Fatalf("expected SecureServing to be true")
		}

		if opts.BindAddress != bind {
			t.Fatalf("expected BindAddress %q, got %q", bind, opts.BindAddress)
		}

		if len(opts.TLSOpts) != len(tlsOpts) {
			t.Fatalf("expected %d TLSOpts, got %d", len(tlsOpts), len(opts.TLSOpts))
		}

		for i := range tlsOpts {
			got := reflect.ValueOf(opts.TLSOpts[i]).Pointer()
			want := reflect.ValueOf(tlsOpts[i]).Pointer()
			if got != want {
				t.Fatalf("expected TLS option pointer %v, got %v", want, got)
			}
		}

		if opts.FilterProvider == nil {
			t.Fatalf("expected FilterProvider to be configured")
		}

		got := reflect.ValueOf(opts.FilterProvider).Pointer()
		want := reflect.ValueOf(filters.WithAuthenticationAndAuthorization).Pointer()
		if got != want {
			t.Fatalf("expected FilterProvider pointer %v, got %v", want, got)
		}
	})
}

func TestRunNewManagerError(t *testing.T) {
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(*rest.Config, ctrl.Options) (managerFacade, ctrl.Manager, error) {
		return nil, nil, errors.New("boom")
	}
	controllerSetupFn = nil
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	if code := run(nil); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestRunManagerFacadeMissing(t *testing.T) {
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(*rest.Config, ctrl.Options) (managerFacade, ctrl.Manager, error) {
		return nil, nil, nil
	}
	controllerSetupFn = nil
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	if code := run(nil); code != 1 {
		t.Fatalf("expected exit code 1 when manager facade missing, got %d", code)
	}
}

func TestRunSetupControllersError(t *testing.T) {
	stub := &stubManager{}
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(*rest.Config, ctrl.Options) (managerFacade, ctrl.Manager, error) {
		return stub, nil, nil
	}
	controllerSetupFn = func(managerFacade) error { return errors.New("controller failure") }
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	if code := run(nil); code != 1 {
		t.Fatalf("expected exit code 1 for controller setup failure, got %d", code)
	}
}

func TestRunHealthzError(t *testing.T) {
	stub := &stubManager{addHealthzErr: errors.New("healthz error")}
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(*rest.Config, ctrl.Options) (managerFacade, ctrl.Manager, error) {
		return stub, nil, nil
	}
	controllerSetupFn = nil
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	if code := run(nil); code != 1 {
		t.Fatalf("expected exit code 1 for healthz error, got %d", code)
	}
}

func TestRunReadyzError(t *testing.T) {
	stub := &stubManager{addReadyzErr: errors.New("readyz error")}
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(*rest.Config, ctrl.Options) (managerFacade, ctrl.Manager, error) {
		return stub, nil, nil
	}
	controllerSetupFn = nil
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	if code := run(nil); code != 1 {
		t.Fatalf("expected exit code 1 for readyz error, got %d", code)
	}
}

func TestRunStartError(t *testing.T) {
	stub := &stubManager{startErr: errors.New("start error")}
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(*rest.Config, ctrl.Options) (managerFacade, ctrl.Manager, error) {
		return stub, nil, nil
	}
	controllerSetupFn = nil
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	if code := run(nil); code != 1 {
		t.Fatalf("expected exit code 1 for start error, got %d", code)
	}
	if !stub.started {
		t.Fatalf("expected Start to be called")
	}
}

func TestRunSuccessWithSecureMetricsAndHTTP2(t *testing.T) {
	stub := &stubManager{}
	var captured ctrl.Options
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(_ *rest.Config, opts ctrl.Options) (managerFacade, ctrl.Manager, error) {
		captured = opts
		return stub, nil, nil
	}
	controllerSetupFn = nil
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	args := []string{
		"--metrics-secure",
		"--enable-http2",
		"--metrics-bind-address", ":8443",
		"--health-probe-bind-address", ":9440",
	}

	if code := run(args); code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !stub.started {
		t.Fatalf("expected Start to be called")
	}
	if captured.Metrics.BindAddress != ":8443" {
		t.Fatalf("expected metrics bind :8443, got %q", captured.Metrics.BindAddress)
	}
	if !captured.Metrics.SecureServing {
		t.Fatalf("expected secure metrics")
	}
	if len(captured.Metrics.TLSOpts) != 0 {
		t.Fatalf("expected no TLS opts when HTTP/2 enabled, got %d", len(captured.Metrics.TLSOpts))
	}
	if captured.HealthProbeBindAddress != ":9440" {
		t.Fatalf("expected probe bind :9440, got %q", captured.HealthProbeBindAddress)
	}
}

func TestRunDisablesHTTP2ByDefault(t *testing.T) {
	stub := &stubManager{}
	var captured ctrl.Options
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(_ *rest.Config, opts ctrl.Options) (managerFacade, ctrl.Manager, error) {
		captured = opts
		return stub, nil, nil
	}
	controllerSetupFn = nil
	signalHandlerFn = func() context.Context { return context.Background() }
	t.Cleanup(func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
	})

	if code := run(nil); code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if len(captured.Metrics.TLSOpts) != 1 {
		t.Fatalf("expected one TLS opt to disable HTTP/2, got %d", len(captured.Metrics.TLSOpts))
	}
}
