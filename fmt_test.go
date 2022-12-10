package logf

import (
	"testing"
)

func TestFormat(t *testing.T) {
	a := Fmt("Hello, {...}", "...", "world")

	if a != "Hello, world" {
		t.Errorf("want %s, got %s", "Hello, world", a)
	}
}
