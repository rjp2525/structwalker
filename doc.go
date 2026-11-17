// Package structwalker walks the fields of a struct and lets you act on them
// by tag.
//
// It is the reflection loop behind every config loader, validator and
// redactor, written once. You say what a tag means. structwalker finds the
// fields, parses the tags, follows nested structs and pointers, keeps track of
// the path and caches the reflection work per type.
//
// # Handlers
//
// Register a handler for a tag key and walk any struct:
//
//	w := structwalker.New()
//	w.Handle("default", func(ctx structwalker.FieldContext) error {
//	    if ctx.IsZero() {
//	        return ctx.SetValue(ctx.Tag.Raw)
//	    }
//	    return nil
//	})
//
//	type Config struct {
//	    Port    int           `default:"8080"`
//	    Timeout time.Duration `default:"30s"`
//	}
//
//	var cfg Config
//	err := w.Walk(&cfg) // cfg.Port == 8080, cfg.Timeout == 30*time.Second
//
// A handler receives a [FieldContext]. It has the parsed [Tag], the field's
// value and its path. [FieldContext.SetValue] writes back to the field.
//
// Several handlers can share a walk. [WithPriority] orders them, [WithFilter] narrows them and
// [WithPhase] runs them after a field's children instead of before.
// [Walker.HandleAll] registers a handler for every field, tagged or not.
//
// # A plain loop
//
// For a one-off there is nothing to register. [Fields] is an iterator:
//
//	for f := range structwalker.Fields(&cfg) {
//	    if tag, ok := f.Lookup("env"); ok {
//	        if err := f.SetValue(os.Getenv(tag.Name)); err != nil {
//	            return err
//	        }
//	    }
//	}
//
// # Steering the walk
//
// A handler returns [ErrSkipField] to leave a field alone or [ErrSkipChildren]
// to stop the walker descending into it. Any other error ends the walk as a
// [*WalkError] that carries the field's path. [WithErrorMode] changes that:
// [CollectErrors] gathers every error into a [*MultiError], which is what a
// validator wants.
//
// By default the walker follows struct fields and pointers.
// [WithSliceElements], [WithMapEntries] and [WithInterfaces] extend it into
// containers. Cycles are detected.
//
// # Middleware
//
// [Walker.Use] wraps every handler call. [WithRecovery], [WithLogging] and
// [WithTiming] are included.
//
// # Concurrency
//
// Build a [Walker] once and share it. Walks can run concurrently and handlers
// can be registered while they do.
package structwalker
