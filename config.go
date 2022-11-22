package logf

import (
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

var stdRef slog.LevelVar
var stdSink ttySink
var stdMutex *sync.Mutex

func writerIsTerminal(w io.Writer) bool {
	file, isFile := w.(*os.File)
	if !isFile {
		return false
	}

	stat, _ := file.Stat()
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func init() {
	ref := new(slog.LevelVar)
	mu := new(sync.Mutex)
	stdMutex = new(sync.Mutex)

	stdSink = ttySink{
		w:       os.Stdout,
		refBase: ref,
		mu:      mu,
	}

	stdSink.ref.Set(stdSink.refBase.Level())
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
// Methods applying to any handler or logger produced by the Config, and default values:
//   - [Config.Writer]: os.Stdout
//   - [Config.Ref]: logf.StdRef
//   - [Config.AddSource]: false
//   - [Config.Level]: INFO
//   - [Config.ReplaceAttr]: (none)
//
// Methods applying only to a [TTY], or a logger based on one, and default values:
//   - [Config.Layout]: time, level, message, attrs
//   - [Config.Colors]: true
//   - [Config.TimeLayout]: "15:04:05"
//   - [Confg.Elapsed]: false
//   - [Config.AddLabel]: true
//   - [Config.Stream]: (disabled) hold: nil, log: nil
//   - [Config.StreamSizes]: sample: 5 lines, hold: 1 line
//   - [Config.StreamRefresh]: 16 milliseconds
//
// 3. A Config method returning a [Logger] or a [TTY] closes the chained invocation:
//   - [Config.TTY] returns a [TTY]
//   - [Config.Logger] returns a [Logger] based on a [TTY].
//   - [Config.Printer] returns a [Logger], based on a [TTY], with a preset layout.
//   - [Config.Streamer] returns a [Logger], based on a [TTY], with a preset streaming configuration.
//   - [Config.JSON] returns a [Logger] based on a [slog.JSONHandler]
//   - [Config.Text] returns a [Logger] based on a [slog.TextHandler]
//
// TODO: document mutex edge cases
type Config struct {
	w          io.Writer
	ref        slog.Leveler
	layout     []ttyField
	replace    func(Attr) Attr
	timeFormat string

	logLevel  slog.Level
	holdLevel slog.Level
	holdCap   int
	sampleCap int
	streaming bool
	refresh   int

	forceTTY    bool
	elapsed     bool
	colors      bool
	addSource   bool
	addLabel    bool
	useStdMutex bool
}

// New opens a Config with default values.
func New() *Config {
	cfg := &Config{
		w:   os.Stdout,
		ref: &stdSink.ref,
		layout: []ttyField{
			ttyTimeField,
			ttyLevelField,
			ttyMessageField,
			ttyAttrsField,
		},
		timeFormat:  "15:04:05",
		colors:      true,
		addLabel:    true,
		sampleCap:   5,
		holdCap:     1,
		refresh:     16,
		useStdMutex: true,
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

// TimeFormat sets a [time.Layout] layout.
// When set, the time field of a log line encodes with the provided layout.
func (cfg *Config) TimeFormat(layout string) *Config {
	cfg.timeFormat = layout
	return cfg
}

// Elapsed configures reporting elapsed time rather than wall clock time in log lines.
// The start time is the creation time of a [TTY], or the time set by [TTY.StartTimeNow].
func (cfg *Config) Elapsed(toggle bool) *Config {
	cfg.elapsed = toggle
	return cfg
}

// Colors toggles [TTY] color encoding, using ANSI escape codes.
//
// TODO: support cygwin escape codes.
func (cfg *Config) Colors(toggle bool) *Config {
	cfg.colors = toggle
	return cfg
}

// AddLabel configures a [TTY] to emit messages that include a label.
// The label is the most recent name set by [Logger.Label] (or [slog.Handler.WithGroup]).
func (cfg *Config) AddLabel(toggle bool) *Config {
	cfg.addLabel = toggle
	return cfg
}

// AddSource configures the inclusion of source file and line information in log lines.
func (cfg *Config) AddSource(toggle bool) *Config {
	cfg.addSource = toggle
	return cfg
}

func (cfg *Config) sourceOK() {
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
}

// Stream configures a [TTY] for streaming display.
// When streaming, log lines may enter a hold buffer or a sample buffer.
// These are displayed at the tail of [TTY] output, and display the n most recent log lines they capture.
func (cfg *Config) streamOK() {
	// bail if no stream capacity
	if cfg.sampleCap == 0 && cfg.holdCap == 0 {
		cfg.streaming = false
		return
	}

	return
}

// [TTY] streaming logic is parameterized by the hold and log levels.
//   - A log line is only created when it exceeds the reference level.
//   - If the resulting log line is below the hold level, it enters the sample buffer.
//   - If the resulting log line is below the log level, it enters the hold buffer.
//   - Otherwise, the log line is written more permanently to terminal output.
func (cfg *Config) Stream(hold, log slog.Level) *Config {
	cfg.streaming = true
	if hold > log {
		hold = log
	}
	cfg.holdLevel = hold
	cfg.logLevel = log

	return cfg
}

// StreamSizes configures the number of lines a [TTY]'s' sample and hold buffers.
func (cfg *Config) StreamSizes(sample, hold int) *Config {
	if sample < 1 {
		sample = 0
	}

	if hold < 1 {
		hold = 0
	}

	cfg.sampleCap = sample
	cfg.holdCap = hold

	return cfg
}

// StreamRefresh configures the number of milliseconds between on-screen updates to buffers.
// A [TTY] may modulate the reference level it enables logging at between refresh ticks.
// By dropping messages that would otherwise enter the sample buffer, the [TTY] may better uncouple from a high volume of logging.
// Messages that would enter the hold buffer or be logged directly are not dropped in this scheme.
func (cfg *Config) StreamRefresh(ms int) *Config {
	cfg.refresh = ms
	return cfg
}

// Layout configures the fields encoded in a [TTY] log line.
//
// Layout recognizes the following strings (and ignores others):
//   - "time" (see also: [Config.Elapsed] and [Config.TimeFormat])
//   - "level"
//   - "message" (see also: [Config.AddLabel])
//   - "attrs"
//
// If [Config.AddSource] is configured, source information is the last field encoded in a log line.
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

// ReplaceAttr configures the use of the given function to replace Attrs when logging.
// See [slog.HandlerOptions].
func (cfg *Config) ReplaceFunc(replace func(a Attr) Attr) *Config {
	cfg.replace = replace
	return cfg
}

// TTY returns a new TTY.
// If the configured Writer is the same as [StdTTY] (default: [os.Stdout]), the new TTY shares a mutex with [StdTTY].
func (cfg *Config) TTY() *TTY {
	// finalize source and stream details
	cfg.sourceOK()
	cfg.streamOK()

	d := time.Duration(cfg.refresh) * time.Millisecond

	// SINK
	sink := &ttySink{
		w:       cfg.w,
		refresh: time.NewTicker(d),
		done:    make(chan struct{}),
		start:   time.Now(),
		replace: cfg.replace,
	}

	// TODO: is this quite right?
	if cfg.useStdMutex {
		sink.mu = stdMutex
	} else {
		sink.mu = new(sync.Mutex)
	}

	sink.ref.Set(cfg.ref.Level())
	sink.refBase = cfg.ref
	if cfg.forceTTY {
		sink.enabled = true
	} else {
		sink.enabled = writerIsTerminal(sink.w)
	}

	if cfg.streaming {
		sink.stream = ttyStream{
			enabled:   true,
			logLevel:  cfg.logLevel,
			holdLevel: cfg.holdLevel,
			hold: spinner{
				cap: cfg.holdCap,
			},
			sample: spinner{
				cap: cfg.sampleCap,
			},
		}
	}

	// ENCODE
	enc := &ttyEncoder{
		sink:       sink,
		fields:     cfg.layout,
		timeFormat: cfg.timeFormat,
		colors:     cfg.colors,
		elapsed:    cfg.elapsed,
		addSource:  cfg.addSource,
	}

	if sink.stream.enabled {
		go sink.update()
	}

	// TTY
	return &TTY{
		enc: enc,
	}
}

func (cfg *Config) ForceTTY() *Config {
	cfg.forceTTY = true
	return cfg
}

// If the configured Writer is a terminal, the returned [*Logger] is [TTY]-based
// Otherwise, the returned [*Logger] a JSONHandler]-based
func (cfg *Config) Logger() *Logger {
	return cfg.
		TTY().
		Logger()
}

// Printer returns a [TTY]-based Logger that only emits messages.
// If the configured Writer is a terminal, the returned [*Logger] is [TTY]-based
// Otherwise, the returned [*Logger] a JSONHandler]-based
func (cfg *Config) Printer() *Logger {
	return cfg.
		Layout("label", "message").
		TTY().
		Logger()
}

// Streamer returns a [TTY]-based Logger configured for streaming.
// The [Logger] samples below INFO, and prints above WARN.
// If the configured Writer is a terminal, the returned [*Logger] is [TTY]-based
// Otherwise, the returned [*Logger] a JSONHandler]-based
func (cfg *Config) Streamer() *Logger {
	return cfg.
		Stream(INFO, WARN).
		TTY().
		Logger()
}

// JSON returns a Logger using a [slog.JSONHandler] for encoding.
//
// Only [Config.Writer], [Config.Level], [Config.AddSource], and [Config.ReplaceAttr] configuration is applied.
func (cfg *Config) JSON() *Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.ref,
		AddSource:   cfg.addSource,
		ReplaceAttr: cfg.replace,
	}.NewJSONHandler(cfg.w)

	return &Logger{
		h: &Handler{
			enc:       enc,
			addSource: cfg.addSource,
			replace:   cfg.replace,
		},
	}
}

// Text returns a Logger using a [slog.TextHandler] for encoding.
//
// Only [Config.Writer], [Config.Level], [Config.AddSource], and [Config.ReplaceAttr] configuration is applied.
func (cfg *Config) Text() *Logger {
	enc := slog.HandlerOptions{
		Level:       cfg.ref,
		AddSource:   cfg.addSource,
		ReplaceAttr: cfg.replace,
	}.NewTextHandler(cfg.w)

	return &Logger{
		h: &Handler{
			enc:       enc,
			addSource: cfg.addSource,
			replace:   cfg.replace,
		},
	}
}
