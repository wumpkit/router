---
title: "Buttons"
weight: 1
---

# Button Commands

Buttons let users click interactive elements on messages. Wumpkit Router allows both static and parameterized custom IDs.

## Defining a Button Command

Implement the `ButtonCommand` interface by embedding `router.ButtonCommandConfig` and defining `Run(ctx *router.ButtonCommandContext) error`:

```go
package main

import (
	"fmt"

	"github.com/wumpkit-go/router"
)

type ConfirmButtonCommand struct {
	router.ButtonCommandConfig
}

func NewConfirmButtonCommand() *ConfirmButtonCommand {
	return &ConfirmButtonCommand{
		ButtonCommandConfig: router.ButtonCommandConfig{
			CustomID: "btn_confirm",
		},
	}
}

func (c *ConfirmButtonCommand) Run(ctx *router.ButtonCommandContext) error {
	// reply privately to the user who clicked
	return ctx.ReplyEphemeral("Action confirmed!")
}
```

Register it on your router:

```go
r.MustRegisterButtonCommand(NewConfirmButtonCommand())
```

## Parameterized Route Matching

Instead of splitting custom IDs manually with strings, use route parameters like `{id}` or `{action}`:

```go
type TicketCloseButtonCommand struct {
	router.ButtonCommandConfig
}

func NewTicketCloseButtonCommand() *TicketCloseButtonCommand {
	return &TicketCloseButtonCommand{
		ButtonCommandConfig: router.ButtonCommandConfig{
			CustomID: "ticket:close:{ticket_id}",
		},
	}
}

func (c *TicketCloseButtonCommand) Run(ctx *router.ButtonCommandContext) error {
	// extract the route parameter directly
	ticketID := ctx.Param("ticket_id")

	// update the message in place
	return ctx.UpdateComponentMessage(router.ReplyOptions{
		Content: fmt.Sprintf("Ticket #%s has been marked as closed.", ticketID),
	})
}
```

When someone clicks a button with custom ID `ticket:close:1042`, the router matches the pattern and extracts `ticket_id = "1042"`.

## Updating Messages In-Place

When a button is clicked, you often want to edit the message the button belongs to (for example, disabling the button or changing text). Use `ctx.UpdateComponentMessage()`:

```go
func (c *CounterButtonCommand) Run(ctx *router.ButtonCommandContext) error {
	return ctx.UpdateComponentMessage(router.ReplyOptions{
		Content: "Counter updated!",
	})
}
```

If you need to do background work before updating the message, call `ctx.DeferUpdate()` first. This acknowledges the click without displaying a loading indicator to the user.
