package structwalker_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rjp2525/structwalker"
)

func ExampleFieldContext_SetValue() {
	type Config struct {
		Port     int           `default:"8080"`
		Timeout  time.Duration `default:"30s"`
		Started  time.Time     `default:"2026-04-07T00:00:00Z"`
		MaxConns *int          `default:"10"`
		Name     string
	}

	w := structwalker.New()
	w.Handle("default", func(ctx structwalker.FieldContext) error {
		if ctx.IsZero() {
			return ctx.SetValue(ctx.Tag.Raw)
		}
		return nil
	})

	cfg := Config{Port: 9000}
	if err := w.Walk(&cfg); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(cfg.Port, cfg.Timeout, cfg.Started.Year(), *cfg.MaxConns)
	// Output:
	// 9000 30s 2026 10
}

func ExampleFieldContext_Lookup() {
	type Signup struct {
		Email string `json:"email_address" validate:"required"`
	}

	w := structwalker.New()
	w.Handle("validate", func(ctx structwalker.FieldContext) error {
		if ctx.Tag.Name == "required" && ctx.IsZero() {
			name := ctx.Field.Name
			if tag, ok := ctx.Lookup("json"); ok {
				name = tag.Name
			}
			return fmt.Errorf("%s is required", name)
		}
		return nil
	})

	err := w.Walk(&Signup{})
	fmt.Println(errors.Unwrap(err))
	// Output:
	// email_address is required
}

func ExampleWalker_WalkContext() {
	type Order struct {
		CustomerID string `lookup:"customer"`
		Customer   string
	}

	type requestID struct{}

	w := structwalker.New()
	w.Handle("lookup", func(fc structwalker.FieldContext) error {
		fmt.Println("looking up", fc.Tag.Name, "for request", fc.Context().Value(requestID{}))
		return nil
	})

	ctx := context.WithValue(context.Background(), requestID{}, "req-42")
	if err := w.WalkContext(ctx, &Order{CustomerID: "c_1"}); err != nil {
		fmt.Println(err)
	}
	// Output:
	// looking up customer for request req-42
}

func ExampleTag() {
	tag := structwalker.ParseTag("validate", "role,required,oneof='admin,editor',max=20")

	fmt.Println(tag.Name)
	fmt.Println(tag.HasFlag("required"))
	fmt.Println(tag.Option("oneof"))
	fmt.Println(tag.OptionOr("min", "0"))
	// Output:
	// role
	// true
	// admin,editor true
	// 0
}
