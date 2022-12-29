package logf

import (
	"time"

	"golang.org/x/exp/slog"
)

type (
	// See [slog.Attr].
	Attr = slog.Attr

	// See [slog.Value].
	Value = slog.Value

	// See [slog.Level]
	Level = slog.Level
)

const (
	DEBUG = slog.DebugLevel
	INFO  = slog.InfoLevel
	WARN  = slog.WarnLevel
	ERROR = slog.ErrorLevel
)

// Below is copy-pasta from Go library code.

/*
FROM: golang.org/x/exp/slog

Copyright 2022 The Go Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.
*/

func concat[T any](head, tail []T) (ts []T) {
	ts = make([]T, len(head)+len(tail))
	copy(ts, head)
	copy(ts[len(head):], tail)
	return
}

func concatOne[T any](head []T, tail T) (ts []T) {
	ts = make([]T, len(head), len(head)+1)
	copy(ts, head)
	ts = append(ts, tail)
	return
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
// Copied from log/log.go.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

// This takes half the time of Time.AppendFormat.
func appendTimeRFC3339Millis(buf []byte, t time.Time) []byte {
	// TODO: try to speed up by indexing the buffer.
	char := func(b byte) {
		buf = append(buf, b)
	}

	year, month, day := t.Date()
	itoa(&buf, year, 4)
	char('-')
	itoa(&buf, int(month), 2)
	char('-')
	itoa(&buf, day, 2)
	char('T')
	hour, min, sec := t.Clock()
	itoa(&buf, hour, 2)
	char(':')
	itoa(&buf, min, 2)
	char(':')
	itoa(&buf, sec, 2)
	ns := t.Nanosecond()
	char('.')
	itoa(&buf, ns/1e6, 3)
	_, offsetSeconds := t.Zone()
	if offsetSeconds == 0 {
		char('Z')
	} else {
		offsetMinutes := offsetSeconds / 60
		if offsetMinutes < 0 {
			char('-')
			offsetMinutes = -offsetMinutes
		} else {
			char('+')
		}
		itoa(&buf, offsetMinutes/60, 2)
		char(':')
		itoa(&buf, offsetMinutes%60, 2)
	}
	return buf
}

/*
FROM: time.Duration.String, modified for appending into buf

Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// fmtFrac formats the fraction of v/10**prec (e.g., ".12345") into the
// tail of buf, omitting trailing zeros. It omits the decimal
// point too when the fraction is 0. It returns the index where the
// output bytes begin and the value v/10**prec.
func fmtFrac(buf []byte, v uint64, prec int) (nw int, nv uint64) {
	// Omit trailing zeros up to and including decimal point.
	w := len(buf)
	print := false
	for i := 0; i < prec; i++ {
		digit := v % 10
		print = print || digit != 0
		if print {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if print {
		w--
		buf[w] = '.'
	}
	return w, v
}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

func appendDuration(buf []byte, d time.Duration) []byte {
	lpos := len(buf)

	for i := 0; i < 32; i++ {
		buf = append(buf, 0x00)
	}

	w := len(buf)

	u := uint64(d)
	neg := d < 0
	if neg {
		u = -u
	}

	if u < uint64(time.Second) {
		// use smaller units
		var prec int
		w--
		buf[w] = 's'
		w--
		switch {
		case u == 0:
			buf[lpos], buf[lpos+1] = '0', 's'
			return buf[:lpos+2]
		case u < uint64(time.Microsecond):
			prec = 0
			buf[w] = 'n'
		case u < uint64(time.Millisecond):
			prec = 3
			w--
			copy(buf[w:], "µ")
		default:
			prec = 6
			buf[w] = 'm'
		}
		w, u = fmtFrac(buf[:w], u, prec)
		w = fmtInt(buf[:w], u)
	} else {
		w--
		buf[w] = 's'

		w, u = fmtFrac(buf[:w], u, 9)

		w = fmtInt(buf[:w], u%60)
		u /= 60

		if u > 0 {
			w--
			buf[w] = 'm'
			w = fmtInt(buf[:w], u%60)
			u /= 60

			if u > 0 {
				w--
				buf[w] = 'h'
				w = fmtInt(buf[:w], u)
			}
		}
	}

	if neg {
		w--
		buf[w] = '-'
	}

	width := 32 - (w - lpos)
	gap := lpos + (32 - width)
	for i := 0; i < width; i++ {
		buf[lpos+i] = buf[gap+i]
	}

	return buf[:lpos+width]
}
