package logf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/exp/slog"
)

// BYTE WRITES

func (s *splicer) Write(p []byte) (int, error) {
	s.text = append(s.text, p...)
	return len(p), nil
}

func (s *splicer) writeByte(c byte) {
	s.text = append(s.text, c)
}

func (s *splicer) writeRune(r rune) {
	s.text = utf8.AppendRune(s.text, r)
}

func (s *splicer) writeString(m string) {
	s.text = append(s.text, m...)
}

// INTERPOLATION WRITES

// scan into unescaped left/right bracket pairs
// if a key is found, clip holds key:verb text.

func (s *splicer) writeUntilKey(msg string) (tail string, clip []byte, found bool) {
	var lpos, rpos int

	if msg, lpos = s.writeUntil(msg, '{'); lpos < 0 {
		return "", nil, false
	}

	if msg, rpos = s.writeUntil(msg, '}'); rpos < 0 {
		s.writeString(missingRightBracket)
		return "", nil, false
	}

	// split clip from text
	s.text, clip = s.text[:lpos], s.text[lpos:]
	return msg, clip, true
}

// while escaping `\`, write message runes to text
// until sep is found, or msg is exahusted
func (s *splicer) writeUntil(msg string, sep rune) (tail string, n int) {
	var esc bool
	for n, r := range msg {
		switch {
		case esc:
			esc = false
			// special case: preserve the `\` from escaping a colon
			if r == ':' {
				s.writeString(`\:`)
				continue
			}
			fallthrough
		default:
			s.writeRune(r)
		case r == '\\':
			esc = true
		case r == sep:
			return msg[n+1:], len(s.text)
		}
	}
	return "", -1
}

// unescape colons in key
// always writes len(key) or less bytes
func unescapeColon(key []byte) []byte {
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

func splitKeyVerb(clip []byte) (key, verb []byte) {
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

// TYPED WRITES

func (s *splicer) writeArg(arg any, verb []byte) {
	v := slog.AnyValue(arg)
	s.writeValue(v, verb)
}

func (s *splicer) writeValue(v slog.Value, verb []byte) {
	if len(verb) > 0 {
		s.writeValueVerb(v, string(verb))
	} else {
		s.writeValueNoVerb(v)
	}
}

func (s *splicer) writeValueNoVerb(v slog.Value) {
	switch v.Kind() {
	case slog.StringKind:
		s.writeString(v.String())
	case slog.BoolKind:
		s.text = strconv.AppendBool(s.text, v.Bool())
	case slog.Float64Kind:
		s.text = strconv.AppendFloat(s.text, v.Float64(), 'g', -1, 64)
	case slog.Int64Kind:
		s.text = strconv.AppendInt(s.text, v.Int64(), 10)
	case slog.Uint64Kind:
		s.text = strconv.AppendUint(s.text, v.Uint64(), 10)
	case slog.DurationKind:
		s.text = appendDuration(s.text, v.Duration())
	case slog.TimeKind:
		s.text = appendTimeRFC3339Millis(s.text, v.Time())
	case slog.GroupKind:
		s.writeGroup(v.Group())
	case slog.LogValuerKind:
		s.writeValueNoVerb(v.Resolve())
	case slog.AnyKind:
		fmt.Fprintf(s, "%v", v.Any())
	default:
		panic(corruptKind)
	}
}

func (s *splicer) writeValueVerb(v slog.Value, verb string) {
	switch v.Kind() {
	case slog.StringKind:
		fmt.Fprintf(s, verb, v.String())
	case slog.BoolKind:
		fmt.Fprintf(s, verb, v.Bool())
	case slog.Float64Kind:
		fmt.Fprintf(s, verb, v.Float64())
	case slog.Int64Kind:
		fmt.Fprintf(s, verb, v.Int64())
	case slog.Uint64Kind:
		fmt.Fprintf(s, verb, v.Uint64())
	case slog.DurationKind:
		s.writeDurationVerb(v.Duration(), verb)
	case slog.TimeKind:
		s.writeTimeVerb(v.Time(), verb)
	case slog.GroupKind:
		s.writeGroup(v.Group())
	case slog.LogValuerKind:
		s.writeValueVerb(v.Resolve(), verb)
	case slog.AnyKind:
		fmt.Fprintf(s, verb, v.Any())
	default:
		panic(corruptKind)
	}
}

func (s *splicer) writeTimeVerb(t time.Time, verb string) {
	switch verb {
	case "epoch":
		s.text = strconv.AppendInt(s.text, t.Unix(), 10)
	case "RFC3339":
		s.text = t.AppendFormat(s.text, time.RFC3339)
	case "kitchen":
		s.text = t.AppendFormat(s.text, time.Kitchen)
	case "stamp":
		s.text = t.AppendFormat(s.text, time.Stamp)
	default:
		// TODO: might be slow /shrug
		s.text = t.AppendFormat(s.text, strings.Replace(verb, ";", ":", -1))
	}
}

func (s *splicer) writeDurationVerb(d time.Duration, verb string) {
	switch verb {
	case "fast":
		s.text = strconv.AppendInt(s.text, int64(d), 10)
	default:
		fmt.Fprintf(s, verb, d.String())
	}
}

func (s *splicer) writeError(err error) {
	if len(s.text) > 0 {
		s.writeString(": ")
	}
	s.writeString(err.Error())
}

func (s *splicer) writeGroup(as []Attr) {
	next := byte('[')
	for _, a := range as {
		s.writeByte(next)
		s.writeString(a.Key)
		s.writeByte('=')
		s.writeValueNoVerb(a.Value)
		next = ' '
	}
	s.writeByte(']')
}
