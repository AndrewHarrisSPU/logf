package logf

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// SCAN

// scan occurs before interpolation and before joining
//   - prematches for keyed interpolation are detected and added to the dictionary
//   - returns args not consumed by unkeyed interpolations
func (s *splicer) scan(msg string, args []any) []any {
	var clip string
	var found bool
	var nUnkeyed int
	s.interpolates = false

	for {
		msg, clip, found = scanNext(msg)
		if !found {
			break
		}
		s.interpolates = true

		key := s.scanClip(clip)
		if len(key) > 0 {
			key = s.scanUnescapeKey(key)
			s.dict[key] = missingAttrValue
		} else {
			nUnkeyed++
		}
	}

	args = s.listUnkeyed(nUnkeyed, args)
	return args
}

func scanNext(msg string) (tail, clip string, found bool) {
	var lpos, rpos int

	if tail, lpos = scanUntilRune(msg, '{'); lpos < 0 {
		return "", "", false
	}
	lpos++

	if tail, rpos = scanUntilRune(tail, '}'); rpos < 0 {
		return "", "", false
	}
	rpos++

	tail, clip = msg[lpos+rpos:], msg[lpos:lpos+rpos-1]
	found = true
	return
}

func scanUntilRune(msg string, sep rune) (tail string, n int) {
	var esc bool
	for n, r := range msg {
		switch {
		case esc:
			esc = false
			fallthrough
		default:
		case r == '\\':
			esc = true
		case r == sep:
			return msg[n+1:], n
		}
	}
	return "", -1
}

// count unkeyed
func (s *splicer) scanClip(clip string) (key string) {
	n := strings.LastIndexByte(clip, ':')

	// no colon, no verb
	if n < 0 {
		// the unique string that is unkeyed with no verb -> unkeyed
		if clip == "{}" {
			return ""
		}
		// otherwise -> keyed
		return clip
	}

	// colon in 0-pos can't be escaped
	// -> unkeyed
	if n == 0 {
		return ""
	}

	// last colon escaped
	// -> clip is key
	if clip[n-1] == '\\' {
		return s.scanUnescapeKey(clip)
	}

	// last colon unescaped
	// -> clip up to n is key
	return s.scanUnescapeKey(clip[:n])
}

// TODO: micro-optimizing allocs etc. here could be possible.
// putting it off for now.
func (s *splicer) scanUnescapeKey(key string) string {
	if !strings.ContainsRune(key, '\\') {
		return key
	}

	lpos := len(s.scratch)
	var esc bool
	for _, r := range key {
		if r == '\\' && !esc {
			esc = true
			continue
		}
		esc = false
		s.scratch = utf8.AppendRune(s.scratch, r)
	}
	rpos := len(s.scratch)

	return string(s.scratch[lpos:rpos])
}

// INTERPOLATE

func (s *splicer) ipol(msg string) bool {
	if !s.interpolates {
		// s.writeString(msg)
		// s.writeUnescape(msg)
		return false
	}

	var clip []byte
	var found bool
	for {
		if msg, clip, found = s.ipolNext(msg); !found {
			break
		}
		s.ipolAttr(clip)
	}
	return true
}

// scan into unescaped left/right bracket pairs
// if a key is found, clip holds key:verb text.
func (s *splicer) ipolNext(msg string) (tail string, clip []byte, found bool) {
	var lpos, rpos int

	if msg, lpos = s.ipolUntilRune(msg, '{'); lpos < 0 {
		return "", nil, false
	}

	if msg, rpos = s.ipolUntilRune(msg, '}'); rpos < 0 {
		return "", nil, false
	}

	// split clip from text
	s.text, clip = s.text[:lpos], s.text[lpos:]
	return msg, clip, true
}

// while escaping `\`, write message runes to text
// until sep is found, or msg is exahusted
func (s *splicer) ipolUntilRune(msg string, sep rune) (tail string, n int) {
	var esc bool
	for n, r := range msg {
		switch {
		case esc:
			esc = false
			// special case: preserve the `\` from escaping a colon
			if r == ':' {
				s.WriteString(`\:`)
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

// alternative to ipolUntilRune
// drops escaping, but doesn't interpolate
func (s *splicer) writeUnescape(msg string) {
	var esc bool
	for _, r := range msg {
		switch {
		case esc:
			esc = false
			fallthrough
		default:
			s.writeRune(r)
		case r == '\\':
			esc = true
		}
	}
	return
}

func (s *splicer) ipolAttr(clip []byte) {
	key, verb := ipolClip(clip)

	if len(key) == 0 {
		s.ipolUnkeyed(verb)
	} else {
		s.ipolKeyed(key, verb)
	}
}

func (s *splicer) ipolUnkeyed(verb []byte) {
	var arg any
	if len(s.list) > 0 {
		arg = s.list[0]
		s.list = s.list[1:]
	} else {
		s.WriteString(missingArg)
		return
	}

	s.writeArg(arg, verb)
}

func (s *splicer) ipolKeyed(key, verb []byte) {
	v, ok := s.dict[string(key)]

	// should be unreachable, but I kept reaching it
	if !ok {
		s.WriteString(missingAttr)
		return
	}

	s.WriteValue(v, verb)
}

func ipolClip(clip []byte) (key, verb []byte) {
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
		key, verb = ipolUnescapeKey(clip), nil
		return
	}

	// colon found at n
	// -> key up to n, verb after n
	key, verb = ipolUnescapeKey(clip[:n]), clip[n+1:]
	return
}

func ipolUnescapeKey(key []byte) []byte {
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
