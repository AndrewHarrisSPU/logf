package logf

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/exp/slog"
)

const testTTYoutput = `   INFO    ok
   INFO    ok	l1:{}
   INFO    ok	l1:{a1:1}
   WARN    ok	l2:{a2:2}
   DEBUG   ok	l1:{}
   INFO    ok	l1:{a1:1}
   WARN    ok	l2:{a2:2}
   ERROR   ok	l3:{a3:3}
`

func TestTTY(t *testing.T) {
	var buf bytes.Buffer

	var ref slog.LevelVar
	ref.Set(DEBUG)

	log := func() Logger {
		return New().
			Writer(&buf).
			Ref(&ref).
			ShowLayout("level", "tags", "message", "\t", "attrs").
			ShowColor(false).
			ShowLevel(LevelText).
			ForceTTY(true).
			Logger()
	}

	log().Info("ok")

	log1 := log().WithGroup("l1")
	log1.Info("ok")

	log2 := log1.With("a1", 1)
	log2.Info("ok")

	log3 := log().WithGroup("l2").With("a2", 2)
	log3.Warn("ok")

	log4 := log().WithGroup("l3").With("a3", 3)

	log1.Debug("ok")
	log2.Info("ok")
	log3.Warn("ok")
	log4.Error("ok", nil)

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

	return GroupValue(as...)
}

const testTTYLogValuerOutput = ` ▏ value1nested
`

func TestTTYLogValuer(t *testing.T) {
	var buf bytes.Buffer

	log := New().
		Writer(&buf).
		ShowLayout("level", "tags", "message").
		ShowColor(false).
		ForceTTY(true).
		Logger()

	lm := logmap{
		"key1": KV("", "value1").Value,
		"key2": logmap{
			"key1nested": KV("", "value1nested").Value,
		}.LogValue(),
	}

	log.Infof("{lm.key2.key1nested}", "lm", lm.LogValue())

	if buf.String() != testTTYLogValuerOutput {
		t.Log(buf.String())
		t.Log(testTTYLogValuerOutput)
		t.Error("TTY output")
	}
}

func TestTTYReplace(t *testing.T) {
	var b bytes.Buffer

	want := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
		b.Reset()
	}

	log := New().
		ReplaceFunc(func(scope []string, a Attr) Attr {
			if a.Key == "secret" {
				if a.Value.Kind() != slog.GroupKind {
					a.Value = slog.StringValue("redacted")
				}
			}
			return a
		}).
		Writer(&b).
		ShowLayout("message", "\t", "attrs").
		ShowColor(false).
		ForceTTY(true).
		Logger()

	log = log.With("secret", 1)

	log.Infof("{secret}", "secret", 2)
	want(`redacted	secret:redacted secret:redacted`)

	log.Infof("{group.secret}, {group.group2.secret}", Group("group", Attrs(
		KV("secret", 3),
		Group("group2", Attrs(
			KV("secret", 4),
			KV("secret", 5),
		)...),
	)...))
	want(`redacted, redacted	secret:redacted group:{secret:redacted group2:{secret:redacted secret:redacted}}`)
}

const testTTYAuxOutput = `{"level":"INFO","msg":"buffer: auto"}
 ▏ buffer: forced TTY
{"level":"INFO","msg":"buffer: forced auxilliary"}
{"level":"INFO","msg":"buffer: forced TTY and auxilliary"}
 ▏ buffer: forced TTY and auxilliary
`

func TestTTYAux(t *testing.T) {
	var b bytes.Buffer

	auto := func(cfg *Config) Logger {
		return cfg.
			ForceTTY(false).
			ForceAux(false).
			Logger()
	}

	forceTTY := func(cfg *Config) Logger {
		return cfg.
			ForceTTY(true).
			ForceAux(false).
			Logger()
	}

	forceAux := func(cfg *Config) Logger {
		return cfg.
			ForceTTY(false).
			ForceAux(true).
			Logger()
	}

	forceBoth := func(cfg *Config) Logger {
		return cfg.
			ForceTTY(true).
			ForceAux(true).
			Logger()
	}

	cfg := New().
		ShowLayout("level", "message", "attrs").
		ShowColor(false).
		Writer(&b).
		ReplaceFunc(func(scope []string, a Attr) Attr {
			if a.Key == "time" {
				a.Key = ""
			}
			return a
		})

	auto(cfg).Info("buffer: auto")
	forceTTY(cfg).Info("buffer: forced TTY")
	forceAux(cfg).Info("buffer: forced auxilliary")
	forceBoth(cfg).Info("buffer: forced TTY and auxilliary")

	want, got := testTTYAuxOutput, b.String()
	if want != got {
		t.Log(want)
		t.Log(got)
		t.Error("TTY aux")
	}
}
