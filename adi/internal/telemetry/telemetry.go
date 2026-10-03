// Package telemetry exports adi's traces over OTLP, configured by the standard
// OpenTelemetry environment variables and by nothing of adi's own, so a
// deployment points it at a collector the way it would any other service.
package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// The exporter reads the endpoint itself. They are named here only to tell
// whether there is one: the exporter falls back to localhost, and a server
// nobody pointed at a collector must not go looking for one.
const (
	endpointEnv       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	tracesEndpointEnv = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
)

const (
	protocolEnv       = "OTEL_EXPORTER_OTLP_PROTOCOL"
	tracesProtocolEnv = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
	// The protocol the OTLP specification makes the default.
	httpProtobuf = "http/protobuf"
)

// NewTracerProvider returns a provider that exports spans to the OTLP endpoint
// the environment names, and nil when it names none. version is the adi build,
// reported as the service's version. The spans are batched: Shutdown sends the
// ones still queued.
func NewTracerProvider(ctx context.Context, version string) (*sdktrace.TracerProvider, error) {
	if os.Getenv(endpointEnv) == "" && os.Getenv(tracesEndpointEnv) == "" {
		return nil, nil
	}
	// Exporting over the one protocol adi has to an endpoint set up for
	// another would lose every span without a word at startup.
	if p := protocol(); p != httpProtobuf {
		return nil, fmt.Errorf("OTLP protocol %q is not supported: adi exports traces over %s", p, httpProtobuf)
	}
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create the OTLP trace exporter: %w", err)
	}
	// The default resource takes the service name from OTEL_SERVICE_NAME. It is
	// merged last so that the environment has the final say on the version too.
	res, err := resource.Merge(resource.NewSchemaless(semconv.ServiceVersion(version)), resource.Default())
	if err != nil {
		return nil, fmt.Errorf("describe the service: %w", err)
	}
	return sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res)), nil
}

// protocol returns the OTLP protocol the environment asks for, the traces
// variable winning over the general one as the specification orders them.
func protocol() string {
	for _, env := range []string{tracesProtocolEnv, protocolEnv} {
		if p := os.Getenv(env); p != "" {
			return p
		}
	}
	return httpProtobuf
}
