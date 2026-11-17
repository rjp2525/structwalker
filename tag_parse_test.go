package structwalker

import (
	"maps"
	"reflect"
	"slices"
	"testing"
)

func TestParseTag_QuotedValues(t *testing.T) {
	tests := []struct {
		raw     string
		name    string
		flags   []string
		options map[string]string
	}{
		{raw: "role,oneof='admin,editor',required", name: "role", flags: []string{"required"}, options: map[string]string{"oneof": "admin,editor"}},
		{raw: "'a,b'", name: "a,b"},
		{raw: "name,'flag,with,commas'", name: "name", flags: []string{"flag,with,commas"}},
		{raw: "name,sep=','", name: "name", options: map[string]string{"sep": ","}},
		{raw: "name, max = 5 ,min=1", name: "name", options: map[string]string{"max": " 5", "min": "1"}},
		{raw: "name,msg=it's fine,required", name: "name", flags: []string{"required"}, options: map[string]string{"msg": "it's fine"}},
		{raw: "name,msg='unterminated,required", name: "name", options: map[string]string{"msg": "'unterminated,required"}},
		{raw: "name,empty=''", name: "name", options: map[string]string{"empty": ""}},
		{raw: "name,expr='a=b'", name: "name", options: map[string]string{"expr": "a=b"}},
		{raw: ",omitempty", name: "", flags: []string{"omitempty"}},
		{raw: "min=0", name: "min=0"},
		{raw: "name,,flag", name: "name", flags: []string{"flag"}},
	}

	for _, tt := range tests {
		tag := ParseTag("key", tt.raw)
		if tag.Name != tt.name || !slices.Equal(tag.Flags, tt.flags) || !maps.Equal(tag.Options, tt.options) {
			t.Errorf("ParseTag(%q) = name %q flags %q options %q", tt.raw, tag.Name, tag.Flags, tag.Options)
		}
		if tag.Raw != tt.raw {
			t.Errorf("ParseTag(%q).Raw = %q", tt.raw, tag.Raw)
		}
	}
}

func TestParseAllTags_UnescapesValues(t *testing.T) {
	tags := ParseAllTags(`desc:"say \"hi\"" path:"C:\\temp" next:"ok"`)

	want := map[string]string{"desc": `say "hi"`, "path": `C:\temp`, "next": "ok"}
	if len(tags) != len(want) {
		t.Fatalf("got %d tags, want %d", len(tags), len(want))
	}
	for _, tag := range tags {
		if tag.Raw != want[tag.Key] {
			t.Errorf("%s = %q, want %q", tag.Key, tag.Raw, want[tag.Key])
		}
	}
}

func TestParseAllTags_StopsAtMalformedInput(t *testing.T) {
	tests := map[string][]string{
		`json:"name" broken validate:"x"`:    {"json"},
		`json:"name" validate:"unterminated`: {"json"},
		`json:name`:                          nil,
		`:"no key"`:                          nil,
		`  json:"a"   db:"b"  `:              {"json", "db"},
		``:                                   nil,
	}

	for tag, want := range tests {
		var keys []string
		for _, parsed := range ParseAllTags(reflect.StructTag(tag)) {
			keys = append(keys, parsed.Key)
		}
		if !slices.Equal(keys, want) {
			t.Errorf("ParseAllTags(%q) keys = %v, want %v", tag, keys, want)
		}
	}
}

func FuzzParseAllTags(f *testing.F) {
	for _, seed := range []string{
		`json:"name,omitempty" validate:"required,max=100"`,
		`desc:"say \"hi\""`,
		`a:"1" a:"2"`,
		`json:"name" broken`,
		"key:\"\x00\"",
		`  spaced:"out"  `,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		tag := reflect.StructTag(raw)
		seen := map[string]bool{}
		for _, parsed := range ParseAllTags(tag) {
			if seen[parsed.Key] {
				continue
			}
			seen[parsed.Key] = true

			want, ok := tag.Lookup(parsed.Key)
			if !ok || want != parsed.Raw {
				t.Errorf("tag %q key %q: got %q, reflect says %q (found %v)", raw, parsed.Key, parsed.Raw, want, ok)
			}
		}
	})
}

func TestFieldContext_Lookup(t *testing.T) {
	type S struct {
		Email string `json:"email_address" validate:"required"`
	}

	var name string
	var missing bool
	w := New()
	w.Handle("validate", func(ctx FieldContext) error {
		tag, _ := ctx.Lookup("json")
		name = tag.Name
		_, found := ctx.Lookup("db")
		missing = !found
		return nil
	})

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if name != "email_address" || !missing {
		t.Errorf("name = %q, missing = %v", name, missing)
	}
}
