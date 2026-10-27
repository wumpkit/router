---
title: "Context Menu Commands"
weight: 2
---

# Context Menu Commands

Context menu commands appear when a user right-clicks a user or a message in Discord, hovers over **Apps**, and clicks the command.

Wumpkit Router supports both User and Message context menu commands with concrete typed contexts.

## 1. User Context Menu Commands

To inspect or interact with a user, implement `UserContextMenuCommand`:

```go
package main

import (
	"fmt"

	"github.com/wumpkit-go/router"
)

type UserProfileCommand struct {
	router.UserContextMenuCommandConfig
}

func NewUserProfileCommand() *UserProfileCommand {
	return &UserProfileCommand{
		UserContextMenuCommandConfig: router.UserContextMenuCommandConfig{
			Name: "User Profile",
		},
	}
}

func (c *UserProfileCommand) Run(ctx *router.UserContextMenuCommandContext) error {
	target := ctx.TargetUser()
	if target == nil {
		return ctx.ReplyEphemeral("Unable to find user information.")
	}

	return ctx.ReplyEphemeral(fmt.Sprintf(
		"User: %s (ID: %s)\nBot: %v",
		target.Username,
		target.ID,
		target.Bot,
	))
}
```

Register it on the router:

```go
r.MustRegisterUserContextMenuCommand(NewUserProfileCommand())
```

## 2. Message Context Menu Commands

To perform an action on a specific message (such as bookmarking, reporting, or translating), implement `MessageContextMenuCommand`:

```go
package main

import (
	"fmt"

	"github.com/wumpkit-go/router"
)

type QuoteMessageCommand struct {
	router.MessageContextMenuCommandConfig
}

func NewQuoteMessageCommand() *QuoteMessageCommand {
	return &QuoteMessageCommand{
		MessageContextMenuCommandConfig: router.MessageContextMenuCommandConfig{
			Name: "Quote Message",
		},
	}
}

func (c *QuoteMessageCommand) Run(ctx *router.MessageContextMenuCommandContext) error {
	msg := ctx.TargetMessage()
	if msg == nil {
		return ctx.ReplyEphemeral("Target message not found.")
	}

	return ctx.Reply(fmt.Sprintf(
		"> %s\n- Quoted from <@%s>",
		msg.Content,
		msg.Author.ID,
	), router.ReplyOptions{})
}
```

Register it on the router:

```go
r.MustRegisterMessageContextMenuCommand(NewQuoteMessageCommand())
```

When you call `r.AutoSync(appID)`, Wumpkit Router registers these context menu commands with Discord automatically.
