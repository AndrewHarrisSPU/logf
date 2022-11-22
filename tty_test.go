package logf

import (
	"bytes"
	"testing"
)

const testTTYoutput = `   INFO    test ok
   INFO    test ok
   INFO    test ok  l1.a1=1
   WARN    test ok  l2.a2=2
   DEBUG   test ok
   INFO    test ok  l1.a1=1
   WARN    test ok  l2.a2=2
   ERROR   test ok  l3.a3=3
`

func TestTTY(t *testing.T) {
	var buf bytes.Buffer

	log := func() *Logger {
		return New().
			Writer(&buf).
			Ref(DEBUG).
			Layout("level", "label", "message", "attrs").
			Colors(false).
			ForceTTY().
			Logger().
			Label("test")
	}

	log().Msg("ok")

	log1 := log().Group("l1")
	log1.Msg("ok")

	log1.With("a1", 1)
	log1.Msg("ok")

	log2 := log().Level(WARN).Group("l2").With("a2", 2)
	log2.Msg("ok")

	log3 := log().Level(ERROR).Group("l3").With("a3", 3)

	log().Level(DEBUG).Msg("ok")
	log1.Msg("ok")
	log2.Msg("ok")
	log3.Msg("ok")

	if buf.String() != testTTYoutput {
		t.Log(buf.String())
		t.Log(testTTYoutput)
		t.Error("TTY output")
	}
}
