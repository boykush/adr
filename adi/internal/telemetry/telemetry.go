// Package telemetry exports adi's traces over OTLP, configured by the standard
// OpenTelemetry environment variables and by nothing of adi's own, so a
// deployment points it at a collector the way it would any other service.
package telemetry

import (
	"context"
	"fmt"

	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// otlpEnv is what adi reads of the standard variables for itself. The others
// the exporter and the resource read on their own.
type otlpEnv struct {
	// The exporter reads the endpoint itself. They are named here only to tell
	// whether there is one: the exporter falls back to localhost, and a server
	// nobody pointed at a collector must not go looking for one.
	Endpoint       string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	TracesEndpoint string `env:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"`

	// The default is the protocol the OTLP specification makes the default.
	Protocol       string `env:"OTEL_EXPORTER_OTLP_PROTOCOL" envDefault:"http/protobuf"`
	TracesProtocol string `env:"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"`
}

// httpProtobuf is the protocol of the exporter adi is built with.
const httpProtobuf = "http/protobuf"

// NewTracerProvider returns a provider that exports spans to the OTLP endpoint
// the environment names, and nil when it names none. version is the adi build,
// reported as the service's version. The spans are batched: Shutdown sends the
// ones still queued.
func NewTracerProvider(ctx context.Context, version string) (*sdktrace.TracerProvider, error) {
	return newTracerProvider(ctx, version, env.Options{})
}

// newTracerProvider reads otlpEnv from the environment opts names, the
// process's unless it names another. Split out so tests can hand it a map and
// leave the process's environment alone. Whatever opts names, the exporter and
// the resource read the process's.
func newTracerProvider(ctx context.Context, version string, opts env.Options) (*sdktrace.TracerProvider, error) {
	otlp, err := env.ParseAsWithOptions[otlpEnv](opts)
	if err != nil {
		return nil, fmt.Errorf("read the environment: %w", err)
	}
	if otlp.Endpoint == "" && otlp.TracesEndpoint == "" {
		return nil, nil
	}
	// Exporting over the one protocol adi has to an endpoint set up for
	// another would lose every span without a word at startup.
	if p := otlp.protocol(); p != httpProtobuf {
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
func (e otlpEnv) protocol() string {
	if e.TracesProtocol != "" {
		return e.TracesProtocol
	}
	return e.Protocol
}
