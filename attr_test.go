package logf

import (
	"bytes"
	"strings"
	"testing"
)

func TestStore(t *testing.T) {
	var b bytes.Buffer
	log := New().
		Writer(&b).
		Layout("attrs").
		Colors(false).
		ForceTTY(true).
		Logger()

	want := func(want string) {
		t.Helper()
		if !strings.Contains(b.String(), want) {
			t.Errorf("\n\texpected %s\n\tin %s", want, b.String())
		}
		b.Reset()
	}

	var store Store

	log.Info("", "store", store)
	want("store:<nil>")

	store = store.WithGroup("chordata")

	log.Info("", "store", store)
	want("store:<nil>")

	store = store.WithAttrs(Attrs("duck", 0, "goose", 1, "platypus", 2))

	log.Info("", "store", store)
	want("store:{chordata:{duck:0 goose:1 platypus:2}}")
}

func TestDecodeJSON(t *testing.T) {
	var b bytes.Buffer
	log := New().
		Writer(&b).
		Layout("attrs").
		Colors(false).
		ForceTTY(true).
		Logger()

	objects := []struct {
		label string
		json  string
		want  string
	}{
		{
			label: "an object",
			json:  `{"asd":"sdf", "dfg":"fgh"}`,
			want:  "object:{asd:sdf dfg:fgh}\n",
		},
		{
			label: "an array",
			json:  `[1, 2, 3, 4, 5]`,
			want:  "object:{0:1 1:2 2:3 3:4 4:5}\n",
		},
		{
			label: "a nil value",
			json:  `{"a":1, "b":null, "c":"three"}`,
			want:  "object:{a:1 b:<nil> c:three}\n",
		},
		{
			label: "stdlib example",
			json: `{
	"Message": "Hello",
	"Array": [1, 2, 3],
	"Null": null,
	"Number": 1.234,
	"Items":
		[
			{"id":0, "label":"foo"},
			{"id":1, "label":"bar"}
		]
}`,
			want: "object:{Message:Hello Array:{0:1 1:2 2:3} Null:<nil> Number:1.234 Items:{0:{id:0 label:foo} 1:{id:1 label:bar}}}\n",
		},
	}

	for _, obj := range objects {
		a, err := DecodeJSON(strings.NewReader(obj.json))
		if err != nil {
			log.Error("JSON", err)
			continue
		}
		log.Info("", "object", a)
		if obj.want != b.String() {
			t.Errorf("want %s, got %s", obj.want, b.String())
		}
		b.Reset()
	}
}
