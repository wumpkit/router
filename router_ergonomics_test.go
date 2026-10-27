package router

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit/wumpkit"
)

type pingCommand struct {
	ChatCommandConfig
	called bool
}

func (p *pingCommand) Run(ctx Context) error {
	p.called = true
	return nil
}

func TestRouter_TypeDefinedCommands(t *testing.T) {
	r := New()

	cmd := &pingCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:        "ping",
			Description: "Ping command",
		},
	}
	r.MustRegisterCommand(cmd)

	retrieved, exists := r.GetCommand("ping")
	require.True(t, exists)
	assert.Equal(t, "ping", retrieved.Config().Name)
	assert.Equal(t, "Ping command", retrieved.Config().Description)

	err := retrieved.Run(&MessageCommandContext{})
	require.NoError(t, err)
	assert.True(t, cmd.called)
}

type ticketButtonCommand struct {
	ButtonCommandConfig
	called     bool
	capturedID string
}

func (b *ticketButtonCommand) Run(ctx *ButtonCommandContext) error {
	b.called = true
	b.capturedID = ctx.Param("id")
	return nil
}

func TestRouter_TypeDefinedComponentCommands(t *testing.T) {
	r := New()

	btn := &ticketButtonCommand{
		ButtonCommandConfig: ButtonCommandConfig{
			CustomID: "ticket:{id}",
		},
	}
	r.MustRegisterButtonCommand(btn)

	m, exists := r.MatchButtonCommand("ticket:12345")
	require.True(t, exists)
	assert.Equal(t, "12345", m.Params["id"])

	bCtx := &ButtonCommandContext{
		patternContext: patternContext{
			matchedPattern: m.Pattern,
			params:         m.Params,
		},
	}
	err := m.Value.Run(bCtx)
	require.NoError(t, err)
	assert.True(t, btn.called)
	assert.Equal(t, "12345", btn.capturedID)
}

func TestRouter_CustomArgParser(t *testing.T) {
	r := New()

	type CustomID int

	var executedVal CustomID
	r.MustRegisterCommand(&mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "order",
			Args: []ArgDef{
				{
					Name: "id",
					Type: ArgTypeString,
					Parser: func(raw string) (any, error) {
						num, err := strconv.Atoi(raw)
						if err != nil {
							return nil, errors.New("invalid order id")
						}
						return CustomID(num), nil
					},
				},
			},
		},
		runFunc: func(ctx Context) error {
			val, ok := ctx.Args().Get("id")
			if ok {
				executedVal = val.(CustomID)
			}
			return nil
		},
	})

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!order 42",
		},
	})

	assert.Equal(t, CustomID(42), executedVal)
}

func TestRouter_BeforeAndAfterHooks(t *testing.T) {
	r := New()

	var sequence []string
	r.Before(func(ctx Context) error {
		sequence = append(sequence, "before")
		return nil
	})

	r.After(func(ctx Context, err error) {
		sequence = append(sequence, "after")
	})

	r.MustRegisterCommand(&mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "test",
		},
		runFunc: func(ctx Context) error {
			sequence = append(sequence, "run")
			return nil
		},
	})

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!test",
		},
	})

	assert.Equal(t, []string{"before", "run", "after"}, sequence)
}

func TestRouter_DynamicPrefix(t *testing.T) {
	r := New(WithDynamicPrefixFunc(func(ctx context.Context, msg *wumpkit.Message) (string, error) {
		if msg.GuildID == 999 {
			return "?", nil
		}
		return "!", nil
	}))

	var called bool
	r.MustRegisterCommand(&mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "test",
		},
		runFunc: func(ctx Context) error {
			called = true
			return nil
		},
	})

	// should not trigger on ! for guild 999
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			GuildID: 999,
			Content: "!test",
		},
	})
	assert.False(t, called)

	// should trigger on ? for guild 999
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			GuildID: 999,
			Content: "?test",
		},
	})
	assert.True(t, called)
}

type mockCooldownStore struct {
	allowFunc func(key string, limit int, duration time.Duration) (bool, time.Duration, error)
}

func (m *mockCooldownStore) Allow(key string, limit int, duration time.Duration) (bool, time.Duration, error) {
	if m.allowFunc != nil {
		return m.allowFunc(key, limit, duration)
	}
	return true, 0, nil
}
