---
title: structwalker
description: Walk any Go struct and act on its fields by tag. Defaults, validation, redaction and whatever else you invent, without writing the reflection loop again.
---

```go
type Config struct {
    Port    int           `default:"8080"`
    Timeout time.Duration `default:"30s"`
    DB      struct {
        Host string `default:"localhost"`
    }
}
```

```go
w := structwalker.New()

w.Handle("default", func(ctx structwalker.FieldContext) error {
    if ctx.IsZero() {
        return ctx.SetValue(ctx.Tag.Raw)
    }
    return nil
})

var cfg Config
err := w.Walk(&cfg)
// cfg.Port       == 8080
// cfg.Timeout    == 30 * time.Second
// cfg.DB.Host    == "localhost"
```
