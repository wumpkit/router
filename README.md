# Wumpkit Router

[![Go Reference](https://pkg.go.dev/badge/github.com/wumpkit/router.svg)](https://pkg.go.dev/github.com/wumpkit/router)
[![Release](https://img.shields.io/github/v/release/wumpkit/router?style=flat&color=3498db)](https://github.com/wumpkit/router/releases)
[![CI Test](https://github.com/wumpkit/router/actions/workflows/test.yml/badge.svg)](https://github.com/wumpkit/router/actions/workflows/test.yml)
[![CodeQL](https://github.com/wumpkit/router/actions/workflows/codeql.yml/badge.svg)](https://github.com/wumpkit/router/actions/workflows/codeql.yml)
[![License: BSD 3-Clause](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

A high-performance, idiomatic command and interaction routing engine for **[Wumpkit](https://github.com/wumpkit/wumpkit)**.

> Requires **Go 1.25+**

## Highlights

- **Type-Defined Commands**: Define commands as idiomatic Go types and structs. No builder patterns, no functional wrapper bloat.
- **Strongly Typed Contexts**: Each command interface accepts its concrete context directly (`*ButtonCommandContext`, `*SelectMenuCommandContext`, etc.) eliminating manual type assertions.
- **Zero-Allocation Routing Fast Paths**: Zero-alloc ASCII path segmentation and sub-microsecond component lookup.
- **Parameterized Component Routing**: Route buttons, menus, and modals with route parameters (`ticket:{id}`, `user:{action}:{id}`) and retrieve parameters via `ctx.Param("id")`.
- **Ergonomic Response API**: Clean helpers for replies, edits, followups, and ephemeral messages (`ctx.ReplyEmbed(...)`, `ctx.ReplyEphemeral(...)`, `ctx.Followup(...)`, `ctx.Edit(...)`, `ctx.Delete()`).
- **Custom Argument Parsers**: Define domain types with custom parsing logic on `ArgDef.Parser`.
- **Router Lifecycle Hooks**: Intercept execution before and after handlers via `r.Before(...)` and `r.After(...)` for tracing, metrics, and auditing.
- **Dynamic & Mention Prefixes**: Dynamic guild/user prefixes via `WithDynamicPrefixFunc(...)` alongside static prefixes and bot mention prefixes.
- **Pluggable Cooldowns**: Extensible rate limiting with the `CooldownStore` interface, featuring built-in GCRA algorithms and external store readiness (e.g., Redis).
- **Zero-Friction Command Sync**: Automatic synchronization with Discord's Application Command REST API on ready via `r.AutoSync(...)`.

## Installation

```bash
go get github.com/wumpkit/router
```

## Quick Start

### 1. Defining Chat and Slash Commands

Commands are pure Go structs implementing the `ChatCommand` interface:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wumpkit/router"
	"github.com/wumpkit/wumpkit"
)

type PingCommand struct {
	router.ChatCommandConfig
}

func NewPingCommand() *PingCommand {
	return &PingCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:        "ping",
			Description: "Check bot responsiveness",
		},
	}
}

func (c *PingCommand) Run(ctx router.Context) error {
	return ctx.ReplyEphemeral("Pong! 🏓")
}

func main() {
	client := wumpkit.New(
		context.Background(),
		wumpkit.WithToken(os.Getenv("DISCORD_TOKEN")),
	)

	r := router.NewWithClient(client, router.WithPrefix("!"))

	// register chat command
	r.MustRegisterCommand(NewPingCommand())

	// wire to gateway events
	r.Attach(client)

	// auto-sync slash commands when the bot connects
	r.AutoSync(wumpkit.Snowflake(123456789012345678))

	if err := client.StartGatewayWithSignals(); err != nil {
		panic(err)
	}
}
```

### 2. Parameterized Component Commands

Buttons, select menus, and modals receive their strongly typed context directly:

```go
type TicketButtonCommand struct {
	router.ButtonCommandConfig
}

func NewTicketButtonCommand() *TicketButtonCommand {
	return &TicketButtonCommand{
		ButtonCommandConfig: router.ButtonCommandConfig{
			CustomID: "ticket:{id}",
		},
	}
}

func (c *TicketButtonCommand) Run(ctx *router.ButtonCommandContext) error {
	ticketID := ctx.Param("id")
	return ctx.ReplyEphemeral(fmt.Sprintf("Opened ticket #%s", ticketID))
}
```

Register component commands procedurally:

```go
// register button command
r.MustRegisterButtonCommand(NewTicketButtonCommand())
```

### 3. Moderation Commands with Arguments and Permissions

```go
type KickCommand struct {
	router.ChatCommandConfig
}

func NewKickCommand() *KickCommand {
	return &KickCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:            "kick",
			Description:     "Kick a member from the server",
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

## Community and Support

- **Discussions**: [GitHub Discussions](https://github.com/wumpkit/router/discussions)
- **Bug Reports & Features**: [GitHub Issues](https://github.com/wumpkit/router/issues)
- **Security**: Please review our [Security Policy](.github/SECURITY.md) for private vulnerability disclosure.
- **Contributing**: Check [CONTRIBUTING.md](CONTRIBUTING.md) and [Code of Conduct](.github/CODE_OF_CONDUCT.md).

## License

This project is licensed under the [BSD 3-Clause License](LICENSE).
