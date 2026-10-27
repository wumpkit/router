---
title: "Wumpkit Router Documentation"
---

# Wumpkit Router

Wumpkit Router is a command and interaction routing engine for Wumpkit bots.

Instead of writing long switch statements inside interaction handlers, you define commands as clean Go structs. The router matches commands, parses arguments, checks permissions, handles cooldowns, and routes button clicks or select menu choices.

## Documentation Sections

* [Getting Started](getting-started/): Installation and your first router bot.
* [Commands](commands/): Chat commands, arguments, subcommands, and context menus.
* [Components & Modals](components/): Button routing with route parameters, select menus, and modal dialogs.
* [Advanced Features](advanced/): Middlewares, lifecycle hooks, and the Context API.

## Quick Example

Here is how you define and register a simple command with the router:

```go
package main

import (
	"context"
	"os"

	"github.com/wumpkit/router"
	"github.com/wumpkit/wumpkit"
)

// 1. define your command struct
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

	// 2. create router and register commands
	r := router.NewWithClient(client, router.WithPrefix("!"))
	r.MustRegisterCommand(NewPingCommand())

	// 3. attach router to client gateway events
	r.Attach(client)

	// 4. start the bot
	if err := client.StartGatewayWithSignals(); err != nil {
		panic(err)
	}
}
```
