package telemetry

import (
	"strings"

	"github.com/caarlos0/env/v11"
)

// contentEnv is the opt-in to spans that keep what a caller sent. The GenAI
// semantic conventions leave such content out by default, and name this
// variable as the example of how to ask for it.
type contentEnv struct {
	Capture boolean `env:"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT"`
}

// boolean is a Boolean as OpenTelemetry reads one from the environment: true
// for "true" in any case, false for anything else. A bool field would go
// through strconv.ParseBool, which takes "1" for true and fails on "yes".
type boolean bool

func (b *boolean) UnmarshalText(text []byte) error {
	*b = boolean(strings.EqualFold(string(text), "true"))
	return nil
}

// CapturesContent reports whether the environment opts in to spans that keep
// what a caller sent.
func CapturesContent() bool {
	return capturesContent(env.Options{})
}

// capturesContent reads contentEnv from the environment opts names, the
// process's unless it names another. Split out so tests can hand it a map and
// leave the process's environment alone.
func capturesContent(opts env.Options) bool {
	content, err := env.ParseAsWithOptions[contentEnv](opts)
	// An opt-in that cannot be read has not been given.
	return err == nil && bool(content.Capture)
}
