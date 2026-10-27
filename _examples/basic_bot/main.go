package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/wumpkit/router"
	"github.com/wumpkit/router/middleware"
	"github.com/wumpkit/wumpkit"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	appIDStr := os.Getenv("DISCORD_APP_ID")

	if token == "" || appIDStr == "" {
		slog.Error("Both DISCORD_TOKEN and DISCORD_APP_ID environment variables are required")
		os.Exit(1)
	}

	appIDNum, err := strconv.ParseUint(appIDStr, 10, 64)
	if err != nil {
		slog.Error("Invalid DISCORD_APP_ID", "error", err)
		os.Exit(1)
	}
	appID := wumpkit.Snowflake(appIDNum)

	// Optional: DISCORD_GUILD_ID syncs slash commands instantly to a test server
	// rather than waiting up to an hour for global Discord propagation.
	var devGuildID wumpkit.Snowflake
	if guildStr := os.Getenv("DISCORD_GUILD_ID"); guildStr != "" {
		if gNum, err := strconv.ParseUint(guildStr, 10, 64); err == nil {
			devGuildID = wumpkit.Snowflake(gNum)
		}
	}

	// 1. Structured Logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// 2. Initialize Wumpkit Bot Client
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

	// 3. Initialize Command Router
	r := router.New(
		router.WithClient(bot),
		router.WithPrefix("!"),
		router.WithLogger(logger),
	)

	// 4. Middlewares (Panic Recovery & Structured Execution Logging)
	r.Use(
		middleware.Recovery(logger),
		middleware.Logging(logger),
	)

	// 5. Register Chat Commands (support both prefix !command and /command)
	r.MustRegisterCommand(newPingCommand(bot))
	r.MustRegisterCommand(newUserInfoCommand())
	r.MustRegisterCommand(newMenuCommand())

	// 6. Register Context Menu Commands (Right Click -> Apps)
	r.MustRegisterUserContextMenuCommand(newUserAvatarCommand())
	r.MustRegisterMessageContextMenuCommand(newQuoteMessageCommand())

	// 7. Register Component Commands (Buttons, Select Menus, Prefix Matching)
	r.MustRegisterButtonCommand(newPingRefreshButton(bot))
	r.MustRegisterButtonCommand(newMenuAcceptButton())
	r.MustRegisterButtonCommand(newMenuDeclineButton())
	r.MustRegisterButtonCommand(newCounterButton())
	r.MustRegisterSelectMenuCommand(newMenuSelectColor())

	// 8. Wire Router to Bot Gateway Events (MessageCreate & InteractionCreate)
	r.AutoRegister()

	// 9. Sync Application Commands with Discord REST API on Ready
	bot.OnReady(func(ctx context.Context, event wumpkit.ReadyEvent) {
		logger.Info("Bot connected and ready",
			"username", event.User.Username,
			"guilds_count", len(event.Guilds),
		)

		if devGuildID != 0 {
			logger.Info("Syncing application commands to dev guild...", "guild_id", devGuildID.String())
			if err := r.SyncApplicationCommands(ctx, appID, devGuildID); err != nil {
				logger.Error("Failed syncing guild application commands", "error", err)
			}
		} else {
			logger.Info("Syncing global application commands...")
			if err := r.SyncApplicationCommands(ctx, appID); err != nil {
				logger.Error("Failed syncing global application commands", "error", err)
			}
		}
	})

	// 10. Start Gateway Connection (blocks until OS SIGINT/SIGTERM)
	logger.Info("Starting bot...")
	if err := bot.StartGatewayWithSignals(); err != nil {
		logger.Error("Bot encountered fatal error", "error", err)
		os.Exit(1)
	}
}
