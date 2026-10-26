package middleware_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/router/middleware"
	"github.com/wumpkit-go/wumpkit"
)

func TestMiddleware_Logging(t *testing.T) {
	logger := &mockLogger{}
	r := router.New(router.WithPrefix("!"), router.WithLogger(logger))
	r.Use(middleware.Logging(logger))

	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "ping",
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!ping",
			Author:  wumpkit.User{ID: 100, Username: "Tester"},
		},
	})

	require.NotEmpty(t, logger.entries)
	found := false
	for _, entry := range logger.entries {
		if entry == "INFO: command executed successfully" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'command executed successfully' log entry")
}

func TestMiddleware_Logging_Error(t *testing.T) {
	logger := &mockLogger{}
	r := router.New(router.WithPrefix("!"))
	r.Use(middleware.Logging(logger))

	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "fail",
		},
		runFunc: func(ctx router.Context) error {
			return errors.New("command execution failed")
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!fail",
			Author:  wumpkit.User{ID: 100},
		},
	})

	found := false
	for _, entry := range logger.entries {
		if entry == "ERROR: command execution failed" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'command execution failed' log entry")
}
