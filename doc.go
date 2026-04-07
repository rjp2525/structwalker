// Package structwalker provides recursive struct walking with tag-based handler dispatch.
//
// Register handlers for struct tag keys, then walk any struct. Each field's tags are
// parsed, matched to registered handlers, and dispatched with full context including
// the parsed tag, field value, dot-separated path, and parent struct.
//
// Reflection metadata is cached per-type so repeated walks skip parsing entirely.
//
// Basic usage:
//
//	w := structwalker.New()
//	w.Handle("json", func(ctx structwalker.FieldContext) error {
//	    fmt.Println(ctx.Path, ctx.Tag.Name)
//	    return nil
//	})
//	err := w.Walk(&myStruct)
package structwalker
