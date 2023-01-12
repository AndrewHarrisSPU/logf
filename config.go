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

// CONFIG

// Config is a base type for [Logger] and [TTY] configuration.
//
// To construct a [Logger] with an already extant [slog.Handler], see [UsingHandler].
//
// If a [TTY] would employ a Writer that isn't a terminal, Config methods result in a [slog.JSONHandler]-based [Logger],
// unless [Config.ForceTTY] is set. [Config.Aux] is available for additional configuration of auxilliary logging.
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
//   - [Config.ReplaceFunc]: nil
//
// Methods applying only to a [TTY], or a logger based on one, and default arguments:
//   - [Config.Aux]: none
//   - [Config.ForceAux]: false
//   - [Config.ForceTTY]: false
//
// Methods configuring the color and encoding of [TTY] fields:
//   - [Config.ShowAttrKey]
//   - [Config.ShowAttrValue]
//   - [Config.ShowColor]: true
//   - [Config.ShowGroup]: "dim"
//   - [Config.ShowLayout]: "level", "time", "tags", "message", "\t", "attrs"
//   - [Config.ShowLevel]: LevelBar
//   - [Config.ShowLevelColors]: "bright cyan", "bright green", "bright yellow", "bright red"
//   - [Config.ShowMessage]: ""
//   - [Config.ShowSource]: "dim", SourceAbs
//   - [Config.ShowTag]: "#", "bright magenta"
//   - [Config.ShowTagEncode]: nil
//   - [Config.ShowTime]: "dim", TimeShort
//
// 3. A Config method returning a [Logger] or a [TTY] closes the chained invocation:
//   - [Config.TTY] returns a [TTY]
//   - [Config.Logger] returns a [Logger] based on a [TTY].
//   - [Config.Printer] returns a [Logger], based on a [TTY], with a preset layout.
//   - [Config.JSON] returns a [Logger] based on a [slog.JSONHandler]
//   - [Config.Text] returns a [Logger] based on a [slog.TextHandler]
type Config struct {
	w *ttySyncWriter

	// slog.Handler config
	ref     *slog.LevelVar
	replace func([]string, Attr) Attr

	// tty gadgets
	aux        slog.Handler
	fmtr       *ttyFormatter
	addSource  bool
	addColors  bool
	enableTTY  bool
	forceTTY   bool
	forceAux   bool
	setDefault bool
}

// New opens a Config with default values.
func New() *Config {
	w, enableTTY := newTTYSyncWriter(os.Stdout, &stdMutex)

	cfg := &Config{
		w:         w,
		ref:       &StdRef,
		replace:   nil,
		addColors: true,

		fmtr:      newTTYFormatter(),
		enableTTY: enableTTY,
	}

	return cfg
}

// NewDefault is in all ways similar to [New], except that
// using NewDefault configures the first logger or handler produced by the configuration to become the slogging default,
// using [slog.SetDefault].
func NewDefault() *Config {
	cfg := New()
	cfg.setDefault = true
	return cfg
}

// CONFIG INTERNAL FIELDS

// Ref configures the use of the given reference [slog.LevelVar].
func (cfg *Config) Ref(level *slog.LevelVar) *Config {
	cfg.ref = level
	return cfg
}

// Writer configures the eventual destination of log lines.
// Configuring a new writer creates a new mutex guarding it.
func (cfg *Config) Writer(w io.Writer) *Config {
	cfg.w, cfg.enableTTY = newTTYSyncWriter(w, new(sync.Mutex))
	return cfg
}

// Colors toggles [TTY] color encoding, using ANSI escape codes.
//
// TODO: support cygwin escape codes.
func (cfg *Config) ShowColor(toggle bool) *Config {
	cfg.addColors = toggle
	return cfg
}

// ShowTime sets a color and an encoder for the [slog.Record.Time] field.
// If the enc argument is nil, the configuration uses the [TimeShort] function.
func (cfg *Config) ShowTime(color string, enc Encoder[time.Time]) *Config {
	if enc == nil {
		enc = EncodeFunc(encTimeShort)
	}
	cfg.fmtr.time = ttyEncoder[time.Time]{newPen(color), enc}
	return cfg
}

// ShowLevel sets an encoder for the [slog.Record.Level] field.
// If the enc argument is nil, the configuration uses the [LevelBar] function.
func (cfg *Config) ShowLevel(enc Encoder[slog.Level]) *Config {
	if enc == nil {
		enc = EncodeFunc(encLevelBar)
	}
	cfg.fmtr.level = ttyEncoder[slog.Level]{newPen(""), enc}
	return cfg
}

// ShowLevelColors configures four colors for DEBUG, INFO, WARN, and ERROR levels.
// These colors are used when a [slog.Record.Level] is encoded.
func (cfg *Config) ShowLevelColors(debug string, info string, warn string, error string) *Config {
	cfg.fmtr.debugPen = newPen(debug)
	cfg.fmtr.infoPen = newPen(info)
	cfg.fmtr.warnPen = newPen(warn)
	cfg.fmtr.errorPen = newPen(error)
	return cfg
}

// ShowMessage sets a color for the [slog.Record.Message] field.
func (cfg *Config) ShowMessage(color string) *Config {
	cfg.fmtr.message = ttyEncoder[string]{newPen(color), nil}
	return cfg
}

// ShowAttrKey sets a color and an encoder for [slog.Attr.Key] encoding.
// If the enc argument is nil, the configuration uses an [Encoder] that simply writes the [slog.Attr.Key].
// TODO: this default does no escaping. Perhaps JSON quoting and escaping would be useful.
func (cfg *Config) ShowAttrKey(color string, enc Encoder[string]) *Config {
	if enc == nil {
		enc = EncodeFunc(encKey)
	}
	cfg.fmtr.key = ttyEncoder[string]{newPen(color), enc}
	return cfg
}

// ShowAttrValue sets a color and an encoder for [slog.Attr.Value] encoding.
// If the enc argument is nil, the configuration uses an default [Encoder].
// TODO: this default does no escaping. Perhaps JSON quoting and escaping would be useful.
func (cfg *Config) ShowAttrValue(color string, enc Encoder[Value]) *Config {
	if enc == nil {
		enc = EncodeFunc(encValue)
	}
	cfg.fmtr.value = ttyEncoder[Value]{newPen(color), enc}
	return cfg
}

// ShowGroup sets a color and a pair of encoders for opening and closing groups.
// If the open or close arguments are nil, [Encoder]s that write "{" or "}" tokens are used.
func (cfg *Config) ShowGroup(color string, open Encoder[int], close Encoder[int]) *Config {
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

// ShowSource sets a color and an encoder for [SourceLine] encoding.
// If the enc argument is nil, the configuration uses the [SourceAbs] function.
// Configurations must set [Config.AddSource] to output source annotations.
func (cfg *Config) ShowSource(color string, enc Encoder[SourceLine]) *Config {
	if enc == nil {
		enc = EncodeFunc(encSourceAbs)
	}
	cfg.fmtr.source = ttyEncoder[SourceLine]{newPen(color), enc}
	return cfg
}

// ShowTag configures tagging values with the given key.
// If tagged, an [Attr]'s value appears,in the given color, in the "tags" field of the log line.
func (cfg *Config) ShowTag(key string, color string) *Config {
	tag := ttyEncoder[Attr]{newPen(color), EncodeFunc(encTag)}
	cfg.fmtr.tag[key] = tag
	return cfg
}

// ShowTag configures tagging values with the given key.
// If tagged, an [Attr] appears, in the given color, encoded by the provided [Encoder], in the "tags" field of the log line.
func (cfg *Config) ShowTagEncode(key string, color string, enc Encoder[Attr]) *Config {
	tag := ttyEncoder[Attr]{newPen(color), enc}
	cfg.fmtr.tag[key] = tag
	return cfg
}

// AddSource configures the inclusion of source file and line information in log lines.
func (cfg *Config) AddSource(toggle bool) *Config {
	cfg.addSource = toggle
	return cfg
}

// ShowLayout configures the fields encoded in a [TTY] log line.
//
// ShowLayout recognizes the following strings (and ignores others):
//
// Log fields:
//   - "time"
//   - "level"
//   - "message" (alt "msg")
//   - "attrs" (alt "attr")
//   - "tags" (alt "tag")
//   - "source" (alt "src")
//
// Spacing:
//   - "\n" (results in a newline, followed by a tab)
//   - " "
//   - "\t"
//
// If [Config.AddSource] is configured, source information is the last field encoded in a log line.
func (cfg *Config) ShowLayout(fields ...string) *Config {
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
		case "msg", "message":
			f = ttyMessageField
		case "attr", "attrs":
			f = ttyAttrsField
		case "tag", "tags":
			f = ttyTagsField
		case "src", "source":
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
func (cfg *Config) ReplaceFunc(replace func(scope []string, a Attr) Attr) *Config {
	cfg.replace = replace
	return cfg
}

// ForceTTY configures any [TTY] produced by the configuration to always encode with
// [TTY] output. This overrides logic that otherwise falls back to JSON output when
// a configured writer is not detected to be a terminal.
func (cfg *Config) ForceTTY(toggle bool) *Config {
	cfg.forceTTY = toggle
	return cfg
}

// Aux configures an auxilliary handler for a [TTY].
// The auxilliary handler is employed:
//   - If, the [TTY]'s writer is not a tty devices, and [Config.ForceTTY] is configured false
//   - Or, if [Config.ForceAux] is configured true.
//
// If these conditions are met but no auxilliary handler has been provided,
// a [slog.JSONHandler] writing to the configured writer is used.
func (cfg *Config) Aux(aux slog.Handler) *Config {
	cfg.aux = aux
	return cfg
}

// ForceAux configures any [TTY] produced by the configuraton to always employ an
// auxilliary handler.
func (cfg *Config) ForceAux(toggle bool) *Config {
	cfg.forceAux = toggle
	return cfg
}

// CONFIG -> HANDLER/LOGGER

// TTY returns a new TTY.
// If the configured Writer is the same as [StdTTY] (default: [os.Stdout]), the new TTY shares a mutex with [StdTTY].
func (cfg *Config) TTY() *TTY {
	// WRITER
	// w, enableTTY := newTTYSyncWriter(cfg.w, cfg.mu)
	// enableTTY = enableTTY || cfg.enableTTY

	// FORMATTER
	fmtr := cfg.fmtr.clone(cfg.addSource, cfg.addColors)

	// FILTER
	filter := &ttyFilter{
		tag: make(map[string]struct{}),
	}

	// DEVICE
	dev := &ttyDevice{
		fmtr:   fmtr,
		w:      cfg.w,
		filter: filter,

		ref:     cfg.ref,
		replace: cfg.replace,
	}

	// TTY
	tty := &TTY{
		dev: dev,
	}

	setDefault := cfg.setDefault
	var enableAux bool

	if !cfg.enableTTY && !cfg.forceTTY {
		dev.w = nil
		enableAux = true
	}

	if enableAux || cfg.forceAux {
		tty.aux = cfg.aux

		if tty.aux == nil {
			// if both TTY and aux are enabled, ensure they share same mutex
			// (not elegant /shrug)
			var w io.Writer
			if !cfg.enableTTY {
				w = cfg.w.Writer
			} else {
				w = cfg.w
			}

			// build a JSON handler
			enc := slog.HandlerOptions{
				Level:       cfg.ref,
				AddSource:   cfg.fmtr.addSource,
				ReplaceAttr: cfg.replace,
			}.NewJSONHandler(w)

			h := &Handler{
				enc:       enc,
				addSource: cfg.fmtr.addSource,
				replace:   cfg.replace,
			}

			tty.aux = h
		}
	}

	if setDefault {
		slog.SetDefault(slog.New(tty))
		cfg.setDefault = false
	}

	return tty
}

// If the configured Writer is a terminal, the returned [*Logger] is [TTY]-based
// Otherwise, the returned [*Logger] a JSONHandler]-based
func (cfg *Config) Logger() Logger {
	tty := cfg.TTY()
	return newLogger(tty)
}

// Printer returns a [TTY]-based Logger that only emits tags and messages.
// If the configured Writer is a terminal, the returned [Logger] is [TTY]-based
// Otherwise, the returned [Logger] a JSONHandler]-based
func (cfg *Config) Printer() Logger {
	tty := cfg.
		ShowLayout("tags", "message").
		TTY()
	return newLogger(tty)
}

// JSON returns a Logger using a [slog.JSONHandler] for encoding.
//
// Only [Config.Writer], [Config.Level], [Config.AddSource], and [Config.ReplaceFunc] configuration is applied.
func (cfg *Config) JSON() Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.ref,
		AddSource:   cfg.fmtr.addSource,
		ReplaceAttr: cfg.replace,
	}.NewJSONHandler(cfg.w.Writer)

	h := &Handler{
		enc:       enc,
		addSource: cfg.fmtr.addSource,
		replace:   cfg.replace,
	}

	if cfg.setDefault {
		slog.SetDefault(slog.New(h))
		cfg.setDefault = false
	}

	return newLogger(h)
}

// Text returns a Logger using a [slog.TextHandler] for encoding.
//
// Only [Config.Writer], [Config.Level], [Config.AddSource], and [Config.ReplaceFunc] configuration is applied.
func (cfg *Config) Text() Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.ref,
		AddSource:   cfg.fmtr.addSource,
		ReplaceAttr: cfg.replace,
	}.NewTextHandler(cfg.w.Writer)

	h := &Handler{
		enc:       enc,
		addSource: cfg.fmtr.addSource,
		replace:   cfg.replace,
	}

	if cfg.setDefault {
		slog.SetDefault(slog.New(h))
		cfg.setDefault = false
	}

	return newLogger(h)
}
