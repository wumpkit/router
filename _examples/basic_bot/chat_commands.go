package main

import (
	"fmt"
	"time"

	"github.com/wumpkit/router"
	"github.com/wumpkit/wumpkit"
)

// PingCommand responds with the bot's current gateway latency and provides
// a button to refresh the latency reading.
type PingCommand struct {
	router.ChatCommandConfig
	client *wumpkit.Client
}

func newPingCommand(client *wumpkit.Client) *PingCommand {
	return &PingCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:        "ping",
			Description: "Check the bot's WebSocket gateway latency.",
			Aliases:     []string{"p", "latency"},
		},
		client: client,
	}
}

func (c *PingCommand) Run(ctx router.Context) error {
	latency := c.client.ShardManager().AverageLatency()

	embed := wumpkit.NewEmbedBuilder().
		SetTitle("🏓  Pong!").
		SetDescription(fmt.Sprintf("Gateway latency: **%v**", latency)).
		SetColor(wumpkit.ColorSummerSky).
		SetTimestamp(time.Now()).
		Build()

	refreshBtn := wumpkit.NewButtonBuilder().
		SetCustomID("ping/refresh").
		SetLabel("Refresh Latency").
		SetStyle(wumpkit.ButtonStylePrimary).
		Build()

	row := wumpkit.NewActionRowBuilder().
		SetComponent(refreshBtn).
		Build()

	_, err := ctx.Reply("", router.ReplyOptions{
		Embeds:     []wumpkit.Embed{embed},
		Components: []wumpkit.Component{row},
	})
	return err
}

// UserInfoCommand displays profile details about a target user or the author.
// Demonstrates optional arguments, user resolution, and named flags.
type UserInfoCommand struct {
	router.ChatCommandConfig
}

func newUserInfoCommand() *UserInfoCommand {
	return &UserInfoCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:        "userinfo",
			Description: "Display profile information for a user in the server.",
			Aliases:     []string{"u", "user", "whois"},
			Args: []router.ArgDef{
				{
					Name:        "target",
					Description: "The user or mention to inspect (defaults to yourself).",
					Type:        router.ArgTypeUser,
					Required:    false,
				},
			},
			Flags: []router.FlagDef{
				{
					Name:        "silent",
					Short:       "s",
					Description: "Send the response ephemerally (visible only to you).",
					Type:        router.FlagTypeBool,
				},
			},
		},
	}
}

func (c *UserInfoCommand) Run(ctx router.Context) error {
	target := ctx.Author()

	// If a user argument was specified, resolve it.
	if ctx.Args().Has("target") {
		targetID := ctx.Args().Snowflake("target")
		if targetID != 0 && ctx.Client() != nil {
			if u, err := ctx.Client().User(ctx, targetID); err == nil {
				target = u
			}
		}
	}

	botBadge := "No"
	if target.Bot {
		botBadge = "Yes (Bot)"
	}

	embed := wumpkit.NewEmbedBuilder().
		SetTitle(fmt.Sprintf("User Info: %s", target.DisplayName())).
		SetThumbnail(target.AvatarURL()).
		SetColor(wumpkit.ColorSummerSky).
		AddField("Username", target.Username, true).
		AddField("User ID", target.ID.String(), true).
		AddField("Bot Account", botBadge, true).
		AddField("Account Created", target.ID.Timestamp().Format(time.RFC1123), false).
		SetTimestamp(time.Now()).
		Build()

	var flags wumpkit.MessageFlags
	if ctx.Flags().Bool("silent") || ctx.Flags().Bool("s") {
		flags = wumpkit.MessageFlagEphemeral
	}

	_, err := ctx.Reply("", router.ReplyOptions{
		Embeds: []wumpkit.Embed{embed},
		Flags:  flags,
	})
	return err
}
