---
title: Installation
description: Add structwalker to a module, what it needs and how to check it works.
---

```bash
go get github.com/rjp2525/structwalker
```

That's the whole install. There are no other modules to pull in.

## Requirements

| | |
|---|---|
| Go | 1.26 or newer |
| Dependencies | none |
| Platforms | anywhere Go runs |

The package uses range-over-func iterators, `reflect.TypeFor` and a few other recent additions, which is where the Go version comes from.

## Check it works

```go
package main

import (
    "fmt"

    "github.com/rjp2525/structwalker"
)

type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    for f := range structwalker.Fields(&User{Name: "Ada"}) {
        tag, _ := f.Lookup("json")
        fmt.Printf("%s (%s) = %q\n", f.Path, tag.Name, f.Value)
    }
}
```

```text
Name (name) = "Ada"
Email (email) = ""
```

## Versioning

structwalker follows semantic versioning. Within v1 nothing you call will be removed or change its signature. Behaviour fixes are listed in the [changelog](https://github.com/rjp2525/structwalker/blob/main/CHANGELOG.md), with a note whenever one could affect you. Coming from v1.0? See [upgrading](/docs/upgrading/).

Next: the [quick start](/docs/quick-start/).
