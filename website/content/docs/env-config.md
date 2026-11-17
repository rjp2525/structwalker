---
title: Config from the environment
description: Load a nested config struct from environment variables with a Fields loop and a prefix per struct.
---

An `env` tag names the variable. A nested struct adds a prefix.

```go
type Database struct {
    Host    string        `env:"HOST"`
    Port    int           `env:"PORT"`
    Timeout time.Duration `env:"TIMEOUT"`
}

type Config struct {
    Debug    bool     `env:"DEBUG"`
    Database Database `env:"DB,prefix"`
}
```

With an `APP_` prefix that reads `APP_DEBUG`, `APP_DB_HOST`, `APP_DB_PORT` and `APP_DB_TIMEOUT`.

## The loop

This one doesn't need a walker. A [`Fields`](/docs/fields/) loop is enough.

```go
func Load(cfg any, prefix string) error {
    prefixes := map[string]string{"": prefix}

    for f := range structwalker.Fields(cfg) {
        tag, ok := f.Lookup("env")
        if !ok {
            continue
        }

        parent := ""
        if i := strings.LastIndex(f.Path, "."); i >= 0 {
            parent = f.Path[:i]
        }
        name := prefixes[parent] + tag.Name

        if tag.HasFlag("prefix") {
            prefixes[f.Path] = name + "_"
            continue
        }

        if value, ok := os.LookupEnv(name); ok {
            if err := f.SetValue(value); err != nil {
                return fmt.Errorf("%s: %w", name, err)
            }
        }
    }
    return nil
}
```

```go
var cfg Config
if err := Load(&cfg, "APP_"); err != nil {
    log.Fatal(err)
}
// {Debug:true Database:{Host:db.internal Port:5432 Timeout:2s}}
```

## How the prefixes work

A struct field is always visited before its children. When the loop reaches `Database` it records the prefix for everything under the path `Database`. By the time it reaches `Database.Host`, the prefix is waiting in the map.

That ordering is something you can rely on. It's the same in `Walk`.

## Errors name the variable

A value that doesn't parse stops the load with both names in the message:

```text
APP_DB_PORT: field Database.Port: cannot parse "abc" as int: invalid syntax
```

## Variations

- **Required variables.** Add a `required` flag and return an error in an `else` branch when `LookupEnv` finds nothing.
- **Defaults.** Read a second tag in the same loop: `f.Lookup("default")`. Or run the [defaults recipe](/docs/defaults/) first.
- **Lists.** `SetValue` doesn't split strings. For a `[]string` field, split the value yourself and call `f.SetValue(parts)`.
- **Secrets from files.** Check for `NAME_FILE` and read the file before calling `SetValue`.

This recipe runs as `Example_envConfig` in the repository's tests.
