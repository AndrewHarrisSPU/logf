package logf

import (
	"golang.org/x/exp/slog"
)

// KV constructs an Attr from a key string and a value.
// See [slog.Any].
func KV(key string, value any) Attr {
	return slog.Any(key, value)
}

// Group constructs a composite Attr from a name and a list of Attrs.
// See [slog.Group].
func Group(name string, as ...Attr) Attr {
	return slog.Group(name, as...)
}

// Attrs constructs a slice of Attrs from a list of arguments. In a loop evaluating the first remaining element:
//   - A string is interpreted as a key for a following value. An Attr consuming two list elements is appended to the return.
//   - An Attr is appended to the return.
//   - A slice of Attrs is flattened into the return.
//   - A [slog.LogValuer] which resolves to a [slog.Group] is flattened into the return.
//
// Malformed lists result in Attrs indicating missing arguments, keys, or values.
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
		case Attr:
			as = append(as, arg)
			args = args[1:]
		case []Attr:
			as = append(as, arg...)
			args = args[1:]
		case slog.LogValuer:
			v := arg.LogValue().Resolve()
			if v.Kind() == slog.GroupKind {
				as = append(as, v.Group()...)
			} else {
				as = append(as, slog.Any(missingKey, arg))
			}
			args = args[1:]
		default:
			as = append(as, slog.Any(missingKey, arg))
			args = args[1:]
		}
	}
	return
}

func scopeAttrs(scope string, as []Attr, replace func(Attr) Attr) []Attr {
	scoped := make([]Attr, 0)
	for _, a := range as {
		if replace != nil {
			a = replace(a)
		}

		if a.Key == "" {
			continue
		}

		scoped = append(scoped, Attr{
			Key:   scope + a.Key,
			Value: a.Value,
		})
	}
	return scoped
}
