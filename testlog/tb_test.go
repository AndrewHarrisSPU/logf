package testlog

import (
	"testing"
)

func Test_Ok(t *testing.T) {
	tb := UsingTB(t)

	tb.Log("should appear")
	tb.Want("should appear")

	tb.Logf( "a number: %d", 42)
	tb.Want("a number: 42")

	// tb.Error("a test error")
}