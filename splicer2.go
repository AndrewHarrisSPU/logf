package logf

import (
	"strings"

	"golang.org/x/exp/slog"
)

type dict map[string]slog.Value

func (d dict) prematch(k string) {
	d[k] = slog.StringValue(missingAttr)
}

func (d dict) match(a Attr) {
	if _, found := d[a.Key]; found {
		d[a.Key] = a.Value
	}
	if a.Value.Kind() == slog.GroupKind {
		d.matchRec([]string{}, a)
	}
}

func (d dict) insert(a Attr) {
	d[a.Key] = a.Value
}

func (d dict) clear() {
	for k := range d {
		delete(d, k)
	}
}

func (d dict) matchRec(keys []string, a Attr) {
	keys = append(keys, a.Key)

	for _, a := range a.Value.Group() {
		// recurse for GroupKind
		if a.Value.Kind() == slog.GroupKind {
			d.matchRec(keys, a)
		}

		// push key
		keys = append(keys, a.Key)

		key := strings.Join(keys, ".")
		if _, found := d[key]; found {
			d[key] = a.Value
		}

		// pop key
		keys = keys[:len(keys)-1]
	}
}
