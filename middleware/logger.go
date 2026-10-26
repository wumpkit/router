package middleware

import (
	"time"

	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/wumpkit"
)

// Logging returns a middleware that logs command executions and duration with structured attributes.
func Logging(logger wumpkit.Logger) router.Middleware {
	return func(next router.CommandHandler) router.CommandHandler {
		return func(ctx router.Context) error {
			start := time.Now()
			err := next(ctx)
			elapsed := time.Since(start)

			if logger == nil {
				return err
			}

			user := ctx.Author()
			cmdName := "unknown"
			if cmd := ctx.Command(); cmd != nil {
				cmdName = router.ExtractCommonConfig(cmd).Name
			} else if msg := ctx.Message(); msg != nil {
				cmdName = msg.Content
			}

			var guildIDStr string
			if gID := ctx.GuildID(); gID != nil {
				guildIDStr = gID.String()
			}

			attrs := []any{
				"command", cmdName,
				"trigger", detectTrigger(ctx),
				"user_id", user.ID.String(),
				"username", user.Username,
				"guild_id", guildIDStr,
				"channel_id", ctx.ChannelID().String(),
				"duration_ms", elapsed.Milliseconds(),
			}

			if err != nil {
				attrs = append(attrs, "error", err.Error())
				logger.Error("command execution failed", attrs...)
			} else {
				logger.Info("command executed successfully", attrs...)
			}

			return err
		}
	}
}

// Logger is an alias for Logging.
func Logger(logger wumpkit.Logger) router.Middleware {
	return Logging(logger)
}

func detectTrigger(ctx router.Context) string {
	switch ev := ctx.Event().(type) {
	case wumpkit.MessageCreateEvent, *wumpkit.MessageCreateEvent, *wumpkit.Message:
		return "prefix"
	case wumpkit.InteractionCreateEvent:
		return interactionTrigger(ev.Interaction)
	case *wumpkit.InteractionCreateEvent:
		return interactionTrigger(ev.Interaction)
	default:
		if in := ctx.Interaction(); in != nil {
			return interactionTrigger(in)
		}
		return "unknown"
	}
}

func interactionTrigger(in wumpkit.Interaction) string {
	switch in.(type) {
	case *wumpkit.ChatInputCommandInteraction:
		return "slash"
	case *wumpkit.UserContextMenuCommandInteraction:
		return "user_command"
	case *wumpkit.MessageContextMenuCommandInteraction:
		return "message_command"
	case *wumpkit.ComponentInteraction:
		return "component"
	case *wumpkit.ModalSubmitInteraction:
		return "modal"
	case *wumpkit.AutoCompleteInteraction:
		return "autocomplete"
	default:
		return "interaction"
	}
}
