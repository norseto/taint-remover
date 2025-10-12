package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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

type stubCtrlManager struct {
	*stubManager
	scheme  *runtime.Scheme
	cfg     *rest.Config
	elected chan struct{}
}

func newStubCtrlManager() *stubCtrlManager {
	return &stubCtrlManager{
		stubManager: &stubManager{},
		scheme:      runtime.NewScheme(),
		cfg:         &rest.Config{},
		elected:     make(chan struct{}),
	}
}

func (s *stubCtrlManager) Add(manager.Runnable) error {
	return nil
}

func (s *stubCtrlManager) Elected() <-chan struct{} {
	return s.elected
}

func (s *stubCtrlManager) AddMetricsServerExtraHandler(string, http.Handler) error {
	return nil
}

func (s *stubCtrlManager) GetWebhookServer() webhook.Server {
	return webhook.NewServer(webhook.Options{})
}

func (s *stubCtrlManager) GetLogger() logr.Logger {
	return logr.Logger{}
}

func (s *stubCtrlManager) GetControllerOptions() config.Controller {
	return config.Controller{}
}

func (s *stubCtrlManager) GetHTTPClient() *http.Client {
	return &http.Client{}
}

func (s *stubCtrlManager) GetConfig() *rest.Config {
	return s.cfg
}

func (s *stubCtrlManager) GetCache() cache.Cache {
	return nil
}

func (s *stubCtrlManager) GetScheme() *runtime.Scheme {
	return s.scheme
}

func (s *stubCtrlManager) GetClient() client.Client {
	return nil
}

func (s *stubCtrlManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

func (s *stubCtrlManager) GetEventRecorderFor(string) record.EventRecorder {
	return nil
}

func (s *stubCtrlManager) GetRESTMapper() meta.RESTMapper {
	return nil
}

func (s *stubCtrlManager) GetAPIReader() client.Reader {
	return nil
}

type recordingReconciler struct {
	called bool
	err    error
}

func (r *recordingReconciler) SetupWithManager(ctrl.Manager) error {
	r.called = true
	return r.err
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

func TestParseFlagsInvalidValue(t *testing.T) {
	if _, err := parseFlags([]string{"--metrics-secure=notabool"}); err == nil {
		t.Fatalf("expected error for invalid boolean flag")
	}
}

func TestRunParseFlagsError(t *testing.T) {
	if code := run([]string{"--unknown"}); code != 2 {
		t.Fatalf("expected exit code 2 for flag parse error, got %d", code)
	}
}

func TestRunGetConfigError(t *testing.T) {
	origGetConfig := getConfigFn
	getConfigFn = func() (*rest.Config, error) { return nil, errors.New("cfg err") }
	t.Cleanup(func() { getConfigFn = origGetConfig })

	if code := run(nil); code != 1 {
		t.Fatalf("expected exit code 1 when config retrieval fails, got %d", code)
	}
}

func TestDefaultManagerFactory(t *testing.T) {
	origNewManager := newManagerFn
	defer func() { newManagerFn = origNewManager }()

	fake := newStubCtrlManager()
	newManagerFn = func(*rest.Config, ctrl.Options) (ctrl.Manager, error) {
		return fake, nil
	}

	facade, mgr, err := defaultManagerFactory(&rest.Config{}, ctrl.Options{})
	if err != nil {
		t.Fatalf("defaultManagerFactory returned error: %v", err)
	}
	if facade != fake || mgr != fake {
		t.Fatalf("expected returned manager to be fake instance")
	}
}

func TestDefaultManagerFactoryError(t *testing.T) {
	origNewManager := newManagerFn
	defer func() { newManagerFn = origNewManager }()

	newManagerFn = func(*rest.Config, ctrl.Options) (ctrl.Manager, error) {
		return nil, errors.New("boom")
	}

	if _, _, err := defaultManagerFactory(&rest.Config{}, ctrl.Options{}); err == nil {
		t.Fatalf("expected error when manager creation fails")
	}
}

func TestDefaultControllerSetupManagerUnavailable(t *testing.T) {
	if err := defaultControllerSetup(&stubManager{}); err == nil {
		t.Fatalf("expected error when facade does not implement ctrl.Manager")
	}
}

func TestDefaultControllerSetupSuccess(t *testing.T) {
	origNewReconciler := newTaintRemoverReconciler
	defer func() { newTaintRemoverReconciler = origNewReconciler }()

	fakeMgr := newStubCtrlManager()
	rec := &recordingReconciler{}

	newTaintRemoverReconciler = func(ctrl.Manager) reconcilerWithSetup {
		return rec
	}

	if err := defaultControllerSetup(fakeMgr); err != nil {
		t.Fatalf("expected success from defaultControllerSetup, got error: %v", err)
	}
	if !rec.called {
		t.Fatalf("expected reconciler SetupWithManager to be invoked")
	}
}

func TestDefaultControllerSetupNilManager(t *testing.T) {
	var nilFacade managerFacade = (*stubCtrlManager)(nil)
	if err := defaultControllerSetup(nilFacade); err == nil {
		t.Fatalf("expected error when manager is nil")
	}
}

func TestMainSuccess(t *testing.T) {
	origGetConfig := getConfigFn
	origManagerFactory := managerFactory
	origControllerSetup := controllerSetupFn
	origSignalHandler := signalHandlerFn
	origNewManager := newManagerFn
	origNewReconciler := newTaintRemoverReconciler
	origArgs := os.Args
	origExit := exitFunc

	defer func() {
		getConfigFn = origGetConfig
		managerFactory = origManagerFactory
		controllerSetupFn = origControllerSetup
		signalHandlerFn = origSignalHandler
		newManagerFn = origNewManager
		newTaintRemoverReconciler = origNewReconciler
		os.Args = origArgs
		exitFunc = origExit
	}()

	fakeMgr := newStubCtrlManager()
	controllerCalled := false
	getConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	managerFactory = func(*rest.Config, ctrl.Options) (managerFacade, ctrl.Manager, error) {
		return fakeMgr, fakeMgr, nil
	}
	controllerSetupFn = func(m managerFacade) error {
		if m != fakeMgr {
			t.Fatalf("expected manager facade to be fakeMgr")
		}
		controllerCalled = true
		return defaultControllerSetup(m)
	}
	signalHandlerFn = func() context.Context { return context.Background() }
	newManagerFn = func(*rest.Config, ctrl.Options) (ctrl.Manager, error) {
		return fakeMgr, nil
	}
	rec := &recordingReconciler{}
	newTaintRemoverReconciler = func(ctrl.Manager) reconcilerWithSetup {
		return rec
	}

	os.Args = []string{"taint-remover"}
	main()

	if !controllerCalled {
		t.Fatalf("expected controllerSetupFn to be invoked")
	}
	if !rec.called {
		t.Fatalf("expected reconciler to be constructed and invoked")
	}
}

func TestMainExitOnError(t *testing.T) {
	origGetConfig := getConfigFn
	origExit := exitFunc
	origArgs := os.Args

	defer func() {
		getConfigFn = origGetConfig
		exitFunc = origExit
		os.Args = origArgs
	}()

	getConfigFn = func() (*rest.Config, error) { return nil, errors.New("cfg err") }
	called := 0
	exitCode := 0
	exitFunc = func(code int) {
		called++
		exitCode = code
	}
	os.Args = []string{"taint-remover"}

	main()

	if called != 1 {
		t.Fatalf("expected exitFunc to be called once, got %d", called)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}
