package structwalker

import (
	"fmt"
	"log/slog"
	"slices"
	"time"
)

// Middleware wraps handler execution for cross-cutting concerns.
type Middleware func(ctx FieldContext, next HandlerFunc) error

// Use adds middleware around every handler, including handlers registered
// before the call. The first middleware added is the outermost.
func (w *Walker) Use(mw ...Middleware) {
	w.update(func(r *registry) *registry { return r.withMiddleware(mw) })
}

func chain(fn HandlerFunc, middleware []Middleware) HandlerFunc {
	for _, mw := range slices.Backward(middleware) {
		next := fn
		fn = func(ctx FieldContext) error { return mw(ctx, next) }
	}
	return fn
}

// WithLogging returns middleware that logs field visits at debug level.
func WithLogging(logger *slog.Logger) Middleware {
	return func(ctx FieldContext, next HandlerFunc) error {
		logger.Debug("structwalker: visiting field",
			"path", ctx.Path,
			"tag", ctx.Tag.Key,
			"depth", ctx.Depth,
		)
		err := next(ctx)
		if err != nil {
			logger.Error("structwalker: handler error",
				"path", ctx.Path,
				"tag", ctx.Tag.Key,
				"error", err,
			)
		}
		return err
	}
}

// WithTiming returns middleware that records handler execution duration.
func WithTiming(fn func(path string, tagKey string, duration time.Duration)) Middleware {
	return func(ctx FieldContext, next HandlerFunc) error {
		start := time.Now()
		err := next(ctx)
		fn(ctx.Path, ctx.Tag.Key, time.Since(start))
		return err
	}
}

// WithRecovery returns middleware that converts handler panics into errors.
func WithRecovery() Middleware {
	return func(ctx FieldContext, next HandlerFunc) (retErr error) {
		defer func() {
			if r := recover(); r != nil {
				retErr = fmt.Errorf("panic in handler for %s (tag %q): %v", ctx.Path, ctx.Tag.Key, r)
			}
		}()
		return next(ctx)
	}
}
