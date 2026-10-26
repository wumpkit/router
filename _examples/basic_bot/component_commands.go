package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/wumpkit"
)

// MenuCommand sends an interactive component showcase containing buttons
// and a select menu.
type MenuCommand struct {
	router.ChatCommandConfig
}

func newMenuCommand() *MenuCommand {
	return &MenuCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name:        "menu",
			Description: "Display an interactive message with buttons and select menus.",
			Aliases:     []string{"components", "buttons"},
		},
	}
}

func (c *MenuCommand) Run(ctx router.Context) error {
	// Row 1: Action Buttons
	btnAccept := wumpkit.NewButtonBuilder().
		SetCustomID("menu/accept").
		SetLabel("Accept Terms").
		SetStyle(wumpkit.ButtonStyleSuccess).
		Build()

	btnDecline := wumpkit.NewButtonBuilder().
		SetCustomID("menu/decline").
		SetLabel("Decline").
		SetStyle(wumpkit.ButtonStyleDanger).
		Build()

	btnCounter := wumpkit.NewButtonBuilder().
		SetCustomID("counter/1").
		SetLabel("Count: 1").
		SetStyle(wumpkit.ButtonStylePrimary).
		Build()

	btnLink := wumpkit.NewButtonBuilder().
		SetURL("https://github.com/wumpkit-go/wumpkit").
		SetLabel("Documentation").
		SetStyle(wumpkit.ButtonStyleLink).
		Build()

	buttonRow := wumpkit.NewActionRowBuilder().
		SetComponent(btnAccept, btnDecline, btnCounter, btnLink).
		Build()

	// Row 2: String Select Menu
	selectMenu := wumpkit.NewStringSelectMenuBuilder().
		SetCustomID("menu/select_color").
		SetPlaceholder("Choose your favorite accent color...").
		SetOptions([]wumpkit.SelectOptionStructure{
			{
				Label:       "Summer Sky",
				Value:       "summer_sky",
				Description: "Light bright blue",
			},
			{
				Label:       "Emerald",
				Value:       "emerald",
				Description: "Fresh vibrant green",
			},
			{
				Label:       "Coral Red",
				Value:       "coral_red",
				Description: "Warm vibrant red",
			},
		}).
		Build()

	selectRow := wumpkit.NewActionRowBuilder().
		SetComponent(selectMenu).
		Build()

	embed := wumpkit.NewEmbedBuilder().
		SetTitle("🎮  Interactive Components Showcase").
		SetDescription("Interact with the buttons or dropdown menu below to test component routing.").
		SetColor(wumpkit.ColorSummerSky).
		SetTimestamp(time.Now()).
		Build()

	_, err := ctx.Reply("", router.ReplyOptions{
		Embeds:     []wumpkit.Embed{embed},
		Components: []wumpkit.Component{buttonRow, selectRow},
	})
	return err
}

// PingRefreshButton handles the latency refresh button click.
type PingRefreshButton struct {
	router.ButtonCommandConfig
	client *wumpkit.Client
}

func newPingRefreshButton(client *wumpkit.Client) *PingRefreshButton {
	return &PingRefreshButton{
		ButtonCommandConfig: router.ButtonCommandConfig{
			CustomID: "ping/refresh",
		},
		client: client,
	}
}

func (b *PingRefreshButton) Run(ctx router.Context) error {
	latency := b.client.ShardManager().AverageLatency()

	embed := wumpkit.NewEmbedBuilder().
		SetTitle("🏓  Pong! (Refreshed)").
		SetDescription(fmt.Sprintf("Gateway latency: **%v**", latency)).
		SetColor(wumpkit.ColorMediumSeaGreen).
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

	if bCtx, ok := ctx.(*router.ButtonCommandContext); ok {
		return bCtx.UpdateComponentMessage(router.ReplyOptions{
			Embeds:     []wumpkit.Embed{embed},
			Components: []wumpkit.Component{row},
		})
	}
	_, err := ctx.Reply("", router.ReplyOptions{
		Embeds:     []wumpkit.Embed{embed},
		Components: []wumpkit.Component{row},
	})
	return err
}

// MenuAcceptButton handles the terms accept button click.
type MenuAcceptButton struct {
	router.ButtonCommandConfig
}

func newMenuAcceptButton() *MenuAcceptButton {
	return &MenuAcceptButton{
		ButtonCommandConfig: router.ButtonCommandConfig{
			CustomID: "menu/accept",
		},
	}
}

func (b *MenuAcceptButton) Run(ctx router.Context) error {
	author := ctx.Author()
	_, err := ctx.Reply(fmt.Sprintf("✅ Thank you %s, you accepted the terms!", author.DisplayName()), router.ReplyOptions{
		Flags: wumpkit.MessageFlagEphemeral,
	})
	return err
}

// MenuDeclineButton handles the terms decline button click.
type MenuDeclineButton struct {
	router.ButtonCommandConfig
}

func newMenuDeclineButton() *MenuDeclineButton {
	return &MenuDeclineButton{
		ButtonCommandConfig: router.ButtonCommandConfig{
			CustomID: "menu/decline",
		},
	}
}

func (b *MenuDeclineButton) Run(ctx router.Context) error {
	author := ctx.Author()
	_, err := ctx.Reply(fmt.Sprintf("❌ %s, you declined the terms.", author.DisplayName()), router.ReplyOptions{
		Flags: wumpkit.MessageFlagEphemeral,
	})
	return err
}

// CounterButton handles path parameter button clicks (e.g. counter/{count} -> counter/1 -> counter/2 ...).
type CounterButton struct {
	router.ButtonCommandConfig
}

func newCounterButton() *CounterButton {
	return &CounterButton{
		ButtonCommandConfig: router.ButtonCommandConfig{
			CustomID: "counter/{count}",
		},
	}
}

func (b *CounterButton) Run(ctx router.Context) error {
	bCtx, _ := ctx.(*router.ButtonCommandContext)
	rawCount := ""
	if bCtx != nil {
		rawCount = bCtx.Param("count")
	}
	current, _ := strconv.Atoi(rawCount)
	next := current + 1

	btnAccept := wumpkit.NewButtonBuilder().
		SetCustomID("menu/accept").
		SetLabel("Accept Terms").
		SetStyle(wumpkit.ButtonStyleSuccess).
		Build()

	btnDecline := wumpkit.NewButtonBuilder().
		SetCustomID("menu/decline").
		SetLabel("Decline").
		SetStyle(wumpkit.ButtonStyleDanger).
		Build()

	btnCounter := wumpkit.NewButtonBuilder().
		SetCustomID(fmt.Sprintf("counter/%d", next)).
		SetLabel(fmt.Sprintf("Count: %d", next)).
		SetStyle(wumpkit.ButtonStylePrimary).
		Build()

	btnLink := wumpkit.NewButtonBuilder().
		SetURL("https://github.com/wumpkit-go/wumpkit").
		SetLabel("Documentation").
		SetStyle(wumpkit.ButtonStyleLink).
		Build()

	buttonRow := wumpkit.NewActionRowBuilder().
		SetComponent(btnAccept, btnDecline, btnCounter, btnLink).
		Build()

	if bCtx != nil {
		return bCtx.UpdateComponentMessage(router.ReplyOptions{
			Components: []wumpkit.Component{buttonRow},
		})
	}
	return nil
}

// MenuSelectColor handles the color select menu dropdown submission.
type MenuSelectColor struct {
	router.SelectMenuCommandConfig
}

func newMenuSelectColor() *MenuSelectColor {
	return &MenuSelectColor{
		SelectMenuCommandConfig: router.SelectMenuCommandConfig{
			CustomID: "menu/select_color",
		},
	}
}

func (s *MenuSelectColor) Run(ctx router.Context) error {
	author := ctx.Author()
	var selected string
	if sCtx, ok := ctx.(*router.SelectMenuCommandContext); ok {
		selected = strings.Join(sCtx.Values(), ", ")
	}
	_, err := ctx.Reply(fmt.Sprintf("🎨 %s, you selected theme color: **%s**", author.DisplayName(), selected), router.ReplyOptions{
		Flags: wumpkit.MessageFlagEphemeral,
	})
	return err
}
