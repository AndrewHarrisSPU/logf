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
	missingAttr         = "!missing-attr"
	missingArg          = "!missing-arg"
	missingKey          = "!missing-key"
	missingRightBracket = "!missing-right-bracket"
)

// the text type is just a byte buffer
type text []byte

// SCAN

// scan into unescaped left/right bracket pairs
// if a key is found, clip holds key:verb text.
func (t *text) scanKey(msg string) (tail string, clip []byte, found bool) {
	var lpos, rpos int

	if msg, lpos = t.escapeUntil(msg, '{'); lpos < 0 {
		return "", nil, false
	}

	if msg, rpos = t.escapeUntil(msg, '}'); rpos < 0 {
		*t = append((*t)[:lpos], missingRightBracket...)
		return "", nil, false
	}

	// split clip from text
	*t, clip = (*t)[:lpos], (*t)[lpos:]
	return msg, clip, true
}

// while escaping `\`, write message runes to text
// until sep is found, or msg is exahusted
func (t *text) escapeUntil(msg string, sep rune) (tail string, n int) {
	var esc bool
	for n, r := range msg {
		switch {
		case esc:
			esc = false
			// special case: preserve the `\` from escaping a colon
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

// unescape colons in key
// always writes len(key) or less bytes
func unescapeColon(key []byte) text {
	var esc bool
	var n int
	for _, c := range key {
		if c == '\\' && !esc {
			esc = true
			continue
		}
		esc = false

		// (invariantly, n is not larger than index)
		key[n] = c
		n++
	}

	// key, sans-escapes, is of length n
	return key[:n]
}

func splitVerb(clip []byte) (key, verb []byte) {
	n := bytes.LastIndexByte(clip, ':')

	// no colon found
	// -> no verb
	if n < 0 {
		key, verb = clip, nil
		return
	}

	// colon in 0-pos can't be escaped
	// -> no key
	if n == 0 {
		key, verb = nil, clip[1:]
		return
	}

	// last colon is escaped
	// -> no verb
	if clip[n-1] == '\\' {
		key, verb = unescapeColon(clip), nil
		return
	}

	// colon found at n
	// -> key up to n, verb after n
	key, verb = unescapeColon(clip[:n]), clip[n+1:]
	return
}

// APPEND

func (t *text) Write(p []byte) (int, error) {
	*t = append(*t, p...)
	return len(*t), nil
}

func (t *text) appendByte(c byte) {
	*t = append(*t, c)
}

func (t *text) appendRune(r rune) {
	*t = utf8.AppendRune(*t, r)
}

func (t *text) appendString(s string) {
	*t = append(*t, s...)
}

func (t *text) appendArg(arg any, verb text) {
	v := slog.AnyValue(arg)
	t.appendValue(v, verb)
}

func (t *text) appendValue(v slog.Value, verb text) {
	if len(verb) > 0 {
		t.appendValueVerb(v, string(verb))
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
	case slog.GroupKind:
		// TODO: no fmt'ing?
		t.appendGroup(v.Group())

	case slog.LogValuerKind:
		t.appendValueVerb(v.Resolve(), verb)
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

	case slog.GroupKind:
		t.appendGroup(v.Group())

	case slog.LogValuerKind:
		t.appendValueNoVerb(v.Resolve())

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

func (t *text) appendGroup(as []Attr) {
	next := byte('[')
	for _, a := range as {
		t.appendByte(next)
		t.appendString(a.Key)
		t.appendByte('=')
		t.appendValueNoVerb(a.Value)
		next = ' '
	}
	t.appendByte(']')
}
