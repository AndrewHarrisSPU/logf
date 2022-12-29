package testlog

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/exp/slog"
)

// Substrings returns a [slog.Handler] and a "want" function.
//
// When a logging call is made using the handler, log lines are written to a buffer.
// Calling "want" tests whether the buffer contains the given string.
// If it does not, t.Errorf is called.
// Calling want clears the buffer.
//
// The handler encodes to JSON, and adds source/line information.
func Substrings(t *testing.T) (h slog.Handler, want func(string)) {
	var b bytes.Buffer

	want = func(wantString string) {
		t.Helper()
		if !strings.Contains(b.String(), wantString) {
			t.Errorf("\n\texpected %s\n\tin %s", wantString, b.String())
		}
		b.Reset()
	}

	h = slog.HandlerOptions{
		ReplaceAttr: noTime,
		AddSource:   true,
	}.NewJSONHandler(&b)

	return h, want
}

func noTime(scope []string, a slog.Attr) slog.Attr {
	if a.Key == "time" {
		a.Key = ""
	}
	return a
}
