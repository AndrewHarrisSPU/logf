package _demotb_test

import (
	"testing"

	"github.com/AndrewHarrisSPU/logf"
)

func Test_Ok(tT *testing.T) {
	t := logf.WithTest(tT)

	t.Log("should appear")
	t.Logf( "a number: %d", 42)
	t.Error("a test error")

	t.Error("should be fresh")
}
