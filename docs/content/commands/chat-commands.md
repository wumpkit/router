---
title: "Chat Commands"
weight: 1
---

# Chat Commands

Chat commands are hybrid commands. By default, they execute both as message prefix commands (`!kick`) and as Discord slash commands (`/kick`).

## Defining a Command

To create a command, declare a struct that embeds `router.ChatCommandConfig` and implements `Run(ctx router.Context) error`:

```go
package main

import (
	"fmt"

	"github.com/wumpkit/router"
	"github.com/wumpkit/wumpkit"
)

type KickCommand struct {
	router.ChatCommandConfig
}

func NewKickCommand() *KickCommand {
	return &KickCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:            "kick",
			Description:     "Kick a member from the server",
			Aliases:         []string{"boot"},
			UserPermissions: wumpkit.PermissionKickMembers,
			AppPermissions:  wumpkit.PermissionKickMembers,
			Args: []router.ArgDef{
				{
					Name:        "target",
					Description: "User to kick",
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

	if !ctx.InGuild() {
		return ctx.ReplyEphemeral("This command can only be used inside a server.")
	}

	guildID := *ctx.GuildID()
	err := ctx.Client().KickMember(ctx, guildID, targetID, reason)
	if err != nil {
		return ctx.ReplyEphemeral("Failed to kick user: " + err.Error())
	}

	return ctx.Reply(fmt.Sprintf("Kicked <@%s>. Reason: %s", targetID, reason), router.ReplyOptions{})
}
```

## Argument Types

The router automatically parses arguments for both prefix text and Discord slash command options.

Available `ArgType` values:

* `ArgTypeString`: Standard text input.
* `ArgTypeInt`: Whole numbers.
* `ArgTypeFloat`: Decimal numbers.
* `ArgTypeBool`: Boolean flags (`true`, `false`, `yes`, `no`, `1`, `0`).
* `ArgTypeUser`: User mentions or snowflake IDs.
* `ArgTypeMember`: Guild members.
* `ArgTypeRole`: Role mentions or role IDs.
* `ArgTypeChannel`: Channel mentions or channel IDs.
* `ArgTypeMentionable`: Either a user or a role.
* `ArgTypeAttachment`: Uploaded file attachments.

## Reading Arguments

Inside your `Run` method, access parsed arguments using `ctx.Args()`:

```go
// required arguments
target := ctx.Args().Snowflake("target")
count := ctx.Args().Int("count")
enabled := ctx.Args().Bool("enabled")

// optional arguments with default fallback values
reason := ctx.Args().StringOr("reason", "Default reason")
limit := ctx.Args().IntOr("limit", 10)
```

## Subcommands

To group commands under a parent command (such as `/tag create` and `/tag delete`), add subcommands to the `Subcommands` slice:

```go
type TagCommand struct {
	router.ChatCommandConfig
}

func NewTagCommand() *TagCommand {
	return &TagCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:        "tag",
			Description: "Tag management commands",
			Subcommands: []router.ChatCommand{
				NewTagGetCommand(),
				NewTagCreateCommand(),
				NewTagDeleteCommand(),
			},
		},
	}
}
```

When subcommands are present, Discord displays them as nested slash command groups, and prefix users run them as `!tag create`.

## Disabling Execution Surfaces

If you want a command to only work as a slash command or only as a prefix command, set `DisabledIn`:

```go
// only works as a slash command (/secret)
DisabledIn: router.DisabledInMessage

// only works as a text prefix command (!secret)
DisabledIn: router.DisabledInSlash
```
