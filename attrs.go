package logf

import (
	"golang.org/x/exp/slog"
)

func KV(key string, value any) Attr {
	return slog.Any(key, value)
}

func Group(key string, as ...Attr) slog.Attr {
	return slog.Group(key, as...)
}

func Attrs(args ...any) (as []Attr) {
	for len(args) > 0 {
		switch arg := args[0].(type) {
		case string:
			if len(args) == 1 {
				as = append(as, slog.String(arg, missingArg))
				return
			}
			as = append(as, slog.Any(arg, args[1]))
			args = args[2:]
		default:
			as = append(as, slog.Any(missingKey, arg))
			args = args[1:]
		}
	}
	return
}

// segment munges arguments to Attrs, returning a slice of attrs 'seg'.
//   - Pairs of (string, any) result in an Attr, appended to seg.
//   - Attrs are appended to seg.
//   - []Attrs, contexts, and logf's Loggers, CtxLoggers, Handlers are flattened and appended seg.
func segment(args ...any) (seg []Attr) {
	for len(args) > 0 {
		switch arg := args[0].(type) {
		case string:
			if len(args) == 1 {
				seg = append(seg, slog.String(arg, missingArg))
				return
			}
			seg = append(seg, slog.Any(arg, args[1]))
			args = args[2:]
		case slog.LogValuer:
			v := arg.LogValue()
			if v.Kind() == slog.GroupKind {
				seg = append(seg, v.Group()...)
			}
			args = args[1:]
		case Attr:
			seg = append(seg, arg)
			args = args[1:]
		case []Attr:
			seg = append(seg, arg...)
			args = args[1:]
		default:
			seg = append(seg, slog.Any(missingKey, arg))
			args = args[1:]
		}
	}
	return
}
