---
title: "Middlewares & Hooks"
weight: 1
---

# Middlewares & Hooks

Middlewares let you intercept command execution to add cross-cutting behavior like crash recovery, execution logging, metrics, or authentication checks.

## Built-In Middlewares

The router package includes built-in middlewares under `github.com/wumpkit/router/middleware`.

### 1. Panic Recovery

Catches unexpected panics inside command handlers and converts them into errors instead of crashing your bot process:

```go
import "github.com/wumpkit/router/middleware"

r.Use(middleware.Recovery(logger))
```

### 2. Execution Logging

Logs when commands start and finish, tracking command name, executing user, guild ID, execution duration, and errors:

```go
r.Use(middleware.Logging(logger))
```

## Adding Middlewares to the Router

Attach middlewares using `r.Use()`:

```go
r := router.New(
	router.WithClient(bot),
	router.WithPrefix("!"),
)

r.Use(
	middleware.Recovery(logger),
	middleware.Logging(logger),
	MyCustomMiddleware(),
)
```

Middlewares execute in the exact order they are passed to `r.Use()`.

## Writing a Custom Middleware

A middleware is a function that takes a `router.CommandHandler` and returns a wrapped `router.CommandHandler`:

```go
package main

import (
	"log/slog"
	"time"

	"github.com/wumpkit/router"
)

func PerformanceTimer() router.Middleware {
	return func(next router.CommandHandler) router.CommandHandler {
		return func(ctx router.Context) error {
			start := time.Now()

			// call next handler in the chain
			err := next(ctx)

			duration := time.Since(start)
			if duration > 500*time.Millisecond {
				slog.Warn("slow command detected",
					"duration", duration,
				)
			}

			return err
		}
	}
}
```

## Lifecycle Hooks

If you need lightweight listeners before or after commands without wrapping the execution chain, use `r.Before()` and `r.After()`:

```go
// runs before any command executes, return error to stop execution
r.Before(func(ctx router.Context) error {
	// example: verify if command is restricted
	return nil
})

// runs after command completes, receiving any error that occurred
r.After(func(ctx router.Context, err error) {
	if err != nil {
		slog.Error("command failed", "error", err)
	}
})
```
