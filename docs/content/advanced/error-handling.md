---
title: "Error Handling"
weight: 3
---

# Error Handling

Wumpkit Router handles errors using standard Go idioms. Errors are values, and error handlers are ordinary Go functions:

```go
type ErrorHandler func(ctx router.Context, err error)
```

By default, the router uses `router.DefaultErrorHandler`, which sends a formatted ephemeral embed describing what went wrong (such as missing arguments, missing permissions, or rate limits).

## Registering an Error Handler

You can set a custom error handler on the router using `r.OnError(...)` or via the `router.WithErrorHandler` option:

```go
r := router.New(
	router.WithClient(bot),
	router.WithErrorHandler(myErrorHandler),
)

// or configure it on an existing router instance
r.OnError(func(ctx router.Context, err error) {
	// inspect errors using standard errors.As or errors.Is
	var cmdErr *router.CommandError
	if errors.As(err, &cmdErr) {
		slog.Error("command failed", "error", cmdErr.Err)
		_ = ctx.ReplyEphemeral("An unexpected error occurred while running this command.")
		return
	}

	// fallback to default embed response for other errors
	router.DefaultErrorHandler(ctx, err)
})
```

## Concrete Error Types

When errors occur during command parsing or execution, the router passes strongly-typed error values to your error handler:

| Error Type | Description |
|---|---|
| `*router.MissingArgumentError` | A required command argument was not provided |
| `*router.InvalidArgumentError` | An argument could not be parsed into the expected type |
| `*router.UserPermissionMissingError` | The executing member does not have required permissions |
| `*router.AppPermissionMissingError` | The bot does not have required permissions in the channel |
| `*router.ContextExcludedError` | The command was run in an excluded channel type (such as direct messages) |
| `*router.CooldownError` | The command rate limit was exceeded |
| `*router.CommandError` | An unhandled error was returned from `Run(ctx)` |
| `router.ErrNSFWRequired` | The command requires an age-restricted channel |

## Inspecting Errors

Inspect errors using standard Go `errors.As` and `errors.Is`:

```go
r.OnError(func(ctx router.Context, err error) {
	var (
		missingArg *router.MissingArgumentError
		invalidArg *router.InvalidArgumentError
		userPerm   *router.UserPermissionMissingError
		appPerm    *router.AppPermissionMissingError
		cooldown   *router.CooldownError
		excluded   *router.ContextExcludedError
		cmdErr     *router.CommandError
	)

	switch {
	case errors.As(err, &missingArg):
		_ = ctx.ReplyEphemeral("Missing argument: " + missingArg.Name)

	case errors.As(err, &invalidArg):
		_ = ctx.ReplyEphemeral("Invalid value for " + invalidArg.Name + ", expected " + invalidArg.Expected)

	case errors.As(err, &userPerm):
		_ = ctx.ReplyEphemeral("You are missing permissions")

	case errors.As(err, &appPerm):
		_ = ctx.ReplyEphemeral("I need permissions to do that")

	case errors.As(err, &cooldown):
		_ = ctx.ReplyEphemeral("Please wait before using this command again")

	case errors.As(err, &excluded):
		_ = ctx.ReplyEphemeral("This command cannot be used in this context")

	case errors.Is(err, router.ErrNSFWRequired):
		_ = ctx.ReplyEphemeral("This command can only be used in NSFW channels")

	case errors.As(err, &cmdErr):
		slog.Error("internal command error", "error", cmdErr.Err)
		_ = ctx.ReplyEphemeral("Something went wrong internally")

	default:
		router.DefaultErrorHandler(ctx, err)
	}
})
```

## Error Wrapping and Unwrap

`*router.CommandError` and `*router.InvalidArgumentError` implement the standard library `Unwrap() error` interface.

If a middleware returns a `*router.CooldownError` from inside `next(ctx)`, the router wraps it in `*router.CommandError`. Calling `errors.As(err, &cooldown)` still matches correctly because `errors.As` traverses unwrapped errors.
