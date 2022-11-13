package logf

import (
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

var StdTTY *TTY
var stdMutex sync.Mutex
var stdEncoder ttyEncoder
var stdPrinter ttyEncoder

func init() {
	StdTTY = &TTY{
		level: INFO,
		enc: &ttyEncoder{
			mu:         &stdMutex,
			w:          os.Stdout,
			layout:     []ttyField{ttyTimeField, ttyLevelField, ttyMessageField, ttyAttrsField},
			colors:     true,
			addSource:  false,
			start:      time.Now(),
			timeFormat: "15:04:05",
			elapsed:    false,
			spin: spinner{
				level: INFO,
				cap:   0,
			},
			addLabel: true,
		},
	}

	stdPrinter = ttyEncoder{
		mu:     &stdMutex,
		w:      os.Stdout,
		layout: []ttyField{ttyMessageField},
		colors: true,
		start:  stdEncoder.start,
		spin: spinner{
			level: INFO,
			cap:   0,
		},
		addLabel: true,
	}
}

// CONFIG

// Config is a base type for many `logf` configuration tasks.
//
// # Most Config methods
//
// A [Logger] constructed by a Config may construct a [slog.Handler] of one of the following types:
//   - [TTY] (constructed by [Config.Logger], [Config.Printer])
//   - [slog.JSONHandler] (constructed by [Config.JSON])
//   - [slog.TextHandler] (constructed by [Config.Text])
//
// To construct a `Logger` with another `slog.Handler` type, see [UsingHandler].
//
// Defaults:
//   - Writer: os.Stdout
//   - Level: INFO
//   - Layout: "time", "level", "message", "attrs"
//   - Colors: true
//   - AddSource: false
//   - AddLabel: true
//   - Spin: INFO, 0
type Config struct {
	level      slog.Leveler
	w          io.Writer
	layout     []ttyField
	timeFormat string
	spinLevel  slog.Leveler
	spinLines  int
	elapsed    bool
	colors     bool
	addSource  bool
	addLabel   bool
	replace    func(Attr) Attr
}

func New() *Config {
	return &Config{
		level: INFO,
		w:     os.Stdout,
		layout: []ttyField{
			ttyTimeField,
			ttyLevelField,
			ttyMessageField,
			ttyAttrsField,
		},
		timeFormat: "15:04:05",
		spinLevel:  nil,
		spinLines:  0,
		elapsed:    false,
		colors:     true,
		addSource:  false,
		addLabel:   true,
	}
}

func (cfg *Config) TTY() *TTY {
	tty := new(TTY)
	tty.level = cfg.level

	tty.enc = &ttyEncoder{
		w:          cfg.w,
		layout:     cfg.layout,
		colors:     cfg.colors,
		addSource:  cfg.addSource,
		start:      time.Now(),
		timeFormat: cfg.timeFormat,
		elapsed:    cfg.elapsed,
		spin: spinner{
			level: cfg.spinLevel,
			cap:   cfg.spinLines,
		},
		addLabel: cfg.addLabel,
		replace:  cfg.replace,
	}

	if tty.enc.w == StdTTY.enc.w {
		tty.enc.mu = StdTTY.enc.mu
	} else {
		tty.enc.mu = new(sync.Mutex)
	}

	if cfg.spinLevel != nil && cfg.spinLines > 0 {
		tty.enc.spin.enabled = true
	}

	return tty
}

func (cfg *Config) Level(level slog.Leveler) *Config {
	cfg.level = level
	return cfg
}

func (cfg *Config) Writer(w io.Writer) *Config {
	cfg.w = w
	return cfg
}

// TimeFormat sets a [time.Layout] layout.
// When set, the "time" field of a log line encodes with the provided layout.
func (cfg *Config) TimeFormat(layout string) *Config {
	cfg.timeFormat = layout
	return cfg
}

// Elapsed configures [TTY] time encoding.
//   - When set, the "time" field of a log line reports elapsed time since the creation of the [TTY].
//   - When unset, the "time" field of a log line reports [Config.TimeFormat].
func (cfg *Config) Elapsed(toggle bool) *Config {
	cfg.elapsed = toggle
	return cfg
}

// Colors toggles [TTY] color encoding.
// When set, log lines may include terminal escape sequences.
func (cfg *Config) Colors(toggle bool) *Config {
	cfg.colors = toggle
	return cfg
}

// AddLabel toggles [TTY] label encoding.
//   - When set, the message field of a log lines begins with a label.
//
// A label is set by [Logger.Label], or [slog.Handler.WithGroup].
func (cfg *Config) AddLabel(toggle bool) *Config {
	cfg.addLabel = toggle
	return cfg
}

// AddSource toggles capturing the source file and line of logging calls.
//
// When set, the last element of a log line reports source information.
func (cfg *Config) AddSource(toggle bool) *Config {
	cfg.addSource = toggle

	var srcFieldOk bool
	for _, field := range cfg.layout {
		if field == ttySourceField {
			srcFieldOk = true
			break
		}
	}

	if !srcFieldOk {
		cfg.layout = append(cfg.layout, ttySourceField)
	}

	return cfg
}

// Spin configures a spin buffer designed for observing transient log lines in a [TTY].
// Log lines with level equal to or above the Spin level, but below the Config level, are logged in a spin buffer.
// The spin buffer rolls over, storing only the given number of the most recent lines logged to it.
func (cfg *Config) Spin(level slog.Leveler, lines int) *Config {
	cfg.spinLevel = level
	cfg.spinLines = lines
	return cfg
}

// Layout configures the fields encoded in a log line.
//
// Layout recognizes these strings:
//   - "time" (see also: [Config.Elapsed] and [Config.TimeFormat])
//   - "level"
//   - "message" (see also: [Config.AddLabel])
//   - "attrs"
//
// See [Config.AddSource] for source file field.
func (cfg *Config) Layout(fields ...string) *Config {
	cfg.layout = cfg.layout[:0]
	var f ttyField
	for _, text := range fields {
		switch text {
		case "time":
			f = ttyTimeField
		case "level":
			f = ttyLevelField
		case "message":
			f = ttyMessageField
		case "attrs":
			f = ttyAttrsField
		default:
			continue
		}

		cfg.layout = append(cfg.layout, f)
	}

	return cfg
}

func (cfg *Config) ReplaceAttr(replace func(a Attr) Attr) *Config {
	cfg.replace = replace
	return cfg
}

// LOGGERS

// Logger is equivalent to `cfg.TTY().Logger()`
func (cfg *Config) Logger() Logger {
	return cfg.TTY().Logger()
}

// Printer returns a Logger that only emits messages.
// It is equivalent to `cfg.Layout("label", "message").Logger()`
func (cfg *Config) Printer() Logger {
	return cfg.Layout("label", "message").TTY().Logger()
}

// JSON returns a Logger using a `slog.JSONHandler` for encoding.
// Only the configuration methods Writer, Level, AddSource apply.
func (cfg *Config) JSON() Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.level,
		AddSource:   cfg.addSource,
		ReplaceAttr: cfg.replace,
	}.NewJSONHandler(cfg.w)

	return Logger{
		h: &Handler{
			enc:       enc,
			addSource: cfg.addSource,
			replace:   cfg.replace,
		},
	}
}

// JSON returns a Logger using a `slog.JSONHandler` for encoding.
// Only the configuration methods Writer, Level, AddSource apply.
func (cfg *Config) Text() Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.level,
		AddSource:   cfg.addSource,
		ReplaceAttr: cfg.replace,
	}.NewTextHandler(cfg.w)

	return Logger{
		h: &Handler{
			enc:       enc,
			addSource: cfg.addSource,
			replace:   cfg.replace,
		},
	}
}
