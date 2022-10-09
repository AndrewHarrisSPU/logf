package logf

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"

	"golang.org/x/exp/slog"
)

const (
	corruptKind         = "!corrupt-kind"
	missingArg          = "!missing-arg"
	missingKey          = "!missing-key"
	missingRightBracket = "!missing-right-bracket"
)

// the text type is just a byte buffer
type text []byte

// SCAN

func (t *text) scanKey(msg string) (tail string, clip string, ok bool) {
	var lpos, rpos int
	if msg, lpos = t.escapeUntil(msg, '{'); lpos < 0 {
		return "", "", false
	}

	if msg, rpos = t.escapeUntil(msg, '}'); rpos < 0 {
		*t = append((*t)[:lpos], missingRightBracket...)
		return "", "", false
	}

	*t, clip = (*t)[:lpos], string( (*t)[lpos:])
	return msg, clip, true
}

func (t *text) escapeUntil(msg string, sep rune) (tail string, n int) {
	var esc bool
	for n, r := range msg {
		switch {
		case esc:
			esc = false
			fallthrough
		default:
			t.appendRune(r)
		case r == '\\':
			esc = true
		case r == sep:
			return msg[n+1:], len(*t)
		}
	}
	return "", -1
}

func splitVerb(clip string) (key, verb string) {
	if n := bytes.IndexByte( []byte(clip), ':'); n >= 0 {
		key, verb = clip[:n], clip[n+1:]
		key = keyEscape(key)
		return
	}
	return keyEscape(clip), ""
}

// APPEND

func (t *text) Write(p []byte) (int, error) {
	*t = append(*t, p...)
	return len(*t), nil
}

func (t *text) appendRune(r rune) {
	*t = utf8.AppendRune(*t, r)
}

func (t *text) appendString(s string) {
	*t = append(*t, s...)
}

func (t *text) appendArg(arg any, verb string) {
	v := slog.AnyValue(arg)
	t.appendValue(v, verb)
}

func (t *text) appendValue(v slog.Value, verb string) {
	if len(verb) > 0 {
		t.appendValueVerb(v, verb)
	} else {
		t.appendValueNoVerb(v)
	}
}

func (t *text) appendValueVerb(v slog.Value, verb string) {
	switch v.Kind() {
	case slog.StringKind:
		fmt.Fprintf(t, verb, v.String())
	case slog.BoolKind:
		fmt.Fprintf(t, verb, v.Bool())
	case slog.Float64Kind:
		fmt.Fprintf(t, verb, v.Float64())
	case slog.Int64Kind:
		fmt.Fprintf(t, verb, v.Int64())
	case slog.Uint64Kind:
		fmt.Fprintf(t, verb, v.Uint64())
	case slog.DurationKind:
		fmt.Fprintf(t, verb, v.String())
	case slog.TimeKind:
		*t = v.Time().AppendFormat(*t, verb)
	case slog.AnyKind:
		fmt.Fprintf(t, verb, v.Any())
	default:
		panic(corruptKind)
	}
}

func (t *text) appendValueNoVerb(v slog.Value) {
	switch v.Kind() {
	case slog.StringKind:
		t.appendString(v.String())

	case slog.BoolKind:
		*t = strconv.AppendBool(*t, v.Bool())
	case slog.Float64Kind:
		*t = strconv.AppendFloat(*t, v.Float64(), 'g', -1, 64)
	case slog.Int64Kind:
		*t = strconv.AppendInt(*t, v.Int64(), 10)
	case slog.Uint64Kind:
		*t = strconv.AppendUint(*t, v.Uint64(), 10)

	case slog.DurationKind:
		*t = appendDuration(*t, v.Duration())
	case slog.TimeKind:
		*t = appendTimeRFC3339Millis(*t, v.Time())

	case slog.AnyKind:
		fmt.Fprintf(t, "%v", v.Any())

	default:
		panic(corruptKind)
	}
}

func (t *text) appendError(err error) {
	if len(*t) > 0 {
		t.appendString(": ")
	}
	t.appendString(err.Error())
}
