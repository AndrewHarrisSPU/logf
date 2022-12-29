package logf

import (
	"errors"
	"io"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

func BenchmarkLoggerSize(b *testing.B) {
	b.Run("logf manual", benchLogfInitManual)
	b.Run("logf init", benchLogfInit)
	b.Run("logf with 5", benchLogfWith5)
	b.Run("logf with 10", benchLogfWith10)
	b.Run("logf with 40", benchLogfWith40)
	b.Run("slog init", benchSlogInit)
	b.Run("slog with 5", benchSlogWith5)
	b.Run("slog with 10", benchSlogWith10)
	b.Run("slog with 40", benchSlogWith40)
}

var globalLog Logger
var globalSlog *slog.Logger

func benchLogfInitManual(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h := &Handler{
			// attrs:     make([]Attr, 0),
			enc:       slog.NewJSONHandler(io.Discard),
			addSource: false,
		}
		globalLog = newLogger(h)
	}
}

func benchLogfInit(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalLog = New().
			Writer(io.Discard).
			JSON()
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
		_ = New().
			Writer(io.Discard).
			JSON().
			With(TestAny5...)
	}
}

func benchLogfWith10(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = New().
			Writer(io.Discard).
			JSON().
			With(TestAny10...)
	}
}

func benchLogfWith40(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = New().
			Writer(io.Discard).
			JSON().
			With(TestAny40...)
	}
}

func benchSlogWith5(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = slog.New(slog.NewJSONHandler(io.Discard)).With(TestAny5...)
	}
}

func benchSlogWith10(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = slog.New(slog.NewJSONHandler(io.Discard)).With(TestAny10...)
	}
}

func benchSlogWith40(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = slog.New(slog.NewJSONHandler(io.Discard)).With(TestAny40...)
	}
}

func BenchmarkAttrs(b *testing.B) {
	for _, handler := range []struct {
		name string
		h    slog.Handler
	}{
		// {"async discard", newAsyncHandler()},
		// {"fastText discard", newFastTextHandler(io.Discard)},
		{"Text discard", slog.NewTextHandler(io.Discard)},
		{"JSON discard", slog.HandlerOptions{AddSource: false}.NewJSONHandler(io.Discard)},
		{"logf discard", New().Writer(io.Discard).JSON().Handler().(handler)},
	} {
		logger := slog.New(handler.h)
		b.Run(handler.name, func(b *testing.B) {
			for _, call := range []struct {
				name string
				f    func()
			}{
				{
					"0 args",
					func() {
						logger.LogAttrs(slog.InfoLevel, TestMessage)
					},
				},
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

func BenchmarkSplicer(b *testing.B) {
	b.Run("splicer init/free", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := newSplicer()
			defer s.free()
		}
	})

	b.Run("splicer scan", func(b *testing.B) {
		s := newSplicer()
		defer s.free()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s.scanMessage(TestMessage)
		}
	})

	// b.Run("splicer join 5 attrs 5 args", func(b *testing.B) {
	// 	s := newSplicer()
	// 	defer s.free()

	// 	b.ReportAllocs()
	// 	for i := 0; i < b.N; i++ {
	// 		s.joinAttrList(TestAttrs)
	// 		s.joinList(TestAny5)
	// 	}
	// })

	b.Run("splicer interpolate 5 unkeyed", func(b *testing.B) {
		s := newSplicer()
		defer s.free()

		s.scanMessage("{} {} {} {} {}")
		s.joinStore(Store{
			scope: []string{},
			as: [][]Attr{
				TestAttrs,
			},
		}, nil)

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s.ipol("{} {} {} {} {}")
		}
	})

	b.Run("splicer interpolate 5 keyed", func(b *testing.B) {
		s := newSplicer()
		defer s.free()

		s.scanMessage("{string} {status} {duration} {time} {error}")
		s.joinStore(Store{
			scope: []string{},
			as: [][]Attr{
				TestAttrs,
			},
		}, nil)

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s.ipol("{string} {status} {duration} {time} {error}")
		}
	})
}

func BenchmarkInterpolation(b *testing.B) {
	// w := os.Stdout
	w := io.Discard

	log := New().
		Writer(w).
		JSON()

	log5 := log.With(TestAny5...)
	log10 := log.With(TestAny10...)
	log40 := log.With(TestAny40...)

	slogger := slog.New(slog.NewJSONHandler(w))
	slogger5 := slogger.With(TestAny5...)
	slogger40 := slogger.With(TestAny40...)

	fs := []struct {
		label string
		fn    func()
	}{
		{
			label: "0 interp, 5 args",
			fn:    func() { log.Info("", TestAny5...) },
		},
		{
			label: "slog, 5 args",
			fn:    func() { slogger.Info("", TestAny5...) },
		},
		{
			label: "5 unkeyed, 5 args",
			fn: func() {
				log.Info("{} {} {} {} {}",
					TestAny5[0],
					TestAny5[1],
					TestAny5[2],
					TestAny5[3],
					TestAny5[4],
				)
			},
		},
		{
			label: "0 interp, with 5",
			fn:    func() { log5.Info(TestMessage) },
		},
		{
			label: "slogger with 5",
			fn:    func() { slogger5.Info(TestMessage) },
		},
		{
			label: "string interp, with 5",
			fn:    func() { log5.Info("{string}") },
		},
		{
			label: "string interp, with 40",
			fn:    func() { log40.Info("{string}") },
		},
		{
			label: "time interp, with 5",
			fn:    func() { log5.Info("{time}") },
		},
		{
			label: "all interp, arg 5",
			fn:    func() { log.Info("{string} {status} {duration} {time} {error}", TestAny5...) },
		},
		{
			label: "all interp, with 5",
			fn:    func() { log5.Info("{string} {status} {duration} {time} {error}") },
		},
		{
			label: "all interp, with 10",
			fn: func() {
				log10.Info("{string} {status} {duration} {time} {error} {string2} {status2} {duration2} {time2} {error2}")
			},
		},
		{
			label: "all interp, with 40",
			fn: func() {
				log40.Info(`{1} {2} {3} {4} {5} {6} {7} {8} {9} {10} {11} {12} {13} {14} {15} {16} {17} {18} {19} {20} {21} {22} {23} {24} {25} {26} {27} {28} {29} {30} {31} {32} {33} {34} {35} {36} {37} {38} {39} {40}`)
			},
		},
		{
			label: "0 interp, with 40",
			fn:    func() { log40.Info("") },
		},
		{
			label: "slogger with 40",
			fn:    func() { slogger40.Info(TestMessage) },
		},
	}

	// for _, f := range fs {
	// 	println(f.label)
	// 	f.fn()
	// }

	for _, f := range fs {
		b.Run(f.label, func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					f.fn()
				}
			})
		})
	}
}

// func TestSanity(t *testing.T){
// 	w := os.Stdout
// 	log := New.
// Writer(w).
// JSON()
// 	log40 := log.With(TestAny40...)
// 	log40.Info( `{1} {2} {3} {4} {5} {6} {7} {8} {9} {10} {11} {12} {13} {14} {15} {16} {17} {18} {19} {20} {21} {22} {23} {24} {25} {26} {27} {28} {29} {30} {31} {32} {33} {34} {35} {36} {37} {38} {39} {40}`)
// }

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
	slog.String("string2", TestString),
	slog.Int("status2", TestInt),
	slog.Duration("duration2", TestDuration),
	slog.Time("time2", TestTime),
	slog.Any("error2", TestError),
}

var TestAny40 = []any{
	slog.String("1", TestString),
	slog.Int("11", TestInt),
	slog.Duration("21", TestDuration),
	slog.Time("31", TestTime),
	slog.Any("2", TestError),
	slog.String("12", TestString),
	slog.Int("22", TestInt),
	slog.Duration("32", TestDuration),
	slog.Time("3", TestTime),
	slog.Any("13", TestError),
	slog.String("23", TestString),
	slog.Int("33", TestInt),
	slog.Duration("4", TestDuration),
	slog.Time("14", TestTime),
	slog.Any("24", TestError),
	slog.String("34", TestString),
	slog.Int("5", TestInt),
	slog.Duration("15", TestDuration),
	slog.Time("25", TestTime),
	slog.Any("35", TestError),
	slog.String("6", TestString),
	slog.Int("16", TestInt),
	slog.Duration("26", TestDuration),
	slog.Time("36", TestTime),
	slog.Any("7", TestError),
	slog.String("17", TestString),
	slog.Int("27", TestInt),
	slog.Duration("37", TestDuration),
	slog.Time("8", TestTime),
	slog.Any("18", TestError),
	slog.String("28", TestString),
	slog.Int("38", TestInt),
	slog.Duration("9", TestDuration),
	slog.Time("19", TestTime),
	slog.Any("29", TestError),
	slog.String("39", TestString),
	slog.Int("10", TestInt),
	slog.Duration("20", TestDuration),
	slog.Time("30", TestTime),
	slog.Any("40", TestError),
}
