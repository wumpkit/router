package router

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit/wumpkit"
)

// ctxKey is an unexported key type for context value tests.
type ctxKey string

func TestContext_CompatibilitySurface(t *testing.T) {
	parentCtx := context.Background()
	channelID := wumpkit.Snowflake(1111)
	guildID := wumpkit.Snowflake(2222)

	c := NewContext(parentCtx, ContextConfig{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
				ChannelID: channelID,
				GuildID:   guildID,
				Member: &wumpkit.InteractionMember{
					FullMember: wumpkit.FullMember{
						User:   wumpkit.User{ID: 3333, Username: "Bob"},
						Member: wumpkit.Member{Nickname: "Bobby"},
					},
				},
			},
		},
	})

	assert.Equal(t, channelID, c.ChannelID())
	assert.Equal(t, &guildID, c.GuildID())
	assert.Equal(t, wumpkit.Snowflake(3333), c.Author().ID)
	assert.Equal(t, "Bobby", c.AuthorMember().Nickname)
	assert.True(t, c.InGuild())
}

func TestContext_InGuild(t *testing.T) {
	guildID := wumpkit.Snowflake(999)

	guildCtx := NewContext(context.Background(), ContextConfig{
		Message: &wumpkit.Message{GuildID: guildID},
	})
	assert.True(t, guildCtx.InGuild())

	dmCtx := NewContext(context.Background(), ContextConfig{})
	assert.False(t, dmCtx.InGuild())
}

func TestContext_StoreAndStdContext(t *testing.T) {
	parentCtx := context.WithValue(context.Background(), ctxKey("request_id"), "req-999")

	c := NewContext(parentCtx, ContextConfig{})

	// standard context.Context Value() inheritance
	assert.Equal(t, "req-999", c.Value(ctxKey("request_id")))

	// router internal key-value store (Get/Set)
	c.Set("internal_key", 42)
	val, ok := c.Get("internal_key")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestContext_Event(t *testing.T) {
	tests := []struct {
		name        string
		config      ContextConfig
		assertEvent func(t *testing.T, ev any)
	}{
		{
			name: "prefix message",
			config: ContextConfig{
				Message: &wumpkit.Message{ID: 12345, Content: "!ping"},
				ShardID: 2,
			},
			assertEvent: func(t *testing.T, ev any) {
				e, ok := ev.(wumpkit.MessageCreateEvent)
				require.True(t, ok)
				assert.Equal(t, wumpkit.Snowflake(12345), e.Message.ID)
				assert.Equal(t, "!ping", e.Message.Content)
				assert.Equal(t, wumpkit.ShardID(2), e.ShardID)
			},
		},
		{
			name: "slash command interaction",
			config: ContextConfig{
				Interaction: &wumpkit.ChatInputCommandInteraction{
					ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
						InteractionFields: wumpkit.InteractionFields{ID: 67890},
					},
				},
				ShardID: 3,
			},
			assertEvent: func(t *testing.T, ev any) {
				e, ok := ev.(wumpkit.InteractionCreateEvent)
				require.True(t, ok)
				inter, ok := e.Interaction.(*wumpkit.ChatInputCommandInteraction)
				require.True(t, ok)
				assert.Equal(t, wumpkit.Snowflake(67890), inter.ID)
				assert.Equal(t, wumpkit.ShardID(3), e.ShardID)
			},
		},
		{
			name: "user context menu command interaction",
			config: ContextConfig{
				Interaction: &wumpkit.UserContextMenuCommandInteraction{
					ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
						InteractionFields: wumpkit.InteractionFields{ID: 111222},
					},
				},
			},
			assertEvent: func(t *testing.T, ev any) {
				e, ok := ev.(wumpkit.InteractionCreateEvent)
				require.True(t, ok)
				inter, ok := e.Interaction.(*wumpkit.UserContextMenuCommandInteraction)
				require.True(t, ok)
				assert.Equal(t, wumpkit.Snowflake(111222), inter.ID)
			},
		},
		{
			name: "message context menu command interaction",
			config: ContextConfig{
				Interaction: &wumpkit.MessageContextMenuCommandInteraction{
					ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
						InteractionFields: wumpkit.InteractionFields{ID: 333444},
					},
				},
			},
			assertEvent: func(t *testing.T, ev any) {
				e, ok := ev.(wumpkit.InteractionCreateEvent)
				require.True(t, ok)
				inter, ok := e.Interaction.(*wumpkit.MessageContextMenuCommandInteraction)
				require.True(t, ok)
				assert.Equal(t, wumpkit.Snowflake(333444), inter.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContext(context.Background(), tt.config)
			tt.assertEvent(t, c.Event())
		})
	}
}

func TestArgs_Has(t *testing.T) {
	args := Args{values: map[string]any{"name": "Alice"}}

	assert.True(t, args.Has("name"))
	assert.False(t, args.Has("missing"))

	empty := Args{}
	assert.False(t, empty.Has("anything"))
}

func TestArgs_SlashTypes(t *testing.T) {
	args := Args{
		values: map[string]any{
			"str":   "hello",
			"int":   int64(42),
			"float": 3.1415,
			"bool":  true,
			"user":  wumpkit.Snowflake(987654321),
		},
	}

	assert.Equal(t, "hello", args.String("str"))
	assert.Equal(t, 42, args.Int("int"))
	assert.Equal(t, 3.1415, args.Float("float"))
	assert.Equal(t, true, args.Bool("bool"))
	assert.Equal(t, wumpkit.Snowflake(987654321), args.Snowflake("user"))

	// missing keys return zero values
	assert.Equal(t, "", args.String("missing"))
	assert.Equal(t, 0, args.Int("missing"))
	assert.Equal(t, 0.0, args.Float("missing"))
	assert.Equal(t, false, args.Bool("missing"))
	assert.Equal(t, wumpkit.Snowflake(0), args.Snowflake("missing"))
}

func TestArgs_AttachmentMissing(t *testing.T) {
	empty := Args{}
	att := empty.Attachment("file")
	assert.Equal(t, wumpkit.Attachment{}, att)
}

func TestArgs_TypeMismatchSafe(t *testing.T) {
	args := Args{
		values: map[string]any{
			"str":       "not-a-number",
			"int_val":   42,
			"float_val": 3.14,
			"bool_val":  true,
			"user_val":  wumpkit.Snowflake(123456789),
		},
	}

	// Safe getters return zero value on mismatch without panicking
	assert.Equal(t, 0, args.Int("str"))
	assert.Equal(t, false, args.Bool("int_val"))
	assert.Equal(t, 0.0, args.Float("bool_val"))
	assert.Equal(t, wumpkit.Snowflake(0), args.Snowflake("float_val"))

	// *E getters return informative errors on mismatch
	_, err := args.IntE("str")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `cannot be parsed as int`)

	_, err = args.BoolE("int_val")
	assert.Error(t, err)

	_, err = args.FloatE("bool_val")
	assert.Error(t, err)

	_, err = args.SnowflakeE("float_val")
	assert.Error(t, err)

	// Valid retrievals
	assert.Equal(t, 42, args.Int("int_val"))
	v, err := args.IntE("int_val")
	assert.NoError(t, err)
	assert.Equal(t, 42, v)

	raw, ok := args.Get("int_val")
	assert.True(t, ok)
	assert.Equal(t, 42, raw)
}

func TestFlags_Has(t *testing.T) {
	flags := Flags{values: map[string]string{"silent": "true"}}

	assert.True(t, flags.Has("silent"))
	assert.False(t, flags.Has("missing"))

	empty := Flags{}
	assert.False(t, empty.Has("anything"))
}

func TestFlags_TypesAndMentions(t *testing.T) {
	flags := Flags{
		values: map[string]string{
			"str":     "hello",
			"int":     "42",
			"float":   "3.1415",
			"bool":    "true",
			"user":    "<@!987654321>",
			"channel": "<#123456789>",
		},
	}

	assert.True(t, flags.Has("str"))
	assert.False(t, flags.Has("nonexistent"))
	assert.Equal(t, "hello", flags.String("str"))
	assert.Equal(t, 42, flags.Int("int"))
	assert.Equal(t, 3.1415, flags.Float("float"))
	assert.Equal(t, true, flags.Bool("bool"))
	assert.Equal(t, wumpkit.Snowflake(987654321), flags.Snowflake("user"))
	assert.Equal(t, wumpkit.Snowflake(123456789), flags.Snowflake("channel"))

	// missing keys return zero values
	assert.Equal(t, "", flags.String("missing"))
	assert.Equal(t, 0, flags.Int("missing"))
	assert.Equal(t, 0.0, flags.Float("missing"))
	assert.Equal(t, false, flags.Bool("missing"))
	assert.Equal(t, wumpkit.Snowflake(0), flags.Snowflake("missing"))
}

func TestSpecializedContexts_Methods(t *testing.T) {
	// ButtonCommandContext
	compInter := &wumpkit.ComponentInteraction{
		Data: wumpkit.ComponentInteractionData{
			ComponentType: wumpkit.ComponentTypeButton,
			CustomID:      "Hello/earth",
		},
	}
	btnCtx := &ButtonCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: context.Background(),
			event:   wumpkit.InteractionCreateEvent{ShardID: 1, Interaction: compInter},
		},
		patternContext: patternContext{
			matchedPattern: "Hello/{world}",
			params:         map[string]string{"world": "earth"},
		},
	}
	assert.Equal(t, "Hello/earth", btnCtx.CustomID())
	assert.Equal(t, "Hello/{world}", btnCtx.MatchedPattern())
	assert.Equal(t, "earth", btnCtx.Param("world"))
	assert.Equal(t, "earth", btnCtx.Param("WORLD"))
	assert.Equal(t, "", btnCtx.Param("missing"))
	assert.Equal(t, map[string]string{"world": "earth"}, btnCtx.Params())
	assert.Equal(t, wumpkit.ShardID(1), btnCtx.ShardID())

	// SelectMenuCommandContext
	selectInter := &wumpkit.ComponentInteraction{
		Data: wumpkit.ComponentInteractionData{
			ComponentType: wumpkit.ComponentTypeStringSelect,
			CustomID:      "menus/roles/choose",
			Values:        []string{"v1", "v2"},
		},
	}
	selectCtx := &SelectMenuCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: context.Background(),
			event:   wumpkit.InteractionCreateEvent{ShardID: 2, Interaction: selectInter},
		},
		patternContext: patternContext{
			matchedPattern: "menus/{menu_id}/choose",
			params:         map[string]string{"menu_id": "roles"},
		},
	}
	assert.Equal(t, "menus/roles/choose", selectCtx.CustomID())
	assert.Equal(t, "menus/{menu_id}/choose", selectCtx.MatchedPattern())
	assert.Equal(t, "roles", selectCtx.Param("menu_id"))
	assert.Equal(t, map[string]string{"menu_id": "roles"}, selectCtx.Params())
	assert.Equal(t, []string{"v1", "v2"}, selectCtx.Values())
	assert.Equal(t, wumpkit.ShardID(2), selectCtx.ShardID())

	// ModalCommandContext
	modalData := &wumpkit.ModalSubmitInteractionData{
		CustomID: "feedback/user/42",
		Components: []wumpkit.Component{
			&wumpkit.ActionRowComponent{
				Components: []wumpkit.InteractiveComponent{
					&wumpkit.TextInputComponent{
						InteractiveComponentFields: wumpkit.InteractiveComponentFields{CustomID: "feedback_text"},
						Value:                      "great bot!",
					},
				},
			},
		},
	}
	modalCtx := &ModalCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: context.Background(),
			event:   wumpkit.InteractionCreateEvent{Interaction: &wumpkit.ModalSubmitInteraction{Data: *modalData}},
		},
		patternContext: patternContext{
			matchedPattern: "feedback/user/{id}",
			params:         map[string]string{"id": "42"},
		},
	}
	assert.Equal(t, "feedback/user/42", modalCtx.CustomID())
	assert.Equal(t, "42", modalCtx.Param("id"))
	assert.Equal(t, "great bot!", modalCtx.InputValue("feedback_text"))
	assert.Equal(t, "", modalCtx.InputValue("missing_field"))
	assert.Equal(t, map[string]string{"feedback_text": "great bot!"}, modalCtx.ModalValues())

	// UserContextMenuCommandContext
	targetUserID := wumpkit.Snowflake(444)
	userCtx := &UserContextMenuCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: context.Background(),
			event: wumpkit.InteractionCreateEvent{
				Interaction: &wumpkit.UserContextMenuCommandInteraction{
					Data: wumpkit.UserContextMenuCommandInteractionData{
						TargetID: targetUserID,
						Resolved: wumpkit.UserContextMenuCommandInteractionDataResolved{
							Users: map[wumpkit.Snowflake]wumpkit.User{
								targetUserID: {ID: targetUserID, Username: "TargetUser"},
							},
							Members: map[wumpkit.Snowflake]wumpkit.PartialMember{
								targetUserID: {Nickname: "TargetNick"},
							},
						},
					},
				},
			},
		},
	}
	require.NotNil(t, userCtx.TargetUser())
	assert.Equal(t, "TargetUser", userCtx.TargetUser().Username)
	require.NotNil(t, userCtx.TargetMember())
	assert.Equal(t, "TargetNick", userCtx.TargetMember().Nickname)

	// MessageContextMenuCommandContext
	targetMsgID := wumpkit.Snowflake(555)
	msgCtx := &MessageContextMenuCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: context.Background(),
			event: wumpkit.InteractionCreateEvent{
				Interaction: &wumpkit.MessageContextMenuCommandInteraction{
					Data: wumpkit.MessageContextMenuCommandInteractionData{
						TargetID: targetMsgID,
						Resolved: wumpkit.MessageContextMenuCommandInteractionDataResolved{
							Messages: map[wumpkit.Snowflake]wumpkit.Message{
								targetMsgID: {ID: targetMsgID, Content: "Target Content"},
							},
						},
					},
				},
			},
		},
	}
	require.NotNil(t, msgCtx.TargetMessage())
	assert.Equal(t, "Target Content", msgCtx.TargetMessage().Content)
}
