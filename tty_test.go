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

	log := func() Logger {
		return New().
			Writer(&buf).
			Ref(DEBUG).
			Layout("level", "tags", "message", "\t", "attrs").
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

type logmap map[string]Value

func (lm logmap) LogValue() Value {
	var as []Attr
	for k, v := range lm {
		as = append(as, KV(k, v))
	}

	return GroupValue(as)
}

const testTTYLogValuerOutput = `▕▎ value1nested
`

func TestLogValuer(t *testing.T) {
	var buf bytes.Buffer

	log := New().
		Writer(&buf).
		Layout("level", "tags", "message").
		Colors(false).
		ForceTTY().
		Logger()

	lm := logmap{
		"key1": KV( "", "value1").Value,
		"key2": logmap{
			"key1nested": KV("", "value1nested").Value,
		}.LogValue(),
	}

	log.Msgf("{key1nested}", lm )

	if buf.String() != testTTYLogValuerOutput {
		t.Log(buf.String())
		t.Log(testTTYLogValuerOutput)
		t.Error("TTY output")
	}
}