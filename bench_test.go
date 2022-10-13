package logf

import (
	"errors"
	"io"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

func BenchmarkLoggerSize(b *testing.B) {
	// b.Run("logf manual", benchLogfInitManual)
	b.Run("logf init", benchLogfInit)
	b.Run("logf with 5", benchLogfWith5)
	b.Run("logf with 10", benchLogfWith10)
	b.Run("logf with 40", benchLogfWith40)
	// b.Run("slog init", benchSlogInit)
	// b.Run("slog with 5", benchSlogWith5)
	// b.Run("slog with 10", benchSlogWith10)
	// b.Run("slog with 40", benchSlogWith40)
}

var globalLog Logger
var globalSlog *slog.Logger

func benchLogfInitManual(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h := &Handler{
			seg:       make([]Attr, 0),
			ref:       INFO,
			enc:       slog.NewTextHandler(io.Discard),
			addSource: false,
		}
		globalLog = Logger{h, INFO}
	}
}

func benchLogfInit(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalLog = New(Using.Writer(io.Discard))
	}
}

func benchSlogInit(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalSlog = slog.New(slog.NewTextHandler(io.Discard))
	}
}

func benchLogfWith5(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = New(Using.Writer(io.Discard)).With(TestAny5...)
	}
}

func benchLogfWith10(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = New(Using.Writer(io.Discard)).With(TestAny10...)
	}
}

func benchLogfWith40(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = New(Using.Writer(io.Discard)).With(TestAny40...)
	}
}

func benchSlogWith5(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = slog.New(slog.NewTextHandler(io.Discard)).With(TestAny5...)
	}
}

func benchSlogWith10(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = slog.New(slog.NewTextHandler(io.Discard)).With(TestAny10...)
	}
}

func benchSlogWith40(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = slog.New(slog.NewTextHandler(io.Discard)).With(TestAny40...)
	}
}

func BenchmarkAttrs(b *testing.B) {
	for _, handler := range []struct {
		name string
		h    slog.Handler
	}{
		// {"async discard", newAsyncHandler()},
		// {"fastText discard", newFastTextHandler(io.Discard)},
		// {"Text discard", slog.NewTextHandler(io.Discard)},
		{"JSON discard", slog.HandlerOptions{AddSource: false}.NewJSONHandler(io.Discard)},
		// {"logf discard", NewHandler( Using.JSON, Using.Writer(io.Discard))},
	} {
		logger := slog.New(handler.h)
		b.Run(handler.name, func(b *testing.B) {
			for _, call := range []struct {
				name string
				f    func()
			}{
				{
					// The number should match nAttrsInline in slog/record.go.
					// This should exercise the code path where no allocations
					// happen in Record or Attr. If there are allocations, they
					// should only be from Duration.String and Time.String.
					"5 args",
					func() {
						logger.LogAttrs(slog.InfoLevel, TestMessage,
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
						)
					},
				},
				{
					"10 args",
					func() {
						logger.LogAttrs(slog.InfoLevel, TestMessage,
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
						)
					},
				},
				{
					"40 args",
					func() {
						logger.LogAttrs(slog.InfoLevel, TestMessage,
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
							slog.String("string", TestString),
							slog.Int("status", TestInt),
							slog.Duration("duration", TestDuration),
							slog.Time("time", TestTime),
							slog.Any("error", TestError),
						)
					},
				},
			} {
				b.Run(call.name, func(b *testing.B) {
					b.ReportAllocs()
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							call.f()
						}
					})
				})
			}
		})
	}
}

const TestMessage = "Test logging, but use a somewhat realistic message length."

var (
	TestTime     = time.Date(2022, time.May, 1, 0, 0, 0, 0, time.UTC)
	TestString   = "7e3b3b2aaeff56a7108fe11e154200dd/7819479873059528190"
	TestInt      = 32768
	TestDuration = 23 * time.Second
	TestError    = errors.New("fail")
)

var TestAttrs = []slog.Attr{
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
}

var TestAny5 = []any{
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
}

var TestAny10 = []any{
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
}

var TestAny40 = []any{
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
	slog.String("string", TestString),
	slog.Int("status", TestInt),
	slog.Duration("duration", TestDuration),
	slog.Time("time", TestTime),
	slog.Any("error", TestError),
}
