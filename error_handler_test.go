package router

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit/wumpkit"
)

func TestRouter_FallsBackToDefaultErrorHandler(t *testing.T) {
	r1 := New()
	assert.NotNil(t, r1.ErrorHandler())
	assert.NotNil(t, r1.errorHandler)
}

func TestErrorHandler_OnMissingArgument(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "need-arg",
			Args: []ArgDef{
				{Name: "username", Type: ArgTypeString, Required: true},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!need-arg",
			Author:  wumpkit.User{ID: 1},
		},
	})

	assert.True(t, handler.missingArgCalled)
	assert.Equal(t, "username", handler.missingArgName)
}

func TestErrorHandler_OnInvalidArgument(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "int-arg",
			Args: []ArgDef{
				{Name: "count", Type: ArgTypeInt, Required: true},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!int-arg not-an-int",
			Author:  wumpkit.User{ID: 1},
		},
	})

	assert.True(t, handler.invalidArgCalled)
	assert.Equal(t, "not-an-int", handler.invalidArgReceived)
}

func TestErrorHandler_OnContextExcluded(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:       "guild-only",
			ExcludedIn: ExcludeDM,
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!guild-only",
			Author:  wumpkit.User{ID: 1},
			GuildID: 0,
		},
	})

	assert.True(t, handler.contextExcludedCalled)
	assert.Equal(t, ExcludeDM, handler.contextExcludedValue)
}

func TestErrorHandler_OnUserPermissionMissing(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithCommandErrorHandler(handler))

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:            "kick",
			UserPermissions: wumpkit.PermissionKickMembers,
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
				Member: &wumpkit.InteractionMember{
					Permissions: wumpkit.PermissionSendMessages,
				},
			},
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name:    "kick",
					GuildID: 100,
				},
			},
		},
	})

	assert.True(t, handler.userPermMissingCalled)
	assert.Equal(t, wumpkit.PermissionKickMembers, handler.userPermMissingValue)
}

func TestErrorHandler_OnAppPermissionMissing(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithCommandErrorHandler(handler))

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:           "bot-ban",
			AppPermissions: wumpkit.PermissionBanMembers,
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
				AppPermissions: wumpkit.PermissionSendMessages,
			},
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name:    "bot-ban",
					GuildID: 100,
				},
			},
		},
	})

	assert.True(t, handler.appPermMissingCalled)
}

func TestErrorHandler_OnCommandError(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "failing-cmd",
		},
		runFunc: func(ctx Context) error {
			return errors.New("command execution failed")
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!failing-cmd",
			Author:  wumpkit.User{ID: 1},
		},
	})

	assert.True(t, handler.executionErrorCalled)
}

func TestErrorHandler_UserContextMenuCommand(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithCommandErrorHandler(handler))

	cmd := &mockUserCommand{
		UserContextMenuCommandConfig: UserContextMenuCommandConfig{
			Name: "Inspect",
		},
		runFunc: func(ctx *UserContextMenuCommandContext) error {
			return errors.New("user command failed")
		},
	}
	require.NoError(t, r.RegisterUserContextMenuCommand(cmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.UserContextMenuCommandInteraction{
			Data: wumpkit.UserContextMenuCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "Inspect",
				},
			},
		},
	})

	assert.True(t, handler.executionErrorCalled)
}

func TestErrorHandler_MessageContextMenuCommand(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithCommandErrorHandler(handler))

	cmd := &mockMessageCommand{
		MessageContextMenuCommandConfig: MessageContextMenuCommandConfig{
			Name: "Quote",
		},
		runFunc: func(ctx *MessageContextMenuCommandContext) error {
			return errors.New("message command failed")
		},
	}
	require.NoError(t, r.RegisterMessageContextMenuCommand(cmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.MessageContextMenuCommandInteraction{
			Data: wumpkit.MessageContextMenuCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "Quote",
				},
			},
		},
	})

	assert.True(t, handler.executionErrorCalled)
}

func TestMiddleware_ErrorPropagation(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))
	r.Use(func(next CommandHandler) CommandHandler {
		return func(ctx Context) error {
			return errors.New("custom pipeline failure")
		}
	})

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "fail",
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!fail",
			Author:  wumpkit.User{ID: 1},
		},
	})

	assert.True(t, handler.executionErrorCalled)
}

func TestRouter_OnError_Idiomatic(t *testing.T) {
	var capturedErr error
	r := New(WithPrefix("!")).OnError(func(ctx Context, err error) {
		capturedErr = err
	})

	// test missing argument error
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "need-arg",
			Args: []ArgDef{
				{Name: "username", Type: ArgTypeString, Required: true},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!need-arg",
			Author:  wumpkit.User{ID: 1},
		},
	})

	var missingArg *MissingArgumentError
	require.True(t, errors.As(capturedErr, &missingArg))
	assert.Equal(t, "username", missingArg.Name)
	assert.Equal(t, "string", missingArg.Expected)
	assert.Contains(t, missingArg.Error(), "missing required argument: <username>")

	// test invalid argument error
	intCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "int-cmd",
			Args: []ArgDef{
				{Name: "amount", Type: ArgTypeInt, Required: true},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(intCmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!int-cmd not-int",
			Author:  wumpkit.User{ID: 1},
		},
	})

	var invalidArg *InvalidArgumentError
	require.True(t, errors.As(capturedErr, &invalidArg))
	assert.Equal(t, "amount", invalidArg.Name)
	assert.Equal(t, "not-int", invalidArg.Received)
	assert.Nil(t, invalidArg.Unwrap())
	assert.Contains(t, invalidArg.Error(), "\"not-int\" is not a valid int")

	// test context excluded error
	guildCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:       "guild-only",
			ExcludedIn: ExcludeDM,
		},
	}
	require.NoError(t, r.RegisterCommand(guildCmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!guild-only",
			Author:  wumpkit.User{ID: 1},
			GuildID: 0,
		},
	})

	var excludedErr *ContextExcludedError
	require.True(t, errors.As(capturedErr, &excludedErr))
	assert.Equal(t, ExcludeDM, excludedErr.Excluded)
	assert.Contains(t, excludedErr.Error(), "excluded: 2")

	// test user permission missing error
	kickCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:            "kick",
			UserPermissions: wumpkit.PermissionKickMembers,
		},
	}
	require.NoError(t, r.RegisterCommand(kickCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
				Member: &wumpkit.InteractionMember{
					Permissions: wumpkit.PermissionSendMessages,
				},
			},
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name:    "kick",
					GuildID: 100,
				},
			},
		},
	})

	var userPermErr *UserPermissionMissingError
	require.True(t, errors.As(capturedErr, &userPermErr))
	assert.Equal(t, wumpkit.PermissionKickMembers, userPermErr.Missing)
	assert.Contains(t, userPermErr.Error(), "user is missing required permission(s)")

	// test app permission missing error
	banCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:           "ban",
			AppPermissions: wumpkit.PermissionBanMembers,
		},
	}
	require.NoError(t, r.RegisterCommand(banCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
				AppPermissions: wumpkit.PermissionSendMessages,
			},
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name:    "ban",
					GuildID: 100,
				},
			},
		},
	})

	var appPermErr *AppPermissionMissingError
	require.True(t, errors.As(capturedErr, &appPermErr))
	assert.Equal(t, wumpkit.PermissionBanMembers, appPermErr.Missing)
	assert.Contains(t, appPermErr.Error(), "bot is missing required permission(s)")

	// test runtime command error
	failCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "fail-run",
		},
		runFunc: func(ctx Context) error {
			return errors.New("underlying failure")
		},
	}
	require.NoError(t, r.RegisterCommand(failCmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!fail-run",
			Author:  wumpkit.User{ID: 1},
		},
	})

	var cmdErr *CommandError
	require.True(t, errors.As(capturedErr, &cmdErr))
	assert.Equal(t, "fail-run", ExtractCommonConfig(cmdErr.Command).Name)
	assert.Equal(t, "underlying failure", cmdErr.Unwrap().Error())
	assert.Contains(t, cmdErr.Error(), "command \"fail-run\" failed: underlying failure")
}

func TestRouter_WithErrorHandler_Option(t *testing.T) {
	called := false
	r := New(WithErrorHandler(func(ctx Context, err error) {
		called = true
	}))

	r.handleError(nil, errors.New("test"))
	assert.True(t, called)

	// test seterrorhandler fluent method
	r.SetErrorHandler(func(ctx Context, err error) {
		called = false
	})
	r.handleError(nil, errors.New("test"))
	assert.False(t, called)
}
