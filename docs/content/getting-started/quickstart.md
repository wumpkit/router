---
title: "Quickstart"
weight: 2
---

# Quickstart

This guide shows you how to build a working bot that handles both prefix commands (`!ping`) and Discord slash commands (`/ping`) using Wumpkit Router.

## Complete Example

Create a `main.go` file:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/router/middleware"
	"github.com/wumpkit-go/wumpkit"
)

// 1. define the command struct
type PingCommand struct {
	router.ChatCommandConfig
}

func NewPingCommand() *PingCommand {
	return &PingCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:        "ping",
			Description: "Check bot latency and response",
			Aliases:     []string{"p", "latency"},
		},
	}
}

// 2. implement the Run method
func (c *PingCommand) Run(ctx router.Context) error {
	return ctx.ReplyEphemeral("Pong! The bot is online and working.")
}

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	appIDStr := os.Getenv("DISCORD_APP_ID")
	if token == "" || appIDStr == "" {
		slog.Error("DISCORD_TOKEN and DISCORD_APP_ID are required")
		os.Exit(1)
	}

	appIDNum, _ := strconv.ParseUint(appIDStr, 10, 64)
	appID := wumpkit.Snowflake(appIDNum)

	logger := slog.Default()

	// 3. create the wumpkit client
	bot := wumpkit.New(
		context.Background(),
		wumpkit.WithToken(token),
		wumpkit.WithGatewayConfig(wumpkit.ShardManagerConfig{
			Intents: wumpkit.GatewayIntentGuilds |
				wumpkit.GatewayIntentGuildMessages |
				wumpkit.GatewayIntentMessageContent,
		}),
		wumpkit.WithLogger(logger),
	)

	// 4. create the router
	r := router.New(
		router.WithClient(bot),
		router.WithPrefix("!"),
		router.WithLogger(logger),
	)

	// 5. add common middlewares
	r.Use(
		middleware.Recovery(logger),
		middleware.Logging(logger),
	)

	// 6. register your commands
	r.MustRegisterCommand(NewPingCommand())

	// 7. attach router to gateway events
	r.Attach(bot)

	// 8. auto-sync application commands with Discord on connect
	r.AutoSync(appID)

	// 9. start the gateway
	slog.Info("Starting bot...")
	if err := bot.StartGatewayWithSignals(); err != nil {
		slog.Error("bot stopped", "error", err)
	}
}
```

## How It Works

1. **Type-Defined Commands**:
   Your command embeds `router.ChatCommandConfig`. You do not need functional builder callbacks. Just set the fields in your constructor and implement `Run(ctx router.Context) error`.

2. **Dual Execution Surface**:
   By default, a `ChatCommand` responds to both text messages matching your prefix (e.g. `!ping` or `!p`) and Discord slash commands (`/ping`).

3. **AutoSync**:
   Calling `r.AutoSync(appID)` registers an `OnReady` hook on the client that automatically syncs all registered chat commands with Discord REST API when your bot connects.

4. **Clean Shutdown**:
   `bot.StartGatewayWithSignals()` waits for `SIGINT` (Ctrl+C) and shuts down gracefully without dropping active handlers.
