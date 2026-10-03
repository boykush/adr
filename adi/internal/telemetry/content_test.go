package telemetry

import (
	"testing"

	"github.com/caarlos0/env/v11"
)

// The variable, spelled as a deployment sets it rather than taken from
// contentEnv, so that the tests hold the declaration to the name.
const captureContentEnv = "OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT"

// TestContentIsCapturedOnlyOnTrue reads the opt-in the way OpenTelemetry reads
// a Boolean. A map stands in for the environment, so the one running the tests
// is neither read nor changed.
func TestContentIsCapturedOnlyOnTrue(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{name: "unset", env: map[string]string{}},
		{name: "empty", env: map[string]string{captureContentEnv: ""}},
		{name: "true", env: map[string]string{captureContentEnv: "true"}, want: true},
		{name: "any case", env: map[string]string{captureContentEnv: "TRUE"}, want: true},
		{name: "false", env: map[string]string{captureContentEnv: "false"}},
		// What strconv.ParseBool takes for true and OpenTelemetry does not.
		{name: "not a boolean", env: map[string]string{captureContentEnv: "1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := capturesContent(env.Options{Environment: c.env}); got != c.want {
				t.Errorf("capturesContent(%v) = %t, want %t", c.env, got, c.want)
			}
		})
	}
	// An environment that cannot be read gives no opt-in. Nothing in the
	// declaration fails to parse, so the options are what fail it here: they
	// require a variable that is not there.
	if capturesContent(env.Options{Environment: map[string]string{}, RequiredIfNoDef: true}) {
		t.Error("capturesContent = true for an environment that could not be read")
	}
}

// TestContentOptInIsReadFromTheProcess is the one test to set the variable for
// real: it is the process's environment a deployment sets it in.
func TestContentOptInIsReadFromTheProcess(t *testing.T) {
	t.Setenv(captureContentEnv, "false")
	if CapturesContent() {
		t.Error("CapturesContent() = true with the variable set to false")
	}
	t.Setenv(captureContentEnv, "true")
	if !CapturesContent() {
		t.Error("CapturesContent() = false with the variable set to true")
	}
}
