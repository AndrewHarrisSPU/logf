package logf

import (
	"encoding/json"
	"errors"
	"strings"

	"log/slog"
	"slices"
	"strconv"
)

type replaceFunc func([]string, Attr) Attr

// KV constructs an Attr from a key string and a value.
// See [slog.Any].
func KV(key string, value any) Attr {
	return slog.Any(key, value)
}

// See [slog.Group].
func Group(name string, as ...any) Attr {
	return slog.Group(name, as...)
}

// See [slog.GroupValue]
func GroupValue(as ...Attr) Value {
	return slog.GroupValue(as...)
}

func expandAttr(list *[]Attr, a Attr) {
	*list = append(*list, a)
}

func expandValuerGroup(list *[]Attr, prefix string, v Value) {
	as := v.Group()
	for _, a := range as {
		a.Key = prefix + a.Key

		expandAttr(list, a)
	}
}

func expandValuer(list *[]Attr, prefix string, lv slog.LogValuer) {
	v := lv.LogValue().Resolve()
	if v.Kind() == slog.KindGroup {
		expandValuerGroup(list, prefix+".", v)
	} else {
		*list = append(*list, slog.Any(prefix, v))
	}
}

func expandHandler(list *[]Attr, prefix string, h slog.Handler) {
	if lv, ok := h.(slog.LogValuer); ok {
		group := lv.LogValue()
		if group.Kind() == slog.KindGroup {
			as := scopeAttrs(prefix, group.Group(), nil)
			*list = append(*list, as...)
		}
	}
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

			// intercept / expand a LogValuer
			if lv, ok := args[1].(slog.LogValuer); ok {
				expandValuer(&as, arg, lv)
				args = args[2:]
				continue
			}

			expandAttr(&as, slog.Any(arg, args[1]))
			args = args[2:]

		case Attr:
			expandAttr(&as, arg)
			args = args[1:]

		case []Attr:
			for _, a := range arg {
				expandAttr(&as, a)
			}
			args = args[1:]

		case *slog.Logger:
			expandHandler(&as, "", arg.Handler())
			args = args[1:]

		case Logger:
			expandHandler(&as, "", arg.Handler())
			args = args[1:]

		case slog.LogValuer:
			expandValuer(&as, "", arg)
			args = args[1:]

		default:
			as = append(as, slog.Any(missingKey, arg))
			args = args[1:]
		}
	}
	return
}

func scopeAttrs(scope string, as []Attr, replace replaceFunc) []Attr {
	if scope == "" {
		return as
	}

	scoped := make([]Attr, 0)
	for _, a := range as {
		if replace != nil {
			a = replace(nil, a)
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

func detectLabel(as []Attr, label Attr) ([]Attr, Attr) {
	var ii int

	for i := range as {
		if as[i].Key == "#" {
			label = as[i]
		} else {
			as[ii] = as[i]
			ii++
		}
	}

	return as[:ii], label
}

// Store implements the `WithAttrs` and `WithGroup` methods of the [slog.Handler] interface.
// Additionally, a Store is a [slog.LogValuer].
type Store struct {
	scope []string
	as    [][]Attr
}

var attrStoreEmptyTail = Attr{}

// LogValue returns a [Value]
func (store Store) LogValue() Value {
	depth := len(store.scope)
	tail := attrStoreEmptyTail

	for depth >= 0 {
		tail = store.frame(depth, tail)
		depth--
	}

	return tail.Value
}

func (store Store) attrsDepth(depth int) []Attr {
	if len(store.as) == len(store.scope) {
		return []Attr{}
	}
	return store.as[depth]
}

func (store Store) attrsDepthAny(depth int) []any {
	as := store.attrsDepth(depth)
	var list []any
	for _, a := range as {
		list = append(list, a)
	}
	return list
}

func (store Store) keyDepth(depth int) string {
	if depth == 0 {
		return ""
	}
	return store.scope[depth-1]
}

// Builds a Group-kind Attr capturing the given frame, and ending with the
// provided tail Attr. This routine is used when building a Store's LogValue,
// iteratively using the result at depth as the tail for another frame call,
// for a successively shallower/lower depth.
func (store Store) frame(depth int, tail Attr) Attr {
	emptyTail := tail.Key == attrStoreEmptyTail.Key && tail.Value.Equal(attrStoreEmptyTail.Value)
	emptyFrame := len(store.attrsDepth(depth)) == 0

	if emptyTail && emptyFrame {
		return attrStoreEmptyTail
	}

	key := store.keyDepth(depth)

	if emptyTail {
		list := append([]any{}, store.attrsDepthAny(depth)...)
		return slog.Group(key, list...)
	}

	if emptyFrame {
		return slog.Group(key, tail)
	}

	return slog.Group(key, concatOne(store.attrsDepthAny(depth), any(tail))...)
}

// Attrs traverses attributes in the [Store], applying the given function to each visited attribute.
// The first, []string-valued argument represents a stack of group keys,
// (same idea as replace functions given to [slog.HandlerOptions]). The
// second is an attribute encountered in traversal.
func (store Store) Attrs(f func([]string, Attr)) {
	for depth := 0; depth <= len(store.scope); depth++ {
		if len(store.as) == depth {
			return
		}
		for _, a := range store.as[depth] {
			f(store.scope[:depth], a)
		}
	}
}

// ReplaceAttr resembles functionality seen in [slog.HandlerOptions]. Unlike [Store.Attrs], it can
// be used to mutate attributes held in the store.
func (store Store) ReplaceAttr(f func([]string, Attr) Attr) {
	for depth := 0; depth <= len(store.scope); depth++ {
		if len(store.as) == depth {
			return
		}
		for i, a := range store.as[depth] {
			store.as[depth][i] = f(store.scope[:depth], a)
		}
	}
}

// WithGroup opens a new group in the [Store].
func (store Store) WithGroup(name string) Store {
	as := slices.Clone(store.as)
	for len(as) <= len(store.scope) {
		as = append(as, []Attr{})
	}

	return Store{
		scope: concatOne(store.scope, name),
		as:    as,
	}
}

// WithAttrs commits attributes to the [Store].
func (store Store) WithAttrs(as []Attr) Store {
	as2 := slices.Clone(store.as)

	if len(as2) == len(store.scope) {
		as2 = concatOne(as2, slices.Clone(as))
	} else {
		as2[len(store.scope)] = concat(store.as[len(store.scope)], as)
	}

	return Store{
		scope: store.scope,
		as:    as2,
	}
}

// JSONValue converst a JSON object to a [Value]. Array values are expanded
// to attributes with a key string derived from array index (i.e., the 0th element is keyed "0").
func JSONValue(object string) (Value, error) {
	dec := json.NewDecoder(strings.NewReader(object))
	dec.UseNumber()

	v, err := parseValue(dec)
	return v, err
}

func parseKey(dec *json.Decoder) (string, error) {
	keyToken, err := dec.Token()
	if err != nil {
		return "", err
	}
	key, ok := keyToken.(string)
	if !ok {
		return "", errors.New("parseKey")
	}
	return key, nil
}

func parseValue(dec *json.Decoder) (Value, error) {
	token, err := dec.Token()
	if err != nil {
		return slog.Value{}, err
	}

	switch v := token.(type) {
	case json.Delim:
		switch v {
		case '{':
			return parseObject(dec)
		case '[':
			return parseArray(dec)
		}
	case bool:
		return slog.BoolValue(v), nil
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return slog.Int64Value(i), nil
		}
		if f, err := v.Float64(); err == nil {
			return slog.Float64Value(f), nil
		}
		return slog.StringValue(v.String()), nil
	case string:
		return slog.StringValue(v), nil
	case nil:
		return slog.AnyValue(nil), nil
	default:
		return Value{}, errors.New("unknown token value")
	}
	return Value{}, errors.New("tokenValue: unreachable code")
}

func parseObject(dec *json.Decoder) (Value, error) {
	var group []Attr
	for dec.More() {
		key, keyErr := parseKey(dec)
		if keyErr != nil {
			return Value{}, keyErr
		}

		val, valErr := parseValue(dec)
		if valErr != nil {
			return Value{}, valErr
		}

		group = append(group, Attr{Key: key, Value: val})
	}

	// closing bracket
	_, err := dec.Token()
	if err != nil {
		return Value{}, err
	}

	return slog.GroupValue(group...), nil
}

func parseArray(dec *json.Decoder) (Value, error) {
	var as []Attr
	var i int
	for dec.More() {
		v, err := parseValue(dec)
		if err != nil {
			return Value{}, err
		}

		as = append(as, Attr{Key: strconv.Itoa(i), Value: v})
		i++
	}

	// closing bracket
	_, err := dec.Token()
	if err != nil {
		return Value{}, err
	}

	return slog.GroupValue(as...), nil
}
