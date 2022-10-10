package logf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
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

	*t, clip = (*t)[:lpos], string((*t)[lpos:])
	return msg, clip, true
}

func (t *text) escapeUntil(msg string, sep rune) (tail string, n int) {
	var esc bool
	for n, r := range msg {
		switch {
		case esc:
			esc = false
			if r == ':' {
				t.appendString(`\:`)
				continue
			}
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
	n := bytes.LastIndexByte([]byte(clip), ':')

	// no colon found
	if n < 0 {
		key, verb = clip, ""
		return
	}

	// colon in 0-pos can't be escaped.
	// interpret entire clip as emtpy key, colon as formatting token, rest of clip as verb
	if n == 0 {
		key, verb = "", clip[1:]
		return
	}

	// if colon is found, but prior rune is '\'
	// interpret entire clip as key string
	if clip[n-1] == '\\' {
		key, verb = colonUnescape(clip), ""
		return
	}

	// unescaped colon means colon is formatting token
	// clip before n is key, clip after n is verb
	key, verb = colonUnescape(clip[:n]), clip[n+1:]
	return
}

func colonUnescape(s string) string {
	return strings.ReplaceAll(s, `\:`, `:`)
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
