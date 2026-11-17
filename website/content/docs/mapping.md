---
title: Mapping structs
description: Turn a struct into column lists, query arguments, CLI flags or a schema by reading its tags in a loop.
---

A lot of glue code is "for each field, take the name from a tag and do something with the value". SQL columns, CLI flags, form fields and JSON schemas are all that loop.

## Columns and values for an INSERT

```go
type Audit struct {
    CreatedBy string `db:"created_by"`
}

type Product struct {
    ID    int     `db:"id,pk"`
    Name  string  `db:"name"`
    Price float64 `db:"price"`
    Notes string  `db:"-"`
    Audit
}
```

```go
func Insert(table string, row any) (string, []any) {
    var columns []string
    var args []any

    for f := range structwalker.Fields(row) {
        tag, ok := f.Lookup("db")
        if !ok || tag.IsIgnored() || tag.HasFlag("pk") {
            continue
        }
        columns = append(columns, tag.Name)
        args = append(args, f.Value.Interface())
    }

    query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?%s)",
        table, strings.Join(columns, ", "), strings.Repeat(", ?", len(columns)-1))
    return query, args
}
```

```go
query, args := Insert("products", &Product{ID: 7, Name: "Lamp", Price: 39.5, Audit: Audit{CreatedBy: "reno"}})
// INSERT INTO products (name, price, created_by) VALUES (?, ?, ?)
// [Lamp 39.5 reno]
```

Things the loop got for free:

- The embedded `Audit` struct was walked, so `created_by` is there.
- `Notes` is skipped by `db:"-"` and `ID` by the `pk` flag.
- Fields without a `db` tag, including the `Audit` field itself, are skipped by the `!ok` check.

## Scanning rows back

The other direction is the same loop with `SetValue`:

```go
func Scan(row map[string]any, dest any) error {
    for f := range structwalker.Fields(dest) {
        tag, ok := f.Lookup("db")
        if !ok || tag.IsIgnored() {
            continue
        }
        if v, ok := row[tag.Name]; ok {
            if err := f.SetValue(v); err != nil {
                return err
            }
        }
    }
    return nil
}
```

`SetValue` takes care of the driver handing you an `int64` for an `int` column or a string for a `time.Duration`.

## CLI flags

Register a flag per tagged field and point it at the field itself:

```go
type Options struct {
    Addr    string `flag:"addr" help:"listen address"`
    Verbose bool   `flag:"verbose" help:"log more"`
}

func Bind(fs *flag.FlagSet, opts any) {
    for f := range structwalker.Fields(opts) {
        tag, ok := f.Lookup("flag")
        if !ok {
            continue
        }
        help, _ := f.Lookup("help")

        switch p := f.Value.Addr().Interface().(type) {
        case *string:
            fs.StringVar(p, tag.Name, *p, help.Raw)
        case *bool:
            fs.BoolVar(p, tag.Name, *p, help.Raw)
        }
    }
}
```

`f.Value.Addr()` gives a pointer to the real field, so parsing the flags writes straight into the struct. That needs `opts` to be a pointer.

## A schema

Walking a zero value is enough when you only care about the shape:

```go
for f := range structwalker.Fields(&Product{}) {
    tag, _ := f.Lookup("db")
    fmt.Printf("%-12s %-10s %s\n", f.Path, f.Field.Type, tag.Name)
}
```

## When to keep a walker

`structwalker.Fields(v)` with no options shares one cached walker, so calling it per row is fine. If you pass options, keep a walker in a package variable and use `w.Fields(v)` so the reflection work is done once per type.

The INSERT recipe runs as `Example_columnMapping` in the repository's tests.
