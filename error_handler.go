package router

import (
	"fmt"
	"strings"
	"time"

	"github.com/wumpkit-go/wumpkit"
)

// CommandErrorHandler handles command lifecycle errors and rejections.
type CommandErrorHandler interface {
	// OnMissingArgument handles missing required positional arguments.
	OnMissingArgument(ctx Context, name string, expected string)
	// OnInvalidArgument handles arguments or flags that failed type conversion.
	OnInvalidArgument(ctx Context, name string, expected string, received string)
	// OnUserPermissionMissing handles missing user permissions.
	OnUserPermissionMissing(ctx Context, missing wumpkit.Permissions)
	// OnAppPermissionMissing handles missing bot/application permissions.
	OnAppPermissionMissing(ctx Context, missing wumpkit.Permissions)
	// OnCooldown handles rate limit / cooldown violations.
	OnCooldown(ctx Context, remaining time.Duration, scope CooldownScope)
	// OnContextExcluded handles execution in an excluded channel or context.
	OnContextExcluded(ctx Context, excluded ExcludedIn)
	// OnNSFWRequired handles commands executed outside NSFW channels.
	OnNSFWRequired(ctx Context)
	// OnCommandError handles unhandled runtime errors returned by commands.
	OnCommandError(ctx Context, cmd any, err error)
}

// DefaultCommandErrorHandler is the standard error handler for commands that responds with formatted red embeds.
type DefaultCommandErrorHandler struct{}

// NewDefaultCommandErrorHandler creates a new DefaultCommandErrorHandler instance.
func NewDefaultCommandErrorHandler() *DefaultCommandErrorHandler {
	return &DefaultCommandErrorHandler{}
}

// OnMissingArgument responds with missing argument error description.
func (h *DefaultCommandErrorHandler) OnMissingArgument(ctx Context, name string, expected string) {
	desc := fmt.Sprintf("Missing required argument: `<%s>` (%s)", name, expected)
	h.replyError(ctx, desc)
}

// OnInvalidArgument responds with invalid argument format description.
func (h *DefaultCommandErrorHandler) OnInvalidArgument(ctx Context, name string, expected string, received string) {
	desc := fmt.Sprintf("`%s` is not a valid %s for `%s`", received, expected, name)
	h.replyError(ctx, desc)
}

// OnUserPermissionMissing responds with missing user permissions description.
func (h *DefaultCommandErrorHandler) OnUserPermissionMissing(ctx Context, missing wumpkit.Permissions) {
	desc := fmt.Sprintf("You are missing the required permission(s): %s", formatPermissions(missing))
	h.replyError(ctx, desc)
}

// OnAppPermissionMissing responds with missing bot permissions description.
func (h *DefaultCommandErrorHandler) OnAppPermissionMissing(ctx Context, missing wumpkit.Permissions) {
	desc := fmt.Sprintf("The bot is missing the required permission(s): %s", formatPermissions(missing))
	h.replyError(ctx, desc)
}

// OnCooldown responds with cooldown rate limit violation description.
func (h *DefaultCommandErrorHandler) OnCooldown(ctx Context, remaining time.Duration, scope CooldownScope) {
	var target string
	switch scope {
	case CooldownScopeGuild:
		target = "This server is"
	case CooldownScopeChannel:
		target = "This channel is"
	case CooldownScopeMember:
		target = "You are in this server"
	case CooldownScopeGlobal:
		target = "This command is globally"
	default:
		target = "You are"
	}
	desc := fmt.Sprintf("%s on cooldown! Please wait `%.1fs` before using this command again", target, remaining.Seconds())
	h.replyError(ctx, desc)
}

// OnContextExcluded responds when a command is executed in an excluded context.
func (h *DefaultCommandErrorHandler) OnContextExcluded(ctx Context, excluded ExcludedIn) {
	desc := "This command cannot be used in this context."
	h.replyError(ctx, desc)
}

// OnNSFWRequired responds when an NSFW command is executed in a non-NSFW channel.
func (h *DefaultCommandErrorHandler) OnNSFWRequired(ctx Context) {
	desc := "This command can only be used in Age-Restricted (NSFW) channels."
	h.replyError(ctx, desc)
}

// OnCommandError responds with command runtime failure description.
func (h *DefaultCommandErrorHandler) OnCommandError(ctx Context, cmd any, err error) {
	cmdName := "unknown"
	if cmd != nil {
		cmdName = ExtractCommonConfig(cmd).Name
	}
	desc := fmt.Sprintf("An error occurred while running command `%s`: ```\n%s\n```", cmdName, err.Error())
	h.replyError(ctx, desc)
}

func (h *DefaultCommandErrorHandler) replyError(ctx Context, desc string) {
	embed := wumpkit.NewEmbedBuilder().
		SetColor(wumpkit.ColorRed).
		SetDescription(desc).
		Build()

	_, _ = ctx.Reply("", ReplyOptions{
		Embeds: []wumpkit.Embed{embed},
		Flags:  wumpkit.MessageFlagEphemeral,
	})
}

func formatPermissions(perms wumpkit.Permissions) string {
	names := perms.Names()
	if len(names) == 0 {
		return "None"
	}
	res := make([]string, len(names))
	for i, name := range names {
		res[i] = fmt.Sprintf("`%s`", string(name))
	}
	return strings.Join(res, ", ")
}
