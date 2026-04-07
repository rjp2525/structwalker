package structwalker

import (
	"reflect"
	"slices"
	"strings"
)

// Tag represents a single parsed struct tag value.
// For example, `scan:"email,redact,max=255"` parses to:
// Key="scan", Name="email", Flags=["redact"], Options={"max":"255"}.
type Tag struct {
	Key     string
	Name    string
	Raw     string
	Flags   []string
	Options map[string]string
}

// HasFlag reports whether the tag contains the given flag.
func (t Tag) HasFlag(flag string) bool {
	return slices.Contains(t.Flags, flag)
}

// Option returns the value for a key=value option and whether it exists.
func (t Tag) Option(key string) (string, bool) {
	v, ok := t.Options[key]
	return v, ok
}

// OptionOr returns the option value for key, or fallback if not present.
func (t Tag) OptionOr(key, fallback string) string {
	if v, ok := t.Options[key]; ok {
		return v
	}
	return fallback
}

// IsIgnored reports whether the tag name is "-", the conventional ignore marker.
func (t Tag) IsIgnored() bool {
	return t.Name == "-"
}

// ParseTag parses a raw tag value string "name,flag1,key=value" into a Tag.
func ParseTag(key, raw string) Tag {
	t := Tag{
		Key: key,
		Raw: raw,
	}

	if raw == "" {
		return t
	}

	parts := strings.Split(raw, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if i == 0 {
			t.Name = part
			continue
		}

		if k, v, ok := strings.Cut(part, "="); ok {
			if t.Options == nil {
				t.Options = make(map[string]string)
			}
			t.Options[k] = v
		} else {
			t.Flags = append(t.Flags, part)
		}
	}

	return t
}

// ParseAllTags extracts all tags from a reflect.StructTag.
func ParseAllTags(tag reflect.StructTag) []Tag {
	raw := string(tag)
	if raw == "" {
		return nil
	}

	var tags []Tag
	for raw != "" {
		raw = strings.TrimLeft(raw, " ")
		if raw == "" {
			break
		}

		i := 0
		for i < len(raw) && raw[i] != ':' && raw[i] != '"' && raw[i] != ' ' {
			i++
		}
		if i >= len(raw) || raw[i] != ':' {
			break
		}
		key := raw[:i]
		raw = raw[i+1:]

		if len(raw) == 0 || raw[0] != '"' {
			break
		}
		raw = raw[1:]

		j := 0
		for j < len(raw) && raw[j] != '"' {
			if raw[j] == '\\' {
				j++
			}
			j++
		}
		if j >= len(raw) {
			break
		}

		value := raw[:j]
		raw = raw[j+1:]

		tags = append(tags, ParseTag(key, value))
	}

	return tags
}
