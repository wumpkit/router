# Wumpkit Router

[![Go Reference](https://pkg.go.dev/badge/github.com/wumpkit-go/router.svg)](https://pkg.go.dev/github.com/wumpkit-go/router)
[![Release](https://img.shields.io/github/v/release/wumpkit-go/router?style=flat&color=3498db)](https://github.com/wumpkit-go/router/releases)
[![CI Test](https://github.com/wumpkit-go/router/actions/workflows/test.yml/badge.svg)](https://github.com/wumpkit-go/router/actions/workflows/test.yml)
[![CodeQL](https://github.com/wumpkit-go/router/actions/workflows/codeql.yml/badge.svg)](https://github.com/wumpkit-go/router/actions/workflows/codeql.yml)
[![License: BSD 3-Clause](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

A high-performance, idiomatic command and interaction routing engine for **[Wumpkit](https://github.com/wumpkit-go/wumpkit)**.

## Highlights

- **Typed Commands & Functional Handlers**: Define commands cleanly using Go structs and types or inline functional handlers (`r.Command`, `r.OnButton`, `r.OnSelectMenu`, `r.OnModal`, `r.OnUserContextMenu`, `r.OnMessageContextMenu`). No builder pattern clutter.
- **Zero-Allocation Routing Fast Paths**: Zero-alloc ASCII path segmentation and sub-microsecond component lookup.
- **Parameterized Component Routing**: Route buttons, menus, and modals with route parameters (`ticket/:id`, `user/:action/:id`) and retrieve parsed parameters via `ctx.Params()`.
- **Typed Context Helpers**: Direct access to interaction payloads with `AsSlashContext()`, `AsButtonContext()`, `AsSelectMenuContext()`, `AsModalContext()`, and `AsMessageContext()`.
- **Ergonomic Response API**: Clean helpers for replies, edits, followups, and ephemeral messages (`ctx.ReplyEmbed(...)`, `ctx.ReplyEphemeral(...)`, `ctx.Followup(...)`, `ctx.Edit(...)`, `ctx.Delete()`).
- **Custom Argument Parsers**: Define domain types with custom argument parsing logic on `ArgDef.Parser`.
- **Router Lifecycle Hooks**: Intercept execution before and after handlers via `r.Before(...)` and `r.After(...)` for tracing, metrics, and auditing.
- **Dynamic & Mention Prefixes**: Dynamic guild/user prefixes via `WithDynamicPrefixFunc(...)` alongside static prefixes and bot mention prefixes.
- **Pluggable Cooldowns**: Extensible rate limiting with the `CooldownStore` interface, featuring built-in GCRA algorithms and external store readiness (e.g., Redis).
- **Zero-Friction Command Sync**: Automatic synchronization with Discord's Application Command REST API on ready via `r.AutoSync(...)`.

## Installation

```bash
go get github.com/wumpkit-go/router
```

## Quick Start

### 1. Functional Handlers

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/wumpkit"
)

func main() {
	client := wumpkit.New(
		context.Background(),
		wumpkit.WithToken(os.Getenv("DISCORD_TOKEN")),
	)

	r := router.NewWithClient(client, router.WithPrefix("!"))

	// chat command (works with both !ping and /ping)
	r.Command("ping", func(ctx router.Context) error {
		return ctx.ReplyEphemeral("Pong! 🏓")
	}, router.WithDescription("Check bot responsiveness"))

	// parameterized button handler
	r.OnButton("ticket/:id", func(ctx *router.ButtonInteractionContext) error {
		ticketID := ctx.Params().Get("id")
		return ctx.ReplyEphemeral(fmt.Sprintf("Opened ticket #%s", ticketID))
	})

	// wire to gateway events
	r.Attach(client)

	// auto-sync slash commands when the bot connects
	r.AutoSync(wumpkit.Snowflake(123456789012345678))

	if err := client.StartGatewayWithSignals(); err != nil {
		panic(err)
	}
}
```

### 2. Defining Commands as Structs

Commands can also be defined cleanly as Go structs:

```go
type KickCommand struct {
	router.ChatCommandConfig
}

func NewKickCommand() *KickCommand {
	return &KickCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:            "kick",
			Description:     "Kick a member from the server.",
			UserPermissions: wumpkit.PermissionKickMembers,
			AppPermissions:  wumpkit.PermissionKickMembers,
			Args: []router.ArgDef{
				{
					Name:        "target",
					Description: "Member to kick",
					Type:        router.ArgTypeUser,
					Required:    true,
				},
				{
					Name:        "reason",
					Description: "Reason for the kick",
					Type:        router.ArgTypeString,
					Required:    false,
				},
			},
		},
	}
}

func (c *KickCommand) Run(ctx router.Context) error {
	targetID := ctx.Args().Snowflake("target")
	reason := ctx.Args().StringOr("reason", "No reason provided")

	if err := ctx.Client().KickMember(ctx, *ctx.GuildID(), targetID, reason); err != nil {
		return ctx.ReplyEphemeral("Failed to kick member: " + err.Error())
	}

	return ctx.Reply(fmt.Sprintf("Kicked <@%s> (%s)", targetID, reason), router.ReplyOptions{})
}
```

Register the struct:

```go
r.MustRegisterCommand(NewKickCommand())
```

## Community and Support

- **Discussions**: [GitHub Discussions](https://github.com/wumpkit-go/router/discussions)
- **Bug Reports & Features**: [GitHub Issues](https://github.com/wumpkit-go/router/issues)
- **Security**: Please review our [Security Policy](.github/SECURITY.md) for private vulnerability disclosure.
- **Contributing**: Check [CONTRIBUTING.md](CONTRIBUTING.md) and [Code of Conduct](.github/CODE_OF_CONDUCT.md).

## License

This project is licensed under the [BSD 3-Clause License](LICENSE).
