---
title: "Context API"
weight: 2
---

# Context API

The `router.Context` interface is passed to every command handler. It provides environment information, user permissions, argument values, and reply helpers.

Because `router.Context` embeds Go standard `context.Context`, you can pass `ctx` directly to database queries, REST API calls, and external HTTP clients.

## Context Methods

### 1. User & Server Metadata

```go
// user who executed the command
user := ctx.Author()

// guild member information (nil if in direct messages)
member := ctx.AuthorMember()

// check if command was executed in a server
if ctx.InGuild() {
	guildID := *ctx.GuildID()
	slog.Info("server command", "guild_id", guildID)
}

// channel id where command was invoked
channelID := ctx.ChannelID()

// permissions of the executing user and the bot
userPerms := ctx.UserPermissions()
botPerms := ctx.AppPermissions()
```

### 2. Responding to Commands

The context provides reply methods that work uniformly across both message prefix commands and Discord interaction slash commands:

```go
// standard public reply
_, err := ctx.Reply("Operation complete.")

// ephemeral reply (visible only to the user who invoked the command)
_, err = ctx.ReplyEphemeral("This response is private.")

// reply with a rich embed
_, err = ctx.ReplyEmbed(embed)

// reply with an embed marked ephemeral
_, err = ctx.ReplyEmbed(embed, true)

// defer command response (shows loading state in Discord)
err = ctx.Defer(false)

// send a followup response after deferring
_, err = ctx.Followup("Long running task finished!")

// edit the bot response
_, err = ctx.Edit("Updated response text.")

// delete the bot response
err = ctx.Delete()
```

### 3. Middleware Key-Value Store

You can share data between middlewares and command handlers using `Set()` and `Get()`:

```go
// inside a middleware
ctx.Set("account_tier", "premium")

// inside your command handler
if tier, ok := ctx.Get("account_tier"); ok {
	slog.Info("executing command for tier", "tier", tier)
}
```

### 4. Client Access

To perform raw Discord REST operations not covered by the context helpers, access the underlying Wumpkit client:

```go
bot := ctx.Client()
_ = bot.AddGuildMemberRole(ctx, guildID, userID, roleID)
```
