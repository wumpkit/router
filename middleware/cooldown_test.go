package middleware_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit-go/router"
	"github.com/wumpkit-go/router/middleware"
	"github.com/wumpkit-go/wumpkit"
)

type cooldownTrackingHandler struct {
	router.DefaultCommandErrorHandler
	cooldownCalled    bool
	cooldownRemaining time.Duration
	cooldownScope     router.CooldownScope
}

func (c *cooldownTrackingHandler) OnCooldown(ctx router.Context, remaining time.Duration, scope router.CooldownScope) {
	c.cooldownCalled = true
	c.cooldownRemaining = remaining
	c.cooldownScope = scope
}

func TestCooldown_MultiRuleGCRA(t *testing.T) {
	r := router.New(router.WithPrefix("!"))
	r.Use(middleware.Cooldown())

	var execCount int
	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "spam",
			Cooldowns: []router.CooldownRule{
				{Duration: 1 * time.Second, Scope: router.CooldownScopeUser, Limit: 2},
				{Duration: 5 * time.Second, Scope: router.CooldownScopeUser, Limit: 5},
			},
		},
		runFunc: func(ctx router.Context) error {
			execCount++
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	evt := wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!spam",
			Author:  wumpkit.User{ID: 12345},
		},
	}

	// 1st and 2nd should pass burst of 2
	r.HandleMessage(context.Background(), evt)
	r.HandleMessage(context.Background(), evt)
	assert.Equal(t, 2, execCount)

	// 3rd should be rejected by 1s limit 2 rule
	r.HandleMessage(context.Background(), evt)
	assert.Equal(t, 2, execCount)
}

func TestCooldown_SingleRule(t *testing.T) {
	r := router.New(router.WithPrefix("!"))
	r.Use(middleware.Cooldown())

	var execCount int
	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "ping",
			Cooldowns: []router.CooldownRule{
				{Duration: 500 * time.Millisecond, Scope: router.CooldownScopeUser, Limit: 1},
			},
		},
		runFunc: func(ctx router.Context) error {
			execCount++
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	evt := wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!ping",
			Author:  wumpkit.User{ID: 100},
		},
	}

	r.HandleMessage(context.Background(), evt)
	assert.Equal(t, 1, execCount)

	// immediately calling again fails
	r.HandleMessage(context.Background(), evt)
	assert.Equal(t, 1, execCount)
}

func TestCooldown_ScopeChannel(t *testing.T) {
	r := router.New(router.WithPrefix("!"))
	r.Use(middleware.Cooldown())

	var execCount int
	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "channel-limited",
			Cooldowns: []router.CooldownRule{
				{Duration: 1 * time.Second, Scope: router.CooldownScopeChannel, Limit: 1},
			},
		},
		runFunc: func(ctx router.Context) error {
			execCount++
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	// User 1 in Channel A
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content:   "!channel-limited",
			ChannelID: 1000,
			Author:    wumpkit.User{ID: 1},
		},
	})
	assert.Equal(t, 1, execCount)

	// User 2 in Channel A should fail
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content:   "!channel-limited",
			ChannelID: 1000,
			Author:    wumpkit.User{ID: 2},
		},
	})
	assert.Equal(t, 1, execCount)

	// User 2 in Channel B should succeed
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content:   "!channel-limited",
			ChannelID: 2000,
			Author:    wumpkit.User{ID: 2},
		},
	})
	assert.Equal(t, 2, execCount)
}

func TestCooldown_ScopeMember(t *testing.T) {
	r := router.New(router.WithPrefix("!"))
	r.Use(middleware.Cooldown())

	var execCount int
	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "member-limited",
			Cooldowns: []router.CooldownRule{
				{Duration: 1 * time.Second, Scope: router.CooldownScopeMember, Limit: 1},
			},
		},
		runFunc: func(ctx router.Context) error {
			execCount++
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	// User 1 in Guild A
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!member-limited",
			GuildID: 10,
			Author:  wumpkit.User{ID: 1},
		},
	})
	assert.Equal(t, 1, execCount)

	// User 1 in Guild A again -> fails
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!member-limited",
			GuildID: 10,
			Author:  wumpkit.User{ID: 1},
		},
	})
	assert.Equal(t, 1, execCount)

	// User 1 in Guild B -> succeeds
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!member-limited",
			GuildID: 20,
			Author:  wumpkit.User{ID: 1},
		},
	})
	assert.Equal(t, 2, execCount)
}

func TestCooldown_ScopeGlobal(t *testing.T) {
	r := router.New(router.WithPrefix("!"))
	r.Use(middleware.Cooldown())

	var execCount int
	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "global-limited",
			Cooldowns: []router.CooldownRule{
				{Duration: 1 * time.Second, Scope: router.CooldownScopeGlobal, Limit: 1},
			},
		},
		runFunc: func(ctx router.Context) error {
			execCount++
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	// User 1
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!global-limited",
			Author:  wumpkit.User{ID: 1},
		},
	})
	assert.Equal(t, 1, execCount)

	// User 2 -> fails because command is globally on cooldown
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!global-limited",
			Author:  wumpkit.User{ID: 2},
		},
	})
	assert.Equal(t, 1, execCount)
}

func TestCooldown_TrackingHandler(t *testing.T) {
	handler := &cooldownTrackingHandler{}
	r := router.New(router.WithPrefix("!"), router.WithCommandErrorHandler(handler))
	r.Use(middleware.Cooldown())

	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "rate-limited",
			Cooldowns: []router.CooldownRule{
				{Duration: 10 * time.Second, Scope: router.CooldownScopeUser, Limit: 1},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	// first call succeeds
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!rate-limited",
			Author:  wumpkit.User{ID: 1},
		},
	})

	// second call hits cooldown
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!rate-limited",
			Author:  wumpkit.User{ID: 1},
		},
	})

	assert.True(t, handler.cooldownCalled)
	assert.Equal(t, router.CooldownScopeUser, handler.cooldownScope)
	assert.True(t, handler.cooldownRemaining > 0)
}

func TestCooldown_Sweeper(t *testing.T) {
	r := router.New(router.WithPrefix("!"))
	r.Use(middleware.Cooldown())

	var execCount int
	cmd := &mockCommand{
		ChatCommandConfig: router.ChatCommandConfig{
			Name: "sweep-cmd",
			Cooldowns: []router.CooldownRule{
				{Duration: 1 * time.Millisecond, Scope: router.CooldownScopeUser, Limit: 1},
			},
		},
		runFunc: func(ctx router.Context) error {
			execCount++
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	// run 600 operations with different users to exceed opCount > 512 threshold and trigger sweep
	for i := 1; i <= 600; i++ {
		r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
			Message: wumpkit.Message{
				Content: "!sweep-cmd",
				Author:  wumpkit.User{ID: wumpkit.Snowflake(i)},
			},
		})
	}

	assert.Equal(t, 600, execCount)
}
