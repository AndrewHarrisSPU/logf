package logf

import (
	"bytes"
	"testing"
)

const testTTYoutput = ` INFO    ok  
 INFO    l1  ok  
 INFO    l1  ok  l1.a1=1
 WARN    l2  ok  l2.a2=2
 DEBUG   ok  
 INFO    l1  ok  l1.a1=1
 WARN    l2  ok  l2.a2=2
 ERROR   l3  ok  l3.a3=3
`

func TestTTY(t *testing.T) {
	var buf bytes.Buffer

	log := New().
		Writer(&buf).
		Level(DEBUG).
		Layout("level", "label", "message", "attrs").
		Colors(false).
		Logger()

	log.Msg("ok")

	log1 := log.Label("l1")
	log1.Msg("ok")

	log1 = log1.With("a1", 1)
	log1.Msg("ok")

	log2 := log.Level(WARN).Label("l2").With("a2", 2)
	log2.Msg("ok")

	log3 := log.Level(ERROR).Label("l3").With("a3", 3)

	log.Level(DEBUG).Msg("ok")
	log1.Msg("ok")
	log2.Msg("ok")
	log3.Msg("ok")

	if buf.String() != testTTYoutput {
		t.Log(buf.String())
		t.Log(testTTYoutput)
		t.Error("TTY output")
	}
}
