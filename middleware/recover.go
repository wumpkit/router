package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/wumpkit"
)

// Recovery returns a middleware that catches runtime panics, logs them, and forwards them as errors.
func Recovery(logger wumpkit.Logger) router.Middleware {
	return func(next router.CommandHandler) router.CommandHandler {
		return func(ctx router.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := string(debug.Stack())
					err = fmt.Errorf("panic in command execution: %v\n%s", r, stack)
					if logger != nil {
						logger.Error("panic recovered during command execution",
							"panic", fmt.Sprintf("%v", r),
							"user_id", ctx.Author().ID.String(),
							"channel_id", ctx.ChannelID().String(),
							"stack", stack,
						)
					}
				}
			}()
			return next(ctx)
		}
	}
}

// Recover is an alias for Recovery.
func Recover(logger wumpkit.Logger) router.Middleware {
	return Recovery(logger)
}
