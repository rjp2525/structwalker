package structwalker

import (
	"reflect"
	"testing"
)

func TestParseTag_Basic(t *testing.T) {
	tag := ParseTag("json", "name,omitempty")
	if tag.Key != "json" {
		t.Errorf("Key = %q, want %q", tag.Key, "json")
	}
	if tag.Name != "name" {
		t.Errorf("Name = %q, want %q", tag.Name, "name")
	}
	if !tag.HasFlag("omitempty") {
		t.Error("expected HasFlag(omitempty) = true")
	}
	if tag.HasFlag("required") {
		t.Error("expected HasFlag(required) = false")
	}
}

func TestParseTag_Options(t *testing.T) {
	tag := ParseTag("validate", "email,required,max=255,format=rfc5322")
	if tag.Name != "email" {
		t.Errorf("Name = %q, want %q", tag.Name, "email")
	}
	if !tag.HasFlag("required") {
		t.Error("expected HasFlag(required) = true")
	}
	if v, ok := tag.Option("max"); !ok || v != "255" {
		t.Errorf("Option(max) = %q, %v; want %q, true", v, ok, "255")
	}
	if v, ok := tag.Option("format"); !ok || v != "rfc5322" {
		t.Errorf("Option(format) = %q, %v; want %q, true", v, ok, "rfc5322")
	}
}

func TestParseTag_OptionOr(t *testing.T) {
	tag := ParseTag("db", "column_name,max=100")
	if v := tag.OptionOr("max", "50"); v != "100" {
		t.Errorf("OptionOr(max) = %q, want %q", v, "100")
	}
	if v := tag.OptionOr("min", "0"); v != "0" {
		t.Errorf("OptionOr(min) = %q, want %q", v, "0")
	}
}

func TestParseTag_Ignored(t *testing.T) {
	tag := ParseTag("json", "-")
	if !tag.IsIgnored() {
		t.Error("expected IsIgnored() = true")
	}
}

func TestParseTag_Empty(t *testing.T) {
	tag := ParseTag("json", "")
	if tag.Name != "" {
		t.Errorf("Name = %q, want empty", tag.Name)
	}
	if len(tag.Flags) != 0 {
		t.Errorf("Flags = %v, want empty", tag.Flags)
	}
	if len(tag.Options) != 0 {
		t.Errorf("Options = %v, want empty", tag.Options)
	}
}

func TestParseTag_NegationFlag(t *testing.T) {
	tag := ParseTag("validate", "field,!omitempty")
	if !tag.HasFlag("!omitempty") {
		t.Error("expected HasFlag(!omitempty) = true")
	}
	if tag.HasFlag("omitempty") {
		t.Error("expected HasFlag(omitempty) = false for negated flag")
	}
}

func TestParseTag_Raw(t *testing.T) {
	raw := "email,required,max=255"
	tag := ParseTag("validate", raw)
	if tag.Raw != raw {
		t.Errorf("Raw = %q, want %q", tag.Raw, raw)
	}
}

func TestParseAllTags(t *testing.T) {
	type testStruct struct {
		Name string `json:"name,omitempty" validate:"required,max=100"`
	}

	sf, _ := reflect.TypeFor[testStruct]().FieldByName("Name")
	tags := ParseAllTags(sf.Tag)

	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	jsonTag := tags[0]
	if jsonTag.Key != "json" || jsonTag.Name != "name" {
		t.Errorf("json tag = %+v", jsonTag)
	}

	validateTag := tags[1]
	if validateTag.Key != "validate" || validateTag.Name != "required" {
		t.Errorf("validate tag = %+v", validateTag)
	}
}

func TestParseAllTags_Empty(t *testing.T) {
	tags := ParseAllTags("")
	if tags != nil {
		t.Errorf("expected nil, got %v", tags)
	}
}

func TestParseAllTags_SingleTag(t *testing.T) {
	type testStruct struct {
		ID int `db:"id"`
	}

	sf, _ := reflect.TypeFor[testStruct]().FieldByName("ID")
	tags := ParseAllTags(sf.Tag)

	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1", len(tags))
	}
	if tags[0].Key != "db" || tags[0].Name != "id" {
		t.Errorf("tag = %+v", tags[0])
	}
}

func TestParseTag_MultipleOptions(t *testing.T) {
	tag := ParseTag("ch", "column_name,type=String,codec=ZSTD,ttl=30d")
	if tag.Name != "column_name" {
		t.Errorf("Name = %q, want %q", tag.Name, "column_name")
	}
	if v, _ := tag.Option("type"); v != "String" {
		t.Errorf("Option(type) = %q, want %q", v, "String")
	}
	if v, _ := tag.Option("codec"); v != "ZSTD" {
		t.Errorf("Option(codec) = %q, want %q", v, "ZSTD")
	}
	if v, _ := tag.Option("ttl"); v != "30d" {
		t.Errorf("Option(ttl) = %q, want %q", v, "30d")
	}
}

func TestParseTag_OnlyName(t *testing.T) {
	tag := ParseTag("json", "field_name")
	if tag.Name != "field_name" {
		t.Errorf("Name = %q, want %q", tag.Name, "field_name")
	}
	if len(tag.Flags) != 0 {
		t.Errorf("Flags = %v, want empty", tag.Flags)
	}
}
