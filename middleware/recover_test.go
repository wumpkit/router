package middleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/router/middleware"
	"github.com/wumpkit-go/wumpkit"
)

type mockLogger struct {
	entries []string
}

func (l *mockLogger) Debug(msg string, args ...any) { l.entries = append(l.entries, "DEBUG: "+msg) }
func (l *mockLogger) Info(msg string, args ...any)  { l.entries = append(l.entries, "INFO: "+msg) }
func (l *mockLogger) Warn(msg string, args ...any)  { l.entries = append(l.entries, "WARN: "+msg) }
func (l *mockLogger) Error(msg string, args ...any) { l.entries = append(l.entries, "ERROR: "+msg) }

type trackingErrorHandler struct {
	router.DefaultCommandErrorHandler
	executionErrorCalled bool
}

func (t *trackingErrorHandler) OnCommandError(ctx router.Context, cmd any, err error) {
	t.executionErrorCalled = true
}

type mockCommand struct {
	router.ChatCommandConfig
	runFunc func(ctx router.Context) error
}

func (m *mockCommand) Config() router.ChatCommandConfig { return m.ChatCommandConfig }
func (m *mockCommand) Run(ctx router.Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	return nil
}

func TestMiddleware_Recovery_LogsAndWraps(t *testing.T) {
	var middlewareCalled bool

	handler := &trackingErrorHandler{}
	logger := &mockLogger{}
	r := router.New(router.WithPrefix("!"), router.WithCommandErrorHandler(handler))

	r.Use(func(next router.CommandHandler) router.CommandHandler {
		return func(ctx router.Context) error {
			middlewareCalled = true
			return next(ctx)
		}
	})
	r.Use(middleware.Recovery(logger))

	panicCmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "panic",
		},
		runFunc: func(ctx router.Context) error {
			panic("intentional test panic")
		},
	}
	require.NoError(t, r.RegisterCommand(panicCmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!panic",
			Author:  wumpkit.User{ID: 100},
		},
	})

	assert.True(t, middlewareCalled)
	assert.True(t, handler.executionErrorCalled)
	assert.NotEmpty(t, logger.entries)
	assert.Contains(t, logger.entries[0], "panic recovered during command execution")
}
