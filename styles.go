package logf

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

// COLORS / STYLES

type pen string

func (p pen) use(b *Buffer){
	if len(p) > 0 {
		b.WriteString(string(p))
	}
	return
}

func (p pen) drop(b *Buffer){
	if len(p) > 0 {
		b.WriteString("\x1b[0m")
	}
	return
}

func newPen(s string) pen {
	var bg, fg byte
	var setBg bool
	var isDim, isBright bool
	var isItalic, isUnderline, isBlink bool

	tokens := strings.Fields(s)
	for _, token := range tokens {
		setColor := func(c byte) {
			if c == 0 {
				return
			}
			if setBg {
				bg = c
			} else {
				fg = c
			}
		}

		switch token {
		case "bg":
			setBg = true
		case "fg":
			setBg = false
		case "black":
			setColor('0')
		case "red":
			setColor('1')
		case "green":
			setColor('2')
		case "yellow":
			setColor('3')
		case "blue":
			setColor('4')
		case "magenta":
			setColor('5')
		case "cyan":
			setColor('6')
		case "white":
			setColor('7')
		case "bold", "bright":
			isBright, isDim = true, false
		case "dim", "dark":
			isBright, isDim = false, true
		case "italic":
			isItalic = true
		case "underline":
			isUnderline = true
		case "blink":
			isBlink = true
		}
	}

	var st []byte
	push := func(sub ...byte) {
		if len(st) == 0 {
			st = append(st, "\x1b["...)
		}
		st = append(st, sub...)
		st = append(st, ';')
	}

	// colors
	if fg != 0 {
		push('3', fg)
	}
	if bg != 0 {
		push('4', bg)
	}

	// effects
	if isBright {
		push('1')
	}
	if isDim {
		push('2')
	}
	if isItalic {
		push('3')
	}
	if isUnderline {
		push('4')
	}
	if isBlink {
		push('5')
	}

	// close
	if len(st) > 0 {
		st[len(st)-1] = 'm'
	}

	return pen(st)
}

func (tty *TTY) levelPen(level slog.Level) (p pen) {
	switch {
	case level < INFO:
		p = tty.fmtr.debugPen
	case level < WARN:
		p = tty.fmtr.infoPen
	case level < ERROR:
		p = tty.fmtr.warnPen
	default:
		p = tty.fmtr.errorPen
	}
	return
}

// CUSTOM ENCODERS

func init() {
	LevelBar = EncodeFunc(encLevelBar)
	LevelText = EncodeFunc(encLevelText)
	TimeShort = EncodeFunc(encTimeShort)
	SourceAbs = EncodeFunc(encSourceAbs)
	SourceShort = EncodeFunc(encSourceShort)
	SourcePkg = EncodeFunc(encSourcePkg)
}

var (
	LevelBar    Encoder[slog.Level]
	LevelText   Encoder[slog.Level]
	Message     Encoder[string]
	TimeShort   Encoder[time.Time]
	SourceAbs   Encoder[SourceLine]
	SourcePkg   Encoder[SourceLine]
	SourceShort Encoder[SourceLine]
)

func encGroupOpen(b *Buffer, _ any) {
	b.WriteString("{")
}

func encGroupClose(b *Buffer, count int){
	for i := 0; i < count; i++ {
		b.WriteByte('}')
	}
	return
}

func encKey(b *Buffer, key string){
	b.WriteString(key)
	b.WriteString(":")
}

func encValue(b *Buffer, v Value){
	b.WriteValue(v, nil)
}

func encTag(b *Buffer, a Attr){
	b.WriteValue(a.Value, nil)
}

func encLevelText(b *Buffer, level slog.Level){
	// compute padding
	width := len(level.String())

	pad := (12 - width) / 2
	pad1 := width % 2

	b.WriteString("      "[:pad+pad1-1])
	b.WriteString(level.String())
	b.WriteString("      "[:pad])
}

func encLevelBar(b *Buffer, level slog.Level){
	b.WriteString("┃")
}

func encTimeShort(b *Buffer, t time.Time){
	b.WriteString(t.Format("15:04:05"))
}

type SourceLine struct {
	File string
	Line int
}

func encSourcePkg(b *Buffer, src SourceLine){
	b.WriteString(filepath.Base(filepath.Dir(src.File)))
}

func encSourceShort(b *Buffer, src SourceLine){
	b.WriteString(filepath.Base(src.File))
	b.WriteString(":")
	b.WriteString(strconv.Itoa(src.Line))
}

func encSourceAbs(b *Buffer, src SourceLine){
	b.WriteString(src.File)
	b.WriteString(":")
	b.WriteString(strconv.Itoa(src.Line))
}