package structwalker_test

import (
	"fmt"
	"strings"

	"github.com/rjp2525/structwalker"
)

func ExampleWalker_basicUsage() {
	type User struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"email"`
		Age   int    `json:"age" validate:"min=0"`
	}

	w := structwalker.New()

	w.Handle("json", func(ctx structwalker.FieldContext) error {
		fmt.Printf("json field: %s -> %s\n", ctx.Path, ctx.Tag.Name)
		return nil
	})

	w.Handle("validate", func(ctx structwalker.FieldContext) error {
		fmt.Printf("validate: %s rule=%s\n", ctx.Path, ctx.Tag.Name)
		return nil
	})

	user := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
	if err := w.Walk(user); err != nil {
		fmt.Println("error:", err)
	}

	// Output:
	// json field: Name -> name
	// validate: Name rule=required
	// json field: Email -> email
	// validate: Email rule=email
	// json field: Age -> age
	// validate: Age rule=min=0
}

func ExampleWalker_sensitiveDataMasking() {
	type APIResponse struct {
		UserID    int    `json:"user_id"`
		Email     string `json:"email" mask:",redact"`
		SSN       string `json:"ssn" mask:",redact"`
		PublicKey string `json:"public_key"`
	}

	w := structwalker.New()
	w.Handle("mask", func(ctx structwalker.FieldContext) error {
		if ctx.Tag.HasFlag("redact") {
			return ctx.SetValue("***REDACTED***")
		}
		return nil
	})

	resp := &APIResponse{
		UserID:    123,
		Email:     "alice@example.com",
		SSN:       "123-45-6789",
		PublicKey: "ssh-rsa AAAA...",
	}
	if err := w.Walk(resp); err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println("Email:", resp.Email)
	fmt.Println("SSN:", resp.SSN)
	fmt.Println("PublicKey:", resp.PublicKey)

	// Output:
	// Email: ***REDACTED***
	// SSN: ***REDACTED***
	// PublicKey: ssh-rsa AAAA...
}

func ExampleWalker_nestedStructs() {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}
	type User struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	w := structwalker.New()
	w.Handle("json", func(ctx structwalker.FieldContext) error {
		fmt.Printf("[depth=%d] %s\n", ctx.Depth, ctx.Path)
		return nil
	})

	user := &User{Name: "Alice", Address: Address{Street: "123 Main St", City: "Springfield"}}
	if err := w.Walk(user); err != nil {
		fmt.Println("error:", err)
	}

	// Output:
	// [depth=0] Name
	// [depth=0] Address
	// [depth=1] Address.Street
	// [depth=1] Address.City
}

func ExampleWalker_handleAll() {
	type Config struct {
		Host string
		Port int
		Name string
	}

	var fields []string
	w := structwalker.New()
	w.HandleAll(func(ctx structwalker.FieldContext) error {
		fields = append(fields, ctx.Field.Name)
		return nil
	})

	if err := w.Walk(&Config{Host: "localhost", Port: 8080, Name: "app"}); err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(strings.Join(fields, ", "))

	// Output:
	// Host, Port, Name
}

func ExampleWalker_skipChildren() {
	type Secrets struct {
		APIKey string `json:"api_key"`
		Token  string `json:"token"`
	}
	type Config struct {
		Name    string  `json:"name"`
		Secrets Secrets `json:"secrets" audit:"skip"`
	}

	w := structwalker.New()
	w.Handle("json", func(ctx structwalker.FieldContext) error {
		fmt.Printf("visited: %s\n", ctx.Path)
		return nil
	})
	w.Handle("audit", func(ctx structwalker.FieldContext) error {
		if ctx.Tag.Name == "skip" {
			return structwalker.SkipChildren
		}
		return nil
	})

	if err := w.Walk(&Config{Name: "myapp", Secrets: Secrets{APIKey: "key", Token: "tok"}}); err != nil {
		fmt.Println("error:", err)
	}

	// Output:
	// visited: Name
	// visited: Secrets
}

func ExampleWalker_errorCollection() {
	type Form struct {
		Name  string `validate:"required"`
		Email string `validate:"required"`
		Age   int    `validate:"required"`
	}

	w := structwalker.New(structwalker.WithErrorMode(structwalker.CollectErrors))
	w.Handle("validate", func(ctx structwalker.FieldContext) error {
		if ctx.Tag.Name == "required" && ctx.IsZero() {
			return fmt.Errorf("field is required")
		}
		return nil
	})

	form := &Form{} // all zero values
	err := w.Walk(form)
	if err != nil {
		fmt.Println("Validation failed:")
		var multi *structwalker.MultiError
		if ok := err.(*structwalker.MultiError); ok != nil {
			multi = ok
			for _, e := range multi.Errors {
				fmt.Printf("  - %s: %s\n", e.Path, e.Err)
			}
		}
	}

	// Output:
	// Validation failed:
	//   - Name: field is required
	//   - Email: field is required
	//   - Age: field is required
}

func ExampleWalker_sliceWalking() {
	type Item struct {
		Name string `tag:"name"`
	}
	type Cart struct {
		Items []Item `tag:"items"`
	}

	w := structwalker.New(structwalker.WithSliceElements(true))
	w.Handle("tag", func(ctx structwalker.FieldContext) error {
		fmt.Printf("%s\n", ctx.Path)
		return nil
	})

	cart := &Cart{Items: []Item{{Name: "Widget"}, {Name: "Gadget"}}}
	if err := w.Walk(cart); err != nil {
		fmt.Println("error:", err)
	}

	// Output:
	// Items
	// Items[0].Name
	// Items[1].Name
}

func ExampleWithRecovery() {
	type S struct {
		Name string `tag:"name"`
	}

	w := structwalker.New()
	w.Use(structwalker.WithRecovery())
	w.Handle("tag", func(ctx structwalker.FieldContext) error {
		panic("something went wrong")
	})

	err := w.Walk(&S{Name: "test"})
	fmt.Println("recovered:", err != nil)

	// Output:
	// recovered: true
}
