---
title: "Select Menus"
weight: 2
---

# Select Menus

Select menus display dropdown lists for users to choose one or more options.

## Defining a Select Menu Command

Implement the `SelectMenuCommand` interface by embedding `router.SelectMenuCommandConfig` and defining `Run(ctx *router.SelectMenuCommandContext) error`:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/wumpkit-go/router"
)

type RoleSelectMenuCommand struct {
	router.SelectMenuCommandConfig
}

func NewRoleSelectMenuCommand() *RoleSelectMenuCommand {
	return &RoleSelectMenuCommand{
		SelectMenuCommandConfig: router.SelectMenuCommandConfig{
			CustomID: "select_roles",
		},
	}
}

func (c *RoleSelectMenuCommand) Run(ctx *router.SelectMenuCommandContext) error {
	// values returns all options selected by the user
	selected := ctx.Values()
	if len(selected) == 0 {
		return ctx.ReplyEphemeral("No roles were selected.")
	}

	return ctx.ReplyEphemeral(fmt.Sprintf(
		"Assigned roles: %s",
		strings.Join(selected, ", "),
	))
}
```

Register it on your router:

```go
r.MustRegisterSelectMenuCommand(NewRoleSelectMenuCommand())
```

## Parameterized Select Menus

Like button commands, select menu custom IDs can include route variables:

```go
type GuildSettingsSelectMenuCommand struct {
	router.SelectMenuCommandConfig
}

func NewGuildSettingsSelectMenuCommand() *GuildSettingsSelectMenuCommand {
	return &GuildSettingsSelectMenuCommand{
		SelectMenuCommandConfig: router.SelectMenuCommandConfig{
			CustomID: "settings:{section}:select",
		},
	}
}

func (c *GuildSettingsSelectMenuCommand) Run(ctx *router.SelectMenuCommandContext) error {
	section := ctx.Param("section")
	choice := ctx.Values()[0]

	return ctx.UpdateComponentMessage(router.ReplyOptions{
		Content: fmt.Sprintf("Updated %s setting to: %s", section, choice),
	})
}
```
