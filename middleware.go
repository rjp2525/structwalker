package structwalker

import (
	"log/slog"
	"runtime/debug"
	"slices"
	"time"
)

// Middleware wraps a handler call. Call next to continue. Return without
// calling it to keep the handler from running.
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

// WithLogging returns middleware that logs each handler call at debug level
// and each handler error at error level.
func WithLogging(logger *slog.Logger) Middleware {
	return func(ctx FieldContext, next HandlerFunc) error {
		logger.Debug("structwalker: visiting field",
			"path", ctx.Path,
			"tag", ctx.Tag.Key,
			"depth", ctx.Depth,
		)
		err := next(ctx)
		if err != nil && !isFlowControl(err) {
			logger.Error("structwalker: handler error",
				"path", ctx.Path,
				"tag", ctx.Tag.Key,
				"error", err,
			)
		}
		return err
	}
}

// WithTiming returns middleware that reports how long each handler call took.
func WithTiming(fn func(path string, tagKey string, duration time.Duration)) Middleware {
	return func(ctx FieldContext, next HandlerFunc) error {
		start := time.Now()
		err := next(ctx)
		fn(ctx.Path, ctx.Tag.Key, time.Since(start))
		return err
	}
}

// WithRecovery returns middleware that turns a handler panic into a
// *PanicError carrying the panic value and the stack.
func WithRecovery() Middleware {
	return func(ctx FieldContext, next HandlerFunc) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = &PanicError{Value: r, Stack: debug.Stack()}
			}
		}()
		return next(ctx)
	}
}
