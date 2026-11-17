package structwalker

import (
	"iter"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Tag is one key of a struct tag, parsed.
//
// The value is a comma-separated list. The first item is the name, items
// with an equals sign are options and the rest are flags:
//
//	`validate:"email,required,max=255"`
//
// parses to Key "validate", Name "email", Flags ["required"] and Options
// {"max": "255"}. Wrap a name or an option value in single quotes to keep a
// comma inside it: `validate:"role,oneof='admin,editor'"`.
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

// Option returns the value of a key=value option and whether it was set.
func (t Tag) Option(key string) (string, bool) {
	v, ok := t.Options[key]
	return v, ok
}

// OptionOr returns the value of a key=value option. It returns fallback if
// the tag does not set it.
func (t Tag) OptionOr(key, fallback string) string {
	if v, ok := t.Options[key]; ok {
		return v
	}
	return fallback
}

// IsIgnored reports whether the tag is "-", the conventional way to opt a
// field out. Tag handlers are not called for an ignored tag.
func (t Tag) IsIgnored() bool {
	return t.Name == "-"
}

// ParseTag parses the value of a single tag key, such as the
// "name,omitempty" in `json:"name,omitempty"`.
func ParseTag(key, raw string) Tag {
	t := Tag{Key: key, Raw: raw}

	first := true
	for part := range tagParts(raw) {
		part = strings.TrimSpace(part)
		switch k, v, isOption := strings.Cut(part, "="); {
		case first:
			t.Name = unquote(part)
		case part == "":
		case isOption:
			if t.Options == nil {
				t.Options = make(map[string]string)
			}
			t.Options[strings.TrimSpace(k)] = unquote(v)
		default:
			t.Flags = append(t.Flags, unquote(part))
		}
		first = false
	}

	return t
}

// tagParts splits a tag value on commas, leaving alone any comma inside a
// single-quoted value. A quote only opens at the start of a name, flag or
// option value, so an apostrophe in the middle of a word is just a character.
func tagParts(raw string) iter.Seq[string] {
	return func(yield func(string) bool) {
		start, quoted := 0, false
		for i := range len(raw) {
			switch {
			case raw[i] == '\'' && quoted:
				quoted = false
			case raw[i] == '\'':
				before := strings.TrimRight(raw[start:i], " ")
				quoted = before == "" || strings.HasSuffix(before, "=")
			case raw[i] == ',' && !quoted:
				if !yield(raw[start:i]) {
					return
				}
				start = i + 1
			}
		}
		yield(raw[start:])
	}
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

// ParseAllTags parses every key in a struct tag, in the order they are
// written.
func ParseAllTags(tag reflect.StructTag) []Tag {
	return parseStructTag(tag, ParseTag)
}

// parseStructTag reads a struct tag the way reflect.StructTag.Lookup does,
// and stops at the first key that does not follow the convention.
func parseStructTag(tag reflect.StructTag, parse func(key, raw string) Tag) []Tag {
	if parse == nil {
		parse = ParseTag
	}

	var tags []Tag
	for rest := strings.TrimLeft(string(tag), " "); rest != ""; rest = strings.TrimLeft(rest, " ") {
		i := strings.IndexFunc(rest, func(r rune) bool {
			return r <= ' ' || r == ':' || r == '"' || r == 0x7f
		})
		if i <= 0 || i+1 >= len(rest) || rest[i] != ':' || rest[i+1] != '"' {
			break
		}
		key := rest[:i]
		rest = rest[i+1:]

		end := 1
		for end < len(rest) && rest[end] != '"' {
			if rest[end] == '\\' {
				end++
			}
			end++
		}
		if end >= len(rest) {
			break
		}

		value, err := strconv.Unquote(rest[:end+1])
		if err != nil {
			break
		}
		rest = rest[end+1:]

		tags = append(tags, parse(key, value))
	}
	return tags
}
