package logf

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// scan occurs before interpolation and before joining (seg, ctx, args)
//   - prematches for keyed interpolation are detected and added to the dictionary
//   - partitions args that are and are not consumed by unkeyed interpolation
func (s *splicer) scan(msg string, args []any) []any {
	var clip string
	var found bool
	var unkeyed int
	for {
		msg, clip, found = scanUntilKey(msg)
		if !found {
			break
		}

		key := s.scanSplitKey(clip)
		if len(key) > 0 {
			key = s.scanUnescape(key)
			s.dict[key] = missingAttrValue
		} else {
			unkeyed++
		}
	}

	for i := 0; i < unkeyed; i++ {
		if len(args) == 0 {
			s.list = append(s.list, missingArg)
			continue
		}
		s.list = append(s.list, args[0])
		args = args[1:]
	}

	return args
}

func scanUntilKey(msg string) (tail, clip string, found bool) {
	var lpos, rpos int

	if tail, lpos = scanUntil(msg, '{'); lpos < 0 {
		return "", "", false
	}
	lpos++

	if tail, rpos = scanUntil(tail, '}'); rpos < 0 {
		return "", "", false
	}
	rpos++

	tail = msg[lpos+rpos:]
	clip = msg[lpos : lpos+rpos-1]
	found = true
	return
}

func scanUntil(msg string, sep rune) (tail string, n int) {
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
func (s *splicer) scanSplitKey(clip string) (key string) {
	n := bytes.LastIndexByte([]byte(clip), ':')

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
		return s.scanUnescape(clip)
	}

	// last colon unescaped
	// -> clip up to n is key
	return s.scanUnescape(clip[:n])
}

// TODO: micro-optimizing allocs etc. here could be possible.
// putting it off for now.
func (s *splicer) scanUnescape(key string) (ukey string) {
	// TODO: is this worth it?
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

	// TODO: this should be safely unsafe

	// u := s.scratch[lpos:rpos]
	// uHeader := (*reflect.SliceHeader)(unsafe.Pointer(&u))
	// uKeyHeader := (*reflect.StringHeader)(unsafe.Pointer(&ukey))
	// uKeyHeader.Data, uKeyHeader.Len = uHeader.Data, uHeader.Len
	// return
}
