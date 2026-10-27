package router

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit/wumpkit"
)

func TestPatternMatcher_ExactMatches(t *testing.T) {
	matcher := NewPatternMatcher[string]()

	require.NoError(t, matcher.Insert("ticket/create", "val_create"))
	require.NoError(t, matcher.Insert("ticket/close", "val_close"))
	require.NoError(t, matcher.Insert("admin/ban", "val_ban"))

	res, ok := matcher.Match("ticket/create")
	assert.True(t, ok)
	assert.Equal(t, "val_create", res.Value)
	assert.Equal(t, "ticket/create", res.Pattern)
	assert.Empty(t, res.Params)

	res, ok = matcher.Match("ticket/close")
	assert.True(t, ok)
	assert.Equal(t, "val_close", res.Value)

	_, ok = matcher.Match("ticket/unknown")
	assert.False(t, ok)
}

func TestPatternMatcher_NamedParameters(t *testing.T) {
	matcher := NewPatternMatcher[string]()

	require.NoError(t, matcher.Insert("hello/{World}/foo/{Bar}", "multi_param_handler"))
	require.NoError(t, matcher.Insert("Hello/{world}", "greeting_handler"))
	require.NoError(t, matcher.Insert("users/{user_id}/profile", "profile_handler"))

	// Test multi-param pattern: hello/{World}/foo/{Bar} matching hello/earth/foo/baz
	res, ok := matcher.Match("hello/earth/foo/baz")
	assert.True(t, ok)
	assert.Equal(t, "multi_param_handler", res.Value)
	assert.Equal(t, "hello/{World}/foo/{Bar}", res.Pattern)
	assert.Equal(t, "earth", res.Params["World"])
	assert.Equal(t, "baz", res.Params["Bar"])

	// Test single param pattern: Hello/{world} matching Hello/earth
	res, ok = matcher.Match("Hello/earth")
	assert.True(t, ok)
	assert.Equal(t, "greeting_handler", res.Value)
	assert.Equal(t, "Hello/{world}", res.Pattern)
	assert.Equal(t, "earth", res.Params["world"])

	// Test users/{user_id}/profile matching users/12345/profile
	res, ok = matcher.Match("users/12345/profile")
	assert.True(t, ok)
	assert.Equal(t, "profile_handler", res.Value)
	assert.Equal(t, "12345", res.Params["user_id"])
}

func TestPatternMatcher_MultipleParametersAndNamespacing(t *testing.T) {
	matcher := NewPatternMatcher[string]()

	require.NoError(t, matcher.Insert("tickets/{ticket_id}/action/{action_name}", "ticket_action_handler"))

	res, ok := matcher.Match("tickets/7788/action/escalate")
	assert.True(t, ok)
	assert.Equal(t, "ticket_action_handler", res.Value)
	assert.Equal(t, "7788", res.Params["ticket_id"])
	assert.Equal(t, "escalate", res.Params["action_name"])
}

func TestPatternMatcher_StaticPrecedenceOverParam(t *testing.T) {
	matcher := NewPatternMatcher[string]()

	require.NoError(t, matcher.Insert("users/new", "static_new"))
	require.NoError(t, matcher.Insert("users/{id}", "param_user"))

	// Exact match should take precedence over parameter
	res, ok := matcher.Match("users/new")
	assert.True(t, ok)
	assert.Equal(t, "static_new", res.Value)
	assert.Empty(t, res.Params)

	// Non-matching exact segment should match parameter
	res, ok = matcher.Match("users/999")
	assert.True(t, ok)
	assert.Equal(t, "param_user", res.Value)
	assert.Equal(t, "999", res.Params["id"])
}

func TestPatternMatcher_LeadingAndTrailingSlashes(t *testing.T) {
	matcher := NewPatternMatcher[string]()

	require.NoError(t, matcher.Insert("/Hello/{world}/", "greeting"))

	res, ok := matcher.Match("/Hello/moon/")
	assert.True(t, ok)
	assert.Equal(t, "greeting", res.Value)
	assert.Equal(t, "moon", res.Params["world"])
}

func TestRouter_ButtonCommandWithHTTPPathParams(t *testing.T) {
	r := New()

	var (
		buttonExecuted  bool
		receivedPattern string
		receivedParam   string
		receivedWorld   string
	)

	btnCmd := &mockButtonCommand{
		ButtonCommandConfig: ButtonCommandConfig{
			CustomID: "Hello/{world}",
		},
		runFunc: func(ctx *ButtonCommandContext) error {
			buttonExecuted = true
			receivedPattern = ctx.MatchedPattern()
			receivedParam = ctx.Param("world")
			receivedWorld = ctx.Param("WORLD")
			return nil
		},
	}
	require.NoError(t, r.RegisterButtonCommand(btnCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ComponentInteraction{
			Data: wumpkit.ComponentInteractionData{
				ComponentType: wumpkit.ComponentTypeButton,
				CustomID:      "Hello/earth",
			},
		},
	})

	assert.True(t, buttonExecuted)
	assert.Equal(t, "Hello/{world}", receivedPattern)
	assert.Equal(t, "earth", receivedParam)
	assert.Equal(t, "earth", receivedWorld)
}

func TestRouter_ButtonCommand_MultiParam(t *testing.T) {
	r := New()

	var (
		buttonExecuted bool
		receivedWorld  string
		receivedBar    string
	)

	btnCmd := &mockButtonCommand{
		ButtonCommandConfig: ButtonCommandConfig{
			CustomID: "hello/{World}/foo/{Bar}",
		},
		runFunc: func(ctx *ButtonCommandContext) error {
			buttonExecuted = true
			receivedWorld = ctx.Param("World")
			receivedBar = ctx.Param("Bar")
			return nil
		},
	}
	require.NoError(t, r.RegisterButtonCommand(btnCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ComponentInteraction{
			Data: wumpkit.ComponentInteractionData{
				ComponentType: wumpkit.ComponentTypeButton,
				CustomID:      "hello/earth/foo/baz",
			},
		},
	})

	assert.True(t, buttonExecuted)
	assert.Equal(t, "earth", receivedWorld)
	assert.Equal(t, "baz", receivedBar)
}

func TestRouter_SelectMenuCommandWithHTTPPathParams(t *testing.T) {
	r := New()

	var (
		selectExecuted bool
		receivedMenu   string
		receivedValues []string
	)

	selectCmd := &mockSelectMenuCommand{
		SelectMenuCommandConfig: SelectMenuCommandConfig{
			CustomID: "menus/{menu_name}/select",
		},
		runFunc: func(ctx *SelectMenuCommandContext) error {
			selectExecuted = true
			receivedMenu = ctx.Param("menu_name")
			receivedValues = ctx.Values()
			return nil
		},
	}
	require.NoError(t, r.RegisterSelectMenuCommand(selectCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ComponentInteraction{
			Data: wumpkit.ComponentInteractionData{
				ComponentType: wumpkit.ComponentTypeStringSelect,
				CustomID:      "menus/roles/select",
				Values:        []string{"role_admin", "role_mod"},
			},
		},
	})

	assert.True(t, selectExecuted)
	assert.Equal(t, "roles", receivedMenu)
	assert.Equal(t, []string{"role_admin", "role_mod"}, receivedValues)
}

func TestRouter_ModalCommandWithHTTPPathParams(t *testing.T) {
	r := New()

	var (
		modalExecuted bool
		receivedID    string
		receivedInput string
	)

	modalCmd := &mockModalCommand{
		ModalCommandConfig: ModalCommandConfig{
			CustomID: "feedback/{category}/submit",
		},
		runFunc: func(ctx *ModalCommandContext) error {
			modalExecuted = true
			receivedID = ctx.Param("category")
			receivedInput = ctx.InputValue("details")
			return nil
		},
	}
	require.NoError(t, r.RegisterModalCommand(modalCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ModalSubmitInteraction{
			Data: wumpkit.ModalSubmitInteractionData{
				CustomID: "feedback/billing/submit",
				Components: []wumpkit.Component{
					&wumpkit.ActionRowComponent{
						Components: []wumpkit.InteractiveComponent{
							&wumpkit.TextInputComponent{
								InteractiveComponentFields: wumpkit.InteractiveComponentFields{CustomID: "details"},
								Value:                      "Invoice issue",
							},
						},
					},
				},
			},
		},
	})

	assert.True(t, modalExecuted)
	assert.Equal(t, "billing", receivedID)
	assert.Equal(t, "Invoice issue", receivedInput)
}

func TestPatternMatcher_DeepNestingAndAmbiguity(t *testing.T) {
	matcher := NewPatternMatcher[string]()

	require.NoError(t, matcher.Insert("a/b/c/d", "static_deep"))
	require.NoError(t, matcher.Insert("a/{b}/c/{d}", "param_deep"))
	require.NoError(t, matcher.Insert("a/b/{c}/d", "semi_static"))

	// Should match exact static
	res, ok := matcher.Match("a/b/c/d")
	assert.True(t, ok)
	assert.Equal(t, "static_deep", res.Value)

	// Should match semi-static (a/b/anything/d)
	res, ok = matcher.Match("a/b/foo/d")
	assert.True(t, ok)
	assert.Equal(t, "semi_static", res.Value)
	assert.Equal(t, "foo", res.Params["c"])

	// Should match param_deep
	res, ok = matcher.Match("a/x/c/y")
	assert.True(t, ok)
	assert.Equal(t, "param_deep", res.Value)
	assert.Equal(t, "x", res.Params["b"])
	assert.Equal(t, "y", res.Params["d"])
}
