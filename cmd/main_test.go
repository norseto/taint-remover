package main

import (
	"crypto/tls"
	"reflect"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
)

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
