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
	assert.NotNil(t, r1.errorHandler)
	_, ok := r1.errorHandler.(*DefaultCommandErrorHandler)
	assert.True(t, ok)
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
