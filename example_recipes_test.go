package structwalker_test

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rjp2525/structwalker"
)

// Load configuration from the environment, with a prefix per nested struct.
func Example_envConfig() {
	type Database struct {
		Host    string        `env:"HOST"`
		Port    int           `env:"PORT"`
		Timeout time.Duration `env:"TIMEOUT"`
	}
	type Config struct {
		Debug    bool     `env:"DEBUG"`
		Database Database `env:"DB,prefix"`
	}

	environ := map[string]string{
		"APP_DEBUG":      "true",
		"APP_DB_HOST":    "db.internal",
		"APP_DB_PORT":    "5432",
		"APP_DB_TIMEOUT": "2s",
	}

	var cfg Config
	prefixes := map[string]string{"": "APP_"}
	for f := range structwalker.Fields(&cfg) {
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

		if value, ok := environ[name]; ok {
			if err := f.SetValue(value); err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	fmt.Printf("%+v\n", cfg)
	// Output:
	// {Debug:true Database:{Host:db.internal Port:5432 Timeout:2s}}
}

// Validate a struct and report every problem at once, using json names.
func Example_validation() {
	type Address struct {
		City string `json:"city" validate:"required"`
		Zip  string `json:"zip" validate:"len=5"`
	}
	type Signup struct {
		Email   string  `json:"email" validate:"required"`
		Age     int     `json:"age" validate:"min=18,max=120"`
		Role    string  `json:"role" validate:"oneof='admin,editor,viewer'"`
		Address Address `json:"address"`
	}

	w := structwalker.New(
		structwalker.WithErrorMode(structwalker.CollectErrors),
		structwalker.WithTagParser(func(key, raw string) structwalker.Tag {
			if key == "validate" {
				raw = "," + raw // every item is a rule, none of them is a name
			}
			return structwalker.ParseTag(key, raw)
		}),
	)

	w.Handle("validate", func(ctx structwalker.FieldContext) error {
		rules, v := ctx.Tag, ctx.Value
		if rules.HasFlag("required") && ctx.IsZero() {
			return errors.New("is required")
		}
		if n, ok := intRule(rules, "min"); ok && v.Int() < n {
			return fmt.Errorf("must be at least %d", n)
		}
		if n, ok := intRule(rules, "max"); ok && v.Int() > n {
			return fmt.Errorf("must be at most %d", n)
		}
		if n, ok := intRule(rules, "len"); ok && int64(v.Len()) != n {
			return fmt.Errorf("must be %d characters", n)
		}
		if allowed, ok := rules.Option("oneof"); ok && !slices.Contains(strings.Split(allowed, ","), v.String()) {
			return fmt.Errorf("must be one of %s", allowed)
		}
		return nil
	})

	err := w.Walk(&Signup{Age: 16, Role: "owner", Address: Address{Zip: "123"}})

	if multi, ok := errors.AsType[*structwalker.MultiError](err); ok {
		for _, e := range multi.Errors {
			fmt.Printf("%s: %v\n", e.Path, e.Err)
		}
	}
	// Output:
	// Email: is required
	// Age: must be at least 18
	// Role: must be one of admin,editor,viewer
	// Address.City: is required
	// Address.Zip: must be 5 characters
}

func intRule(rules structwalker.Tag, name string) (int64, bool) {
	n, err := strconv.ParseInt(rules.OptionOr(name, ""), 10, 64)
	return n, err == nil
}

// Mask sensitive fields before a value is logged.
func Example_redaction() {
	type Card struct {
		Number string `log:"last4"`
		CVV    string `log:"-"`
	}
	type Customer struct {
		Name  string
		Email string `log:"redact"`
		Cards []Card
	}

	w := structwalker.New(structwalker.WithSliceElements(true))
	w.HandleAll(func(ctx structwalker.FieldContext) error {
		tag, _ := ctx.Lookup("log")
		switch {
		case tag.IsIgnored():
			return ctx.SetValue(nil)
		case tag.Name == "redact":
			return ctx.SetValue("[redacted]")
		case tag.Name == "last4":
			s := ctx.Value.String()
			keep := max(len(s)-4, 0)
			return ctx.SetValue(strings.Repeat("*", keep) + s[keep:])
		}
		return nil
	})

	customer := Customer{
		Name:  "Ada",
		Email: "ada@example.com",
		Cards: []Card{{Number: "4242424242424242", CVV: "123"}},
	}

	safe := customer
	safe.Cards = slices.Clone(customer.Cards)
	if err := w.Walk(&safe); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v\n", safe)
	fmt.Println(customer.Cards[0].CVV)
	// Output:
	// {Name:Ada Email:[redacted] Cards:[{Number:************4242 CVV:}]}
	// 123
}

// Turn a struct into the columns and values for an INSERT.
func Example_columnMapping() {
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

	p := Product{ID: 7, Name: "Lamp", Price: 39.5, Audit: Audit{CreatedBy: "reno"}}

	var columns []string
	var args []any
	for f := range structwalker.Fields(&p) {
		tag, ok := f.Lookup("db")
		if !ok || tag.IsIgnored() || tag.HasFlag("pk") {
			continue
		}
		columns = append(columns, tag.Name)
		args = append(args, f.Value.Interface())
	}

	fmt.Printf("INSERT INTO products (%s) VALUES (?%s)\n",
		strings.Join(columns, ", "), strings.Repeat(", ?", len(columns)-1))
	fmt.Println(args...)
	// Output:
	// INSERT INTO products (name, price, created_by) VALUES (?, ?, ?)
	// Lamp 39.5 reno
}

// List what changed between two values of the same type.
func Example_diff() {
	type Limits struct {
		Requests int
		Burst    int
	}
	type Plan struct {
		Name   string
		Price  int
		Limits Limits
	}

	before := Plan{Name: "Pro", Price: 20, Limits: Limits{Requests: 1000, Burst: 50}}
	after := Plan{Name: "Pro", Price: 25, Limits: Limits{Requests: 5000, Burst: 50}}

	old := map[string]any{}
	for f := range structwalker.Fields(&before) {
		if f.Value.Kind() != reflect.Struct {
			old[f.Path] = f.Value.Interface()
		}
	}

	for f := range structwalker.Fields(&after) {
		was, tracked := old[f.Path]
		if now := f.Value.Interface(); tracked && !reflect.DeepEqual(was, now) {
			fmt.Printf("%s: %v -> %v\n", f.Path, was, now)
		}
	}
	// Output:
	// Price: 20 -> 25
	// Limits.Requests: 1000 -> 5000
}
