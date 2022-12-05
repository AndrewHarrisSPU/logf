package logf

import (
	"bytes"
	"testing"
)

const testTTYoutput = `   INFO    ok
   INFO    ok
   INFO    ok   l1:{a1:1}
   WARN    ok   l2:{a2:2}
   DEBUG   ok
   INFO    ok   l1:{a1:1}
   WARN    ok   l2:{a2:2}
   ERROR   ok   l3:{a3:3}
`

func TestTTY(t *testing.T) {
	var buf bytes.Buffer

	log := func() *Logger {
		return New().
			Writer(&buf).
			Ref(DEBUG).
			Layout("level", "label", "message", "\t", "attrs").
			Colors(false).
			Level(LevelText).
			ForceTTY().
			Logger()
	}

	log().Msg("ok")

	log1 := log().Group("l1")
	log1.Msg("ok")

	log2 := log1.With("a1", 1)
	log2.Msg("ok")

	log3 := log().Level(WARN).Group("l2").With("a2", 2)
	log3.Msg("ok")

	log4 := log().Level(ERROR).Group("l3").With("a3", 3)

	log1.Level(DEBUG).Msg("ok")
	log2.Msg("ok")
	log3.Msg("ok")
	log4.Msg("ok")

	if buf.String() != testTTYoutput {
		t.Log(buf.String())
		t.Log(testTTYoutput)
		t.Error("TTY output")
	}
}
