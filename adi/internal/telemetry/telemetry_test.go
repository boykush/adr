package telemetry

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/caarlos0/env/v11"
)

const serviceName = "adr-under-test"

// The variables, spelled as a deployment sets them rather than taken from
// otlpEnv, so that the tests hold the declaration to the names.
const (
	endpointEnv       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	tracesEndpointEnv = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	protocolEnv       = "OTEL_EXPORTER_OTLP_PROTOCOL"
	tracesProtocolEnv = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
)

// The SDK reads the service name once a process, so it is named before the
// first test rather than by the test that looks for it.
func TestMain(m *testing.M) {
	if err := os.Setenv("OTEL_SERVICE_NAME", serviceName); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// setEnv gives the process the OTLP environment the test names and none of the
// one the tests run in, where a collector may well be named. It is for a test
// that exports: the exporter reads the endpoint from the process's environment,
// where a map handed to adi does not reach.
func setEnv(t *testing.T, environment map[string]string) {
	t.Helper()
	for _, name := range []string{endpointEnv, tracesEndpointEnv, protocolEnv, tracesProtocolEnv} {
		t.Setenv(name, environment[name])
	}
}

// posted is one request the collector received.
type posted struct {
	path string
	body []byte
}

// newCollector stands in for the OTLP/HTTP receiver, keeping what it is sent.
func newCollector(t *testing.T) (url string, received <-chan posted) {
	t.Helper()
	requests := make(chan posted, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read the export: %v", err)
		}
		requests <- posted{path: r.URL.Path, body: body}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, requests
}

// exported returns what a shutdown sent the collector. The shutdown has
// returned by now, so an export still to come is one it did not wait for.
func exported(t *testing.T, received <-chan posted) posted {
	t.Helper()
	select {
	case got := <-received:
		return got
	default:
		t.Fatal("the shutdown returned with the span unsent")
		return posted{}
	}
}

// TestNoEndpointNoProvider is the server nobody pointed at a collector: local
// use and stdio, which must not start exporting to a default.
func TestNoEndpointNoProvider(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		// An empty map, not a missing one, which would have the process's
		// environment read.
		{name: "unset", env: map[string]string{}},
		// A variable named and left blank points at no collector either.
		{name: "blank", env: map[string]string{endpointEnv: "", tracesEndpointEnv: ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp, err := newTracerProvider(context.Background(), "test", env.Options{Environment: c.env})
			if err != nil {
				t.Fatalf("NewTracerProvider: %v", err)
			}
			if tp != nil {
				t.Error("got a tracer provider with no endpoint in the environment")
			}
		})
	}
}

// TestSpansReachTheEndpoint names the collector by each of the two variables
// and reads what arrives there.
func TestSpansReachTheEndpoint(t *testing.T) {
	cases := []struct {
		env string
		// path is appended to the collector's address in the variable.
		path     string
		wantPath string
	}{
		// The general variable is a base, to which the exporter adds the path
		// of the signal.
		{env: endpointEnv, wantPath: "/v1/traces"},
		// The traces variable is the whole address.
		{env: tracesEndpointEnv, path: "/traces", wantPath: "/traces"},
	}
	for _, c := range cases {
		t.Run(c.env, func(t *testing.T) {
			ctx := context.Background()
			url, received := newCollector(t)
			setEnv(t, map[string]string{c.env: url + c.path})

			tp, err := NewTracerProvider(ctx, "v-under-test")
			if err != nil {
				t.Fatalf("NewTracerProvider: %v", err)
			}
			_, span := tp.Tracer("test").Start(ctx, "span-under-test")
			span.End()

			// Spans leave in batches, so one this fresh is still queued. It is
			// the shutdown that has to send it.
			select {
			case <-received:
				t.Fatal("the span was exported before the shutdown")
			default:
			}
			if err := tp.Shutdown(ctx); err != nil {
				t.Fatalf("shutdown: %v", err)
			}

			got := exported(t, received)
			if got.path != c.wantPath {
				t.Errorf("exported to %s, want %s", got.path, c.wantPath)
			}
			// Protobuf carries a string as its bytes, so the export can be read
			// for them without decoding it.
			for _, want := range []string{serviceName, "v-under-test", "span-under-test"} {
				if !bytes.Contains(got.body, []byte(want)) {
					t.Errorf("the export does not carry %q", want)
				}
			}
		})
	}
}

// TestAnotherProtocolIsRefused asks, with the standard variables, for a
// protocol adi does not export over.
func TestAnotherProtocolIsRefused(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		refused bool
	}{
		{name: "general", env: map[string]string{protocolEnv: "grpc"}, refused: true},
		{name: "traces", env: map[string]string{tracesProtocolEnv: "grpc"}, refused: true},
		// The variable for traces is the more specific, and wins.
		{name: "traces over general", env: map[string]string{protocolEnv: "grpc", tracesProtocolEnv: "http/protobuf"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			c.env[endpointEnv] = "http://collector.invalid:4317"

			tp, err := newTracerProvider(ctx, "test", env.Options{Environment: c.env})
			if !c.refused {
				if err != nil {
					t.Fatalf("NewTracerProvider: %v", err)
				}
				if err := tp.Shutdown(ctx); err != nil {
					t.Fatalf("shutdown: %v", err)
				}
				return
			}
			if tp != nil {
				t.Error("got a tracer provider for a protocol adi does not export over")
			}
			// Whoever set the variable needs to hear which value was turned down.
			if err == nil || !strings.Contains(err.Error(), `"grpc"`) {
				t.Errorf("err = %v, want it to name the protocol", err)
			}
		})
	}
}
