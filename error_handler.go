package router

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wumpkit/wumpkit"
)

// errnsfwrequired indicates a command requires an age-restricted channel
var ErrNSFWRequired = errors.New(
	"router: command can only be used in age-restricted (nsfw) channels",
)

// missingargumenterror represents a missing required argument
type MissingArgumentError struct {
	Name     string
	Expected string
}

func (e *MissingArgumentError) Error() string {
	return fmt.Sprintf("missing required argument: <%s> (%s)", e.Name, e.Expected)
}

// invalidargumenterror represents an argument that failed type conversion
type InvalidArgumentError struct {
	Name     string
	Expected string
	Received string
	Err      error
}

func (e *InvalidArgumentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%q is not a valid %s for %q: %v", e.Received, e.Expected, e.Name, e.Err)
	}
	return fmt.Sprintf("%q is not a valid %s for %q", e.Received, e.Expected, e.Name)
}

func (e *InvalidArgumentError) Unwrap() error {
	return e.Err
}

// userpermissionmissingerror represents missing user permissions
type UserPermissionMissingError struct {
	Missing wumpkit.Permissions
}

func (e *UserPermissionMissingError) Error() string {
	return fmt.Sprintf("user is missing required permission(s): %s", formatPermissions(e.Missing))
}

// apppermissionmissingerror represents missing bot permissions
type AppPermissionMissingError struct {
	Missing wumpkit.Permissions
}

func (e *AppPermissionMissingError) Error() string {
	return fmt.Sprintf("bot is missing required permission(s): %s", formatPermissions(e.Missing))
}

// contextexcludederror represents execution in an excluded context
type ContextExcludedError struct {
	Excluded ExcludedIn
}

func (e *ContextExcludedError) Error() string {
	return fmt.Sprintf("command cannot be executed in this context (excluded: %v)", e.Excluded)
}

// commanderror wraps a command execution error and the command instance
type CommandError struct {
	Command any
	Err     error
}

func (e *CommandError) Error() string {
	cmdName := "unknown"
	if e.Command != nil {
		cmdName = ExtractCommonConfig(e.Command).Name
	}
	return fmt.Sprintf("command %q failed: %v", cmdName, e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// errorhandler handles router and command lifecycle errors
type ErrorHandler func(ctx Context, err error)

// defaulterrorhandler is the standard error handler responding with formatted embeds
func DefaultErrorHandler(ctx Context, err error) {
	if ctx == nil || err == nil {
		return
	}

	var (
		missingArg *MissingArgumentError
		invalidArg *InvalidArgumentError
		userPerm   *UserPermissionMissingError
		appPerm    *AppPermissionMissingError
		cooldown   *CooldownError
		excluded   *ContextExcludedError
		cmdErr     *CommandError
	)

	var desc string
	switch {
	case errors.As(err, &missingArg):
		desc = fmt.Sprintf(
			"Missing required argument: `<%s>` (%s)",
			missingArg.Name,
			missingArg.Expected,
		)
	case errors.As(err, &invalidArg):
		desc = fmt.Sprintf(
			"`%s` is not a valid %s for `%s`",
			invalidArg.Received,
			invalidArg.Expected,
			invalidArg.Name,
		)
	case errors.As(err, &userPerm):
		desc = fmt.Sprintf(
			"You are missing the required permission(s): %s",
			formatPermissions(userPerm.Missing),
		)
	case errors.As(err, &appPerm):
		desc = fmt.Sprintf(
			"The bot is missing the required permission(s): %s",
			formatPermissions(appPerm.Missing),
		)
	case errors.As(err, &cooldown):
		var target string
		switch cooldown.Scope {
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
		desc = fmt.Sprintf(
			"%s on cooldown! Please wait `%.1fs` before using this command again",
			target,
			cooldown.Remaining.Seconds(),
		)
	case errors.As(err, &excluded):
		desc = "This command cannot be used in this context."
	case errors.Is(err, ErrNSFWRequired):
		desc = "This command can only be used in Age-Restricted (NSFW) channels."
	case errors.As(err, &cmdErr):
		cmdName := "unknown"
		if cmdErr.Command != nil {
			cmdName = ExtractCommonConfig(cmdErr.Command).Name
		}
		desc = fmt.Sprintf(
			"An error occurred while running command `%s`: ```\n%s\n```",
			cmdName,
			cmdErr.Err.Error(),
		)
	default:
		cmdName := "unknown"
		if ctx.Command() != nil {
			cmdName = ExtractCommonConfig(ctx.Command()).Name
		}
		desc = fmt.Sprintf(
			"An error occurred while running command `%s`: ```\n%s\n```",
			cmdName,
			err.Error(),
		)
	}

	embed := wumpkit.NewEmbedBuilder().
		SetColor(wumpkit.ColorRed).
		SetDescription(desc).
		Build()

	_, _ = ctx.Reply("", ReplyOptions{
		Embeds: []wumpkit.Embed{embed},
		Flags:  wumpkit.MessageFlagEphemeral,
	})
}

// adaptcommanderrorhandler wraps a legacy commanderrorhandler as an errorhandler
func AdaptCommandErrorHandler(h CommandErrorHandler) ErrorHandler {
	if h == nil {
		return DefaultErrorHandler
	}
	return func(ctx Context, err error) {
		var (
			missingArg *MissingArgumentError
			invalidArg *InvalidArgumentError
			userPerm   *UserPermissionMissingError
			appPerm    *AppPermissionMissingError
			cooldown   *CooldownError
			excluded   *ContextExcludedError
			cmdErr     *CommandError
		)

		switch {
		case errors.As(err, &missingArg):
			h.OnMissingArgument(ctx, missingArg.Name, missingArg.Expected)
		case errors.As(err, &invalidArg):
			h.OnInvalidArgument(ctx, invalidArg.Name, invalidArg.Expected, invalidArg.Received)
		case errors.As(err, &userPerm):
			h.OnUserPermissionMissing(ctx, userPerm.Missing)
		case errors.As(err, &appPerm):
			h.OnAppPermissionMissing(ctx, appPerm.Missing)
		case errors.As(err, &cooldown):
			h.OnCooldown(ctx, cooldown.Remaining, cooldown.Scope)
		case errors.As(err, &excluded):
			h.OnContextExcluded(ctx, excluded.Excluded)
		case errors.Is(err, ErrNSFWRequired):
			h.OnNSFWRequired(ctx)
		case errors.As(err, &cmdErr):
			h.OnCommandError(ctx, cmdErr.Command, cmdErr.Err)
		default:
			var cmd any
			if ctx != nil {
				cmd = ctx.Command()
			}
			h.OnCommandError(ctx, cmd, err)
		}
	}
}

// commanderrorhandler is a legacy interface for handling command errors
type CommandErrorHandler interface {
	OnMissingArgument(ctx Context, name string, expected string)
	OnInvalidArgument(ctx Context, name string, expected string, received string)
	OnUserPermissionMissing(ctx Context, missing wumpkit.Permissions)
	OnAppPermissionMissing(ctx Context, missing wumpkit.Permissions)
	OnCooldown(ctx Context, remaining time.Duration, scope CooldownScope)
	OnContextExcluded(ctx Context, excluded ExcludedIn)
	OnNSFWRequired(ctx Context)
	OnCommandError(ctx Context, cmd any, err error)
}

type commandErrorHandlerAdapter struct {
	handler ErrorHandler
}

func (a *commandErrorHandlerAdapter) OnMissingArgument(ctx Context, name string, expected string) {
	a.handler(ctx, &MissingArgumentError{Name: name, Expected: expected})
}

func (a *commandErrorHandlerAdapter) OnInvalidArgument(
	ctx Context,
	name string,
	expected string,
	received string,
) {
	a.handler(ctx, &InvalidArgumentError{Name: name, Expected: expected, Received: received})
}

func (a *commandErrorHandlerAdapter) OnUserPermissionMissing(
	ctx Context,
	missing wumpkit.Permissions,
) {
	a.handler(ctx, &UserPermissionMissingError{Missing: missing})
}

func (a *commandErrorHandlerAdapter) OnAppPermissionMissing(
	ctx Context,
	missing wumpkit.Permissions,
) {
	a.handler(ctx, &AppPermissionMissingError{Missing: missing})
}

func (a *commandErrorHandlerAdapter) OnCooldown(
	ctx Context,
	remaining time.Duration,
	scope CooldownScope,
) {
	a.handler(ctx, &CooldownError{Remaining: remaining, Scope: scope})
}

func (a *commandErrorHandlerAdapter) OnContextExcluded(ctx Context, excluded ExcludedIn) {
	a.handler(ctx, &ContextExcludedError{Excluded: excluded})
}

func (a *commandErrorHandlerAdapter) OnNSFWRequired(ctx Context) {
	a.handler(ctx, ErrNSFWRequired)
}

func (a *commandErrorHandlerAdapter) OnCommandError(ctx Context, cmd any, err error) {
	a.handler(ctx, &CommandError{Command: cmd, Err: err})
}

// defaultcommanderrorhandler is a legacy adapter for defaulterrorhandler
type DefaultCommandErrorHandler struct{}

// newdefaultcommanderrorhandler creates a new defaultcommanderrorhandler instance
func NewDefaultCommandErrorHandler() *DefaultCommandErrorHandler {
	return &DefaultCommandErrorHandler{}
}

func (h *DefaultCommandErrorHandler) OnMissingArgument(ctx Context, name string, expected string) {
	DefaultErrorHandler(ctx, &MissingArgumentError{Name: name, Expected: expected})
}

func (h *DefaultCommandErrorHandler) OnInvalidArgument(
	ctx Context,
	name string,
	expected string,
	received string,
) {
	DefaultErrorHandler(
		ctx,
		&InvalidArgumentError{Name: name, Expected: expected, Received: received},
	)
}

func (h *DefaultCommandErrorHandler) OnUserPermissionMissing(
	ctx Context,
	missing wumpkit.Permissions,
) {
	DefaultErrorHandler(ctx, &UserPermissionMissingError{Missing: missing})
}

func (h *DefaultCommandErrorHandler) OnAppPermissionMissing(
	ctx Context,
	missing wumpkit.Permissions,
) {
	DefaultErrorHandler(ctx, &AppPermissionMissingError{Missing: missing})
}

func (h *DefaultCommandErrorHandler) OnCooldown(
	ctx Context,
	remaining time.Duration,
	scope CooldownScope,
) {
	DefaultErrorHandler(ctx, &CooldownError{Remaining: remaining, Scope: scope})
}

func (h *DefaultCommandErrorHandler) OnContextExcluded(ctx Context, excluded ExcludedIn) {
	DefaultErrorHandler(ctx, &ContextExcludedError{Excluded: excluded})
}

func (h *DefaultCommandErrorHandler) OnNSFWRequired(ctx Context) {
	DefaultErrorHandler(ctx, ErrNSFWRequired)
}

func (h *DefaultCommandErrorHandler) OnCommandError(ctx Context, cmd any, err error) {
	DefaultErrorHandler(ctx, &CommandError{Command: cmd, Err: err})
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
