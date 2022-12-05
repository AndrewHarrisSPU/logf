package logf

import (
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// StdRef is a global [slog.LevelVar] used in default-ish configurations.
var StdRef slog.LevelVar

var stdMutex sync.Mutex

func writerIsTerminal(w io.Writer) bool {
	file, isFile := w.(*os.File)
	if !isFile {
		return false
	}

	stat, _ := file.Stat()
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

// CONFIG

// Config is a base type for `logf` handler and logger configuration.
//
// To construct a [Logger] with an already extant [slog.Handler], see [UsingHandler].
//
// # Typical usage
//
// 1. The [logf.New] function opens a new Config instance.
//
// 2. Next, zero or more Config methods are chained to set configuration fields.
//
// Methods applying to any handler or logger produced by the Config, and defaults:
//   - [Config.Writer]: os.Stdout
//   - [Config.Ref]: logf.StdRef
//   - [Config.AddSource]: false
//   - [Config.Level]: INFO
//   - [Config.ReplaceFunc]: nil
//
// Methods applying only to a [TTY], or a logger based on one, and default arguments:
//   - [Config.Layout]: "level", "time", "tags", "message", "\t", "attrs"
//   - [Config.AttrKey]: "dim cyan"
//   - [Config.AttrValue]: "cyan"
//   - [Config.ForceTTY]: false
//   - [Config.Group]: "dim"
//   - [Config.Level]: LevelBar
//   - [Config.LevelColors]: "bright cyan", "bright green", "bright yellow", "bright red"
//   - [Config.Message]: ""
//   - [Config.Source]: "dim", SourceAbs
//   - [Config.Tag]: "#", "bright magenta"
//   - [Config.TagEncoce]: nil
//   - [Config.Time]: "dim", TimeShort
//   - [Config.Colors]: true
//
// 3. A Config method returning a [Logger] or a [TTY] closes the chained invocation:
//   - [Config.TTY] returns a [TTY]
//   - [Config.Logger] returns a [Logger] based on a [TTY].
//   - [Config.Printer] returns a [Logger], based on a [TTY], with a preset layout.
//   - [Config.JSON] returns a [Logger] based on a [slog.JSONHandler]
//   - [Config.Text] returns a [Logger] based on a [slog.TextHandler]
//
type Config struct {
	// sink config
	w           io.Writer
	useStdMutex bool

	// slog.Handler config
	ref     slog.Leveler
	replace func(Attr) Attr

	// tty gadgets
	fmtr      ttyFormatter
	useColors bool
	forceTTY  bool
}

// New opens a Config with default values.
func New() *Config {
	cfg := &Config{
		w:           os.Stdout,
		ref:         &StdRef,
		replace:     nil,
		useColors:   true,
		useStdMutex: true,

		fmtr: ttyFormatter{
			// layout
			layout: []ttyField{
				ttyLevelField,
				ttyTimeField,
				ttyTagsField,
				ttyMessageField,
				ttyTabField,
				ttyAttrsField,
			},

			// field encodings
			time: ttyEncoder[time.Time]{
				"\x1b[2m",
				EncodeFunc(encTimeShort),
			},
			level: ttyEncoder[slog.Level]{
				"",
				EncodeFunc(encLevelBar),
			},
			message: ttyEncoder[string]{
				"",
				nil,
			},
			key: ttyEncoder[string]{
				"\x1b[36;2m",
				EncodeFunc(encKey),
			},
			value: ttyEncoder[Value]{
				"\x1b[36m",
				EncodeFunc(encValue),
			},
			source: ttyEncoder[SourceLine]{
				"\x1b[2m",
				EncodeFunc(encSourceAbs),
			},
			groupOpen:  EncodeFunc(encGroupOpen),
			groupClose: EncodeFunc(encGroupClose),

			// level colors
			groupPen: "\x1b[2m",
			debugPen: "\x1b[2m",
			infoPen:  "\x1b[32;1m",
			warnPen:  "\x1b[33;1m",
			errorPen: "\x1b[31;1m",

			// tags
			tag: map[string]ttyEncoder[Attr]{
				"#": ttyEncoder[Attr]{
					"\x1b[35;1m",
					EncodeFunc(encTag),
				},
			},
		},
	}

	return cfg
}

// Ref configures the use of the given reference [slog.Leveler].
func (cfg *Config) Ref(level slog.Leveler) *Config {
	cfg.ref = level
	return cfg
}

// Writer configures the eventual destination of log lines.
func (cfg *Config) Writer(w io.Writer) *Config {
	cfg.w = w
	cfg.useStdMutex = false
	return cfg
}

// Colors toggles [TTY] color encoding, using ANSI escape codes.
//
// TODO: support cygwin escape codes.
func (cfg *Config) Colors(toggle bool) *Config {
	cfg.useColors = toggle
	return cfg
}

// Time sets a color and an encoder for the [slog.Record.Time] field.
// If the enc argument is nil, the configuration uses the [TimeShort] function.
func (cfg *Config) Time(color string, enc Encoder[time.Time]) *Config {
	if enc == nil {
		enc = EncodeFunc(encTimeShort)
	}
	cfg.fmtr.time = ttyEncoder[time.Time]{newPen(color), enc}
	return cfg
}

// Level sets an encoder for the [slog.Record.Level] field.
// If the enc argument is nil, the configuration uses the [LevelBar] function.
func (cfg *Config) Level(enc Encoder[slog.Level]) *Config {
	if enc == nil {
		enc = EncodeFunc(encLevelBar)
	}
	cfg.fmtr.level = ttyEncoder[slog.Level]{newPen(""), enc}
	return cfg
}

// LevelColors configures four colors for DEBUG, INFO, WARN, and ERROR levels.
// These colors are used when a [slog.Record.Level] is encoded.
func (cfg *Config) LevelColors(debug string, info string, warn string, error string) *Config {
	cfg.fmtr.debugPen = newPen(debug)
	cfg.fmtr.infoPen = newPen(info)
	cfg.fmtr.warnPen = newPen(warn)
	cfg.fmtr.errorPen = newPen(error)
	return cfg
}

// Message sets a color for the [slog.Record.Message] field.
func (cfg *Config) Message(color string) *Config {
	cfg.fmtr.message = ttyEncoder[string]{newPen(color), nil}
	return cfg
}

// AttrKey sets a color and an encoder for [slog.Attr.Key] encoding.
// If the enc argument is nil, the configuration uses an [Encoder] that simply writes the [slog.Attr.Key].
// TODO: this default does no escaping. Perhaps JSON quoting and escaping would be useful.
func (cfg *Config) AttrKey(color string, enc Encoder[string]) *Config {
	if enc == nil {
		enc = EncodeFunc(encKey)
	}
	cfg.fmtr.key = ttyEncoder[string]{newPen(color), enc}
	return cfg
}

// AttrValue sets a color and an encoder for [slog.Attr.Value] encoding.
// If the enc argument is nil, the configuration uses an default [Encoder].
// TODO: this default does no escaping. Perhaps JSON quoting and escaping would be useful.
func (cfg *Config) AttrValue(color string, enc Encoder[Value]) *Config {
	if enc == nil {
		enc = EncodeFunc(encValue)
	}
	cfg.fmtr.value = ttyEncoder[Value]{newPen(color), enc}
	return cfg
}

// Group sets a color and a pair of encoders for opening and closing groups.
// If the open or close arguments are nil, [Encoder]s that write "{" or "}" tokens are used.
func (cfg *Config) Group(color string, open Encoder[struct{}], close Encoder[int]) *Config {
	cfg.fmtr.groupPen = newPen(color)
	if open == nil {
		open = EncodeFunc(encGroupOpen)
	}
	if close == nil {
		close = EncodeFunc(encGroupClose)
	}
	cfg.fmtr.groupOpen = open
	cfg.fmtr.groupClose = close
	return cfg
}

// Source sets a color and an encoder for [SourceLine] encoding.
// If the enc argument is nil, the configuration uses the [SourceAbs] function.
// Configurations must set [Config.AddSource] to output source annotations.
func (cfg *Config) Source(color string, enc Encoder[SourceLine]) *Config {
	if enc == nil {
		enc = EncodeFunc(encSourceAbs)
	}
	cfg.fmtr.source = ttyEncoder[SourceLine]{newPen(color), enc}
	return cfg
}

// Tag configures tagging values with the given key.
// If tagged, an [Attr]'s value appears,in the given color, in the "tags" field of the log line.
func (cfg *Config) Tag(key string, color string) *Config {
	tag := ttyEncoder[Attr]{newPen(color), EncodeFunc[Attr](encTag)}
	cfg.fmtr.tag[key] = tag
	return cfg
}

// Tag configures tagging values with the given key.
// If tagged, an [Attr] appears, in the given color, encoded by the provided [Encoder], in the "tags" field of the log line.
func (cfg *Config) TagEncode(key string, color string, enc Encoder[Attr]) *Config {
	tag := ttyEncoder[Attr]{newPen(color), enc}
	cfg.fmtr.tag[key] = tag
	return cfg
}

// AddSource configures the inclusion of source file and line information in log lines.
func (cfg *Config) AddSource(toggle bool) *Config {
	cfg.fmtr.addSource = toggle
	return cfg
}

// Layout configures the fields encoded in a [TTY] log line.
//
// Layout recognizes the following strings (and ignores others):
//
// Log fields:
//   - "time"
//   - "level"
//   - "message"
//   - "attrs"
//   - "tags"
//   - "source"
//
// Spacing:
//   - "\n"
//   - " "
//   - "\t"
//
// If [Config.AddSource] is configured, source information is the last field encoded in a log line.
func (cfg *Config) Layout(fields ...string) *Config {
	cfg.fmtr.layout = cfg.fmtr.layout[:0]

	var f ttyField
	for _, s := range fields {
		switch s {
		case " ":
			f = ttySpaceField
		case "\n":
			f = ttyNewlineField
		case "\t":
			f = ttyTabField
		case "time":
			f = ttyTimeField
		case "level":
			f = ttyLevelField
		case "message":
			f = ttyMessageField
		case "attrs":
			f = ttyAttrsField
		case "tags":
			f = ttyTagsField
		case "source":
			f = ttySourceField
		default:
			continue
		}

		cfg.fmtr.layout = append(cfg.fmtr.layout, f)
	}
	return cfg
}

// ReplaceAttr configures the use of the given function to replace Attrs when logging.
// See [slog.HandlerOptions].
func (cfg *Config) ReplaceFunc(replace func(a Attr) Attr) *Config {
	cfg.replace = replace
	return cfg
}

// TTY returns a new TTY.
// If the configured Writer is the same as [StdTTY] (default: [os.Stdout]), the new TTY shares a mutex with [StdTTY].
func (cfg *Config) TTY() *TTY {
	// SINK
	sink := &ttySink{
		w:       cfg.w,
		ref:     cfg.ref,
		replace: cfg.replace,
	}

	if cfg.useStdMutex {
		sink.mu = &stdMutex
	} else {
		sink.mu = new(sync.Mutex)
	}

	if cfg.forceTTY {
		sink.enabled = true
	} else {
		sink.enabled = writerIsTerminal(sink.w)
	}

	// FORMATTER
	fmtr := cfg.fmtr

	fmtr.sink = sink

	var sourceInLayout bool
	fmtr.layout = make([]ttyField, len(cfg.fmtr.layout))
	for i, f := range cfg.fmtr.layout {
		fmtr.layout[i] = f
		if f == ttySourceField {
			sourceInLayout = true
		}
	}

	if fmtr.addSource && !sourceInLayout {
		fmtr.layout = append(fmtr.layout, ttyNewlineField, ttySourceField)
	}

	fmtr.tag = make(map[string]ttyEncoder[Attr], len(cfg.fmtr.tag))
	for k, v := range cfg.fmtr.tag {
		fmtr.tag[k] = v
	}

	if !cfg.useColors {
		fmtr.time.color = ""
		fmtr.level.color = ""
		fmtr.message.color = ""
		fmtr.key.color = ""
		fmtr.value.color = ""
		fmtr.source.color = ""

		fmtr.groupPen = ""
		fmtr.debugPen = ""
		fmtr.infoPen = ""
		fmtr.warnPen = ""
		fmtr.errorPen = ""

		fmtr.tag["#"] = ttyEncoder[Attr]{
			"",
			EncodeFunc(encTag),
		}
	}

	// TTY
	return &TTY{
		tag:  slog.String("", ""),
		fmtr: &fmtr,
	}
}

func (cfg *Config) ForceTTY() *Config {
	cfg.forceTTY = true
	return cfg
}

// If the configured Writer is a terminal, the returned [*Logger] is [TTY]-based
// Otherwise, the returned [*Logger] a JSONHandler]-based
func (cfg *Config) Logger() Logger {
	return cfg.
		TTY().
		Logger()
}

// Printer returns a [TTY]-based Logger that only emits tags and messages.
// If the configured Writer is a terminal, the returned [Logger] is [TTY]-based
// Otherwise, the returned [Logger] a JSONHandler]-based
func (cfg *Config) Printer() Logger {
	return cfg.
		Layout("tags", "message").
		TTY().
		Logger()
}

// JSON returns a Logger using a [slog.JSONHandler] for encoding.
//
// Only [Config.Writer], [Config.Level], [Config.AddSource], and [Config.ReplaceFunc] configuration is applied.
func (cfg *Config) JSON() Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.ref,
		AddSource:   cfg.fmtr.addSource,
		ReplaceAttr: cfg.replace,
	}.NewJSONHandler(cfg.w)

	return Logger{
		h: &Handler{
			tag:       slog.String("", ""),
			enc:       enc,
			addSource: cfg.fmtr.addSource,
			replace:   cfg.replace,
		},
	}
}

// Text returns a Logger using a [slog.TextHandler] for encoding.
//
// Only [Config.Writer], [Config.Level], [Config.AddSource], and [Config.ReplaceFunc] configuration is applied.
func (cfg *Config) Text() Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.ref,
		AddSource:   cfg.fmtr.addSource,
		ReplaceAttr: cfg.replace,
	}.NewTextHandler(cfg.w)

	return Logger{
		h: &Handler{
			tag:       slog.String("", ""),
			enc:       enc,
			addSource: cfg.fmtr.addSource,
			replace:   cfg.replace,
		},
	}
}
