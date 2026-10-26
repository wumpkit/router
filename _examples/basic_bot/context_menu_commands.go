package main

import (
	"fmt"
	"time"

	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/wumpkit"
)

// UserAvatarCommand allows users to right-click any user -> Apps -> "View Avatar"
// to view their avatar in full resolution.
type UserAvatarCommand struct {
	router.UserContextMenuCommandConfig
}

func newUserAvatarCommand() *UserAvatarCommand {
	return &UserAvatarCommand{
		UserContextMenuCommandConfig: router.UserContextMenuCommandConfig{
			Name: "View Avatar",
		},
	}
}

func (c *UserAvatarCommand) Run(ctx router.Context) error {
	var target *wumpkit.User
	if uCtx, ok := ctx.(*router.UserContextMenuCommandContext); ok {
		target = uCtx.TargetUser()
	}
	if target == nil {
		author := ctx.Author()
		target = &author
	}

	avatarURL := target.AvatarURL()

	embed := wumpkit.NewEmbedBuilder().
		SetTitle(fmt.Sprintf("%s's Avatar", target.DisplayName())).
		SetImage(avatarURL).
		SetColor(wumpkit.ColorSummerSky).
		SetTimestamp(time.Now()).
		Build()

	_, err := ctx.Reply("", router.ReplyOptions{
		Embeds: []wumpkit.Embed{embed},
		Flags:  wumpkit.MessageFlagEphemeral,
	})
	return err
}

// QuoteMessageCommand allows users to right-click any message -> Apps -> "Quote Message"
// to format and quote it into the channel.
type QuoteMessageCommand struct {
	router.MessageContextMenuCommandConfig
}

func newQuoteMessageCommand() *QuoteMessageCommand {
	return &QuoteMessageCommand{
		MessageContextMenuCommandConfig: router.MessageContextMenuCommandConfig{
			Name: "Quote Message",
		},
	}
}

func (c *QuoteMessageCommand) Run(ctx router.Context) error {
	var msg *wumpkit.Message
	if mCtx, ok := ctx.(*router.MessageContextMenuCommandContext); ok {
		msg = mCtx.TargetMessage()
	}
	if msg == nil {
		return fmt.Errorf("no target message found")
	}

	author := ctx.Author()
	embedBuilder := wumpkit.NewEmbedBuilder().
		SetAuthor(msg.Author.DisplayName(), "", msg.Author.AvatarURL()).
		SetDescription(msg.Content).
		SetColor(wumpkit.ColorDarkSlateGray).
		SetTimestamp(msg.ID.Timestamp()).
		SetFooter(fmt.Sprintf("Quoted by %s", author.DisplayName()), author.AvatarURL())

	if len(msg.Attachments) > 0 {
		embedBuilder.AddField(
			"Attachments",
			fmt.Sprintf("📎 %d attachment(s)", len(msg.Attachments)),
			false,
		)
	}

	_, err := ctx.Reply("", router.ReplyOptions{
		Embeds: []wumpkit.Embed{embedBuilder.Build()},
	})
	return err
}
