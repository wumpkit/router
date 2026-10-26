package router

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit-go/wumpkit"
)

func TestRouter_HandleMessageExecution(t *testing.T) {
	r := New(WithPrefix("!"))

	var (
		executed       bool
		receivedUser   wumpkit.Snowflake
		receivedReason string
		receivedDays   int
		receivedSilent bool
	)

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "ban",
			Args: []ArgDef{
				{Name: "user", Type: ArgTypeUser, Required: true},
				{Name: "reason", Type: ArgTypeString, Endless: true},
			},
			Flags: []FlagDef{
				{Name: "days", Type: FlagTypeInt, Default: 1},
				{Name: "silent", Short: "s", Type: FlagTypeBool},
			},
		},
		runFunc: func(ctx Context) error {
			executed = true
			receivedUser = ctx.Args().Snowflake("user")
			receivedReason = ctx.Args().String("reason")
			receivedDays = ctx.Flags().Int("days")
			receivedSilent = ctx.Flags().Bool("s")
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!ban <@!123456789012345678> violating community guidelines --days=7 -s",
			Author: wumpkit.User{
				ID:       9999,
				Username: "Admin",
			},
		},
	})

	assert.True(t, executed)
	assert.Equal(t, wumpkit.Snowflake(123456789012345678), receivedUser)
	assert.Equal(t, "violating community guidelines", receivedReason)
	assert.Equal(t, 7, receivedDays)
	assert.True(t, receivedSilent)
}

func TestRouter_BotMessagesIgnored(t *testing.T) {
	r := New(WithPrefix("!"))

	var executed bool
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "ping",
		},
		runFunc: func(ctx Context) error {
			executed = true
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!ping",
			Author:  wumpkit.User{ID: 100, Bot: true},
		},
	})
	assert.False(t, executed, "bot messages should be ignored")
}

func TestRouter_NoPrefix(t *testing.T) {
	r := New(WithPrefix(""))

	var executed bool
	pingCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "ping",
		},
		runFunc: func(ctx Context) error {
			executed = true
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(pingCmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "ping",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.True(t, executed)
}

func TestRouter_PrefixAliases(t *testing.T) {
	r := New(WithPrefix("!"))

	var executed bool
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:    "ping",
			Aliases: []string{"latency", "p"},
		},
		runFunc: func(ctx Context) error {
			executed = true
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!latency",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.True(t, executed)

	executed = false
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!p",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.True(t, executed)
}

func TestRouter_WithPrefixFunc(t *testing.T) {
	r := New(WithPrefixFunc(func(msg *wumpkit.Message) string {
		if msg.GuildID == 999 {
			return "?"
		}
		return "!"
	}))

	var executed bool
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "ping",
		},
		runFunc: func(ctx Context) error {
			executed = true
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!ping",
			Author:  wumpkit.User{ID: 100},
			GuildID: 123,
		},
	})
	assert.True(t, executed, "should match '!' prefix for guild 123")

	executed = false
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "?ping",
			Author:  wumpkit.User{ID: 100},
			GuildID: 999,
		},
	})
	assert.True(t, executed, "should match '?' prefix for guild 999")
}

func TestRouter_HelpFlagMiddleware(t *testing.T) {
	var helpIntercepted bool
	var commandRan bool

	r := New(WithPrefix("!"))
	r.Use(func(next CommandHandler) CommandHandler {
		return func(ctx Context) error {
			if ctx.Flags().Bool("help") || ctx.Flags().Bool("h") {
				helpIntercepted = true
				return nil
			}
			return next(ctx)
		}
	})

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "ping",
			Flags: []FlagDef{
				{Name: "help", Short: "h", Type: FlagTypeBool},
			},
		},
		runFunc: func(ctx Context) error {
			commandRan = true
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!ping --help",
			Author:  wumpkit.User{ID: 100},
		},
	})

	assert.True(t, helpIntercepted)
	assert.False(t, commandRan)

	helpIntercepted = false
	commandRan = false
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!ping",
			Author:  wumpkit.User{ID: 100},
		},
	})

	assert.False(t, helpIntercepted)
	assert.True(t, commandRan)
}

func TestRouter_ComponentRouting(t *testing.T) {
	r := New()

	var (
		buttonHandled     bool
		selectMenuHandled bool
		modalHandled      bool
		receivedValues    []string
		receivedCustomID  string
	)

	btnCmd := &mockButtonCommand{
		ButtonCommandConfig: ButtonCommandConfig{
			CustomID: "ticket:create",
		},
		runFunc: func(ctx *ButtonCommandContext) error {
			buttonHandled = true
			receivedCustomID = ctx.CustomID()
			return nil
		},
	}
	require.NoError(t, r.RegisterButtonCommand(btnCmd))

	selectCmd := &mockSelectMenuCommand{
		SelectMenuCommandConfig: SelectMenuCommandConfig{
			CustomID: "role:select",
		},
		runFunc: func(ctx *SelectMenuCommandContext) error {
			selectMenuHandled = true
			receivedValues = ctx.Values()
			return nil
		},
	}
	require.NoError(t, r.RegisterSelectMenuCommand(selectCmd))

	modalCmd := &mockModalCommand{
		ModalCommandConfig: ModalCommandConfig{
			CustomID: "feedback:modal",
		},
		runFunc: func(ctx *ModalCommandContext) error {
			modalHandled = true
			return nil
		},
	}
	require.NoError(t, r.RegisterModalCommand(modalCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ComponentInteraction{
			Data: wumpkit.ComponentInteractionData{
				ComponentType: wumpkit.ComponentTypeButton,
				CustomID:      "ticket:create",
			},
		},
	})
	assert.True(t, buttonHandled)
	assert.Equal(t, "ticket:create", receivedCustomID)

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ComponentInteraction{
			Data: wumpkit.ComponentInteractionData{
				ComponentType: wumpkit.ComponentTypeStringSelect,
				CustomID:      "role:select",
				Values:        []string{"opt1", "opt2"},
			},
		},
	})
	assert.True(t, selectMenuHandled)
	assert.Equal(t, []string{"opt1", "opt2"}, receivedValues)

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ModalSubmitInteraction{
			Data: wumpkit.ModalSubmitInteractionData{CustomID: "feedback:modal"},
		},
	})
	assert.True(t, modalHandled)
}

func TestRouter_ComponentInteractionErrorLogging(t *testing.T) {
	logger := &mockLogger{}
	r := New(WithLogger(logger))
	r.Use(func(next CommandHandler) CommandHandler {
		return func(ctx Context) error {
			err := next(ctx)
			if err != nil {
				logger.Error("command execution failed")
			}
			return err
		}
	})

	btnCmd := &mockButtonCommand{
		ButtonCommandConfig: ButtonCommandConfig{
			CustomID: "fail-btn",
		},
		runFunc: func(ctx *ButtonCommandContext) error {
			return errors.New("button failure")
		},
	}
	require.NoError(t, r.RegisterButtonCommand(btnCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ComponentInteraction{
			Data: wumpkit.ComponentInteractionData{
				ComponentType: wumpkit.ComponentTypeButton,
				CustomID:      "fail-btn",
			},
		},
	})
	assert.Contains(t, logger.entries[len(logger.entries)-1], "command execution failed")

	selectCmd := &mockSelectMenuCommand{
		SelectMenuCommandConfig: SelectMenuCommandConfig{
			CustomID: "fail-select",
		},
		runFunc: func(ctx *SelectMenuCommandContext) error {
			return errors.New("select menu failure")
		},
	}
	require.NoError(t, r.RegisterSelectMenuCommand(selectCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ComponentInteraction{
			Data: wumpkit.ComponentInteractionData{
				ComponentType: wumpkit.ComponentTypeStringSelect,
				CustomID:      "fail-select",
			},
		},
	})
	assert.Contains(t, logger.entries[len(logger.entries)-1], "command execution failed")

	modalCmd := &mockModalCommand{
		ModalCommandConfig: ModalCommandConfig{
			CustomID: "fail-modal",
		},
		runFunc: func(ctx *ModalCommandContext) error {
			return errors.New("modal failure")
		},
	}
	require.NoError(t, r.RegisterModalCommand(modalCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ModalSubmitInteraction{
			Data: wumpkit.ModalSubmitInteractionData{CustomID: "fail-modal"},
		},
	})
	assert.Contains(t, logger.entries[len(logger.entries)-1], "command execution failed")
}

func TestRouter_Subcommands(t *testing.T) {
	r := New(WithPrefix("!"))

	var (
		prefixLeafExecuted  bool
		roleAddLeafExecuted bool
	)

	prefixLeaf := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "prefix",
			Args: []ArgDef{
				{Name: "new_prefix", Type: ArgTypeString, Required: true},
			},
		},
		runFunc: func(ctx Context) error {
			prefixLeafExecuted = true
			return nil
		},
	}

	roleAddLeaf := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "add",
			Args: []ArgDef{
				{Name: "role", Type: ArgTypeRole, Required: true},
			},
		},
		runFunc: func(ctx Context) error {
			roleAddLeafExecuted = true
			return nil
		},
	}

	roleGroup := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:        "role",
			Subcommands: []ChatCommand{roleAddLeaf},
		},
	}

	configRoot := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:        "config",
			Subcommands: []ChatCommand{prefixLeaf, roleGroup},
		},
	}
	require.NoError(t, r.RegisterCommand(configRoot))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!config prefix ?",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.True(t, prefixLeafExecuted)

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!config role add <@&12345>",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.True(t, roleAddLeafExecuted)

	prefixLeafExecuted = false
	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "config",
				},
				Options: []wumpkit.ChatInputInteractionCommandOption{
					{
						Type: wumpkit.ApplicationCommandOptionTypeSubCommand,
						Name: "prefix",
						Options: []wumpkit.ChatInputInteractionCommandOption{
							{Name: "new_prefix", Value: json.RawMessage(`"?"`)},
						},
					},
				},
			},
		},
	})
	assert.True(t, prefixLeafExecuted)

	roleAddLeafExecuted = false
	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "config",
				},
				Options: []wumpkit.ChatInputInteractionCommandOption{
					{
						Type: wumpkit.ApplicationCommandOptionTypeSubCommandGroup,
						Name: "role",
						Options: []wumpkit.ChatInputInteractionCommandOption{
							{
								Type: wumpkit.ApplicationCommandOptionTypeSubCommand,
								Name: "add",
								Options: []wumpkit.ChatInputInteractionCommandOption{
									{Name: "role", Value: json.RawMessage(`"12345"`)},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.True(t, roleAddLeafExecuted)
}

func TestRouter_Autocomplete(t *testing.T) {
	r := New()

	var hookCalled bool
	r.OnAutocomplete("tags", "query", func(ctx context.Context, inter *wumpkit.AutoCompleteInteraction) error {
		hookCalled = true
		return nil
	})

	autoCmd := &mockAutoCommand{
		mockCommand: mockCommand{
			ChatCommandConfig: ChatCommandConfig{Name: "search"},
		},
	}
	require.NoError(t, r.RegisterCommand(autoCmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.AutoCompleteInteraction{
			Data: wumpkit.AutoCompleteInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "tags",
				},
				Options: []wumpkit.ChatInputInteractionCommandOption{
					{
						Name:    "query",
						Focused: true,
						Value:   json.RawMessage(`"disc"`),
					},
				},
			},
		},
	})
	assert.True(t, hookCalled)

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.AutoCompleteInteraction{
			Data: wumpkit.AutoCompleteInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "search",
				},
				Options: []wumpkit.ChatInputInteractionCommandOption{
					{
						Name:    "query",
						Focused: true,
						Value:   json.RawMessage(`"auto"`),
					},
				},
			},
		},
	})
	assert.True(t, autoCmd.autocompleteCalled)
}

func TestRouter_DisabledInEnum(t *testing.T) {
	r := New(WithPrefix("!"))

	var slashExecuted, prefixExecuted, bothExecuted bool

	slashOnlyCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:       "slash-only",
			DisabledIn: DisabledInMessage,
		},
		runFunc: func(ctx Context) error {
			slashExecuted = true
			return nil
		},
	}
	prefixOnlyCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:       "prefix-only",
			DisabledIn: DisabledInSlash,
		},
		runFunc: func(ctx Context) error {
			prefixExecuted = true
			return nil
		},
	}
	bothDisabledCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:       "both-disabled",
			DisabledIn: DisabledInBoth,
		},
		runFunc: func(ctx Context) error {
			bothExecuted = true
			return nil
		},
	}

	require.NoError(t, r.RegisterCommand(slashOnlyCmd))
	require.NoError(t, r.RegisterCommand(prefixOnlyCmd))
	require.NoError(t, r.RegisterCommand(bothDisabledCmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!slash-only",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.False(t, slashExecuted)

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "prefix-only",
				},
			},
		},
	})
	assert.False(t, prefixExecuted)

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!both-disabled",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.False(t, bothExecuted)

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "both-disabled",
				},
			},
		},
	})
	assert.False(t, bothExecuted)
}

func TestRouter_ExcludeGuild(t *testing.T) {
	r := New(WithPrefix("!"))

	var executed bool
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:       "dm-only",
			ExcludedIn: ExcludeGuild,
		},
		runFunc: func(ctx Context) error {
			executed = true
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	// Message with a GuildID = guild context → should be excluded
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!dm-only",
			Author:  wumpkit.User{ID: 100},
			GuildID: 12345,
		},
	})
	assert.False(t, executed, "command should not run in a guild")

	// Message with no GuildID = DM context → should execute
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!dm-only",
			Author:  wumpkit.User{ID: 100},
		},
	})
	assert.True(t, executed, "command should run in DM")
}

func TestRouter_UserContextMenuCommand(t *testing.T) {
	r := New()

	var (
		middlewareInvocations int
		executed              bool
		receivedTarget        *wumpkit.User
	)

	r.Use(func(next CommandHandler) CommandHandler {
		return func(ctx Context) error {
			middlewareInvocations++
			return next(ctx)
		}
	})

	targetUserID := wumpkit.Snowflake(112233)
	targetUser := wumpkit.User{ID: targetUserID, Username: "TargetUser"}

	cmd := &mockUserCommand{
		UserContextMenuCommandConfig: UserContextMenuCommandConfig{
			Name: "InspectUser",
		},
		runFunc: func(ctx *UserContextMenuCommandContext) error {
			executed = true
			receivedTarget = ctx.TargetUser()
			return nil
		},
	}
	require.NoError(t, r.RegisterUserContextMenuCommand(cmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.UserContextMenuCommandInteraction{
			ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
				Channel: wumpkit.ResolvedChannel{
					ChannelFields: wumpkit.ChannelFields{
						ID:   5555,
						Type: wumpkit.ChannelTypeGuildText,
					},
				},
			},
			Data: wumpkit.UserContextMenuCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "InspectUser",
				},
				TargetID: targetUserID,
				Resolved: wumpkit.UserContextMenuCommandInteractionDataResolved{
					Users: map[wumpkit.Snowflake]wumpkit.User{
						targetUserID: targetUser,
					},
				},
			},
		},
	})

	assert.True(t, executed)
	require.NotNil(t, receivedTarget)
	assert.Equal(t, targetUserID, receivedTarget.ID)
	assert.Equal(t, "TargetUser", receivedTarget.Username)
	assert.Equal(t, 1, middlewareInvocations)
}

func TestRouter_MessageContextMenuCommand(t *testing.T) {
	r := New()

	var (
		middlewareInvocations int
		executed              bool
		receivedMessage       *wumpkit.Message
	)

	r.Use(func(next CommandHandler) CommandHandler {
		return func(ctx Context) error {
			middlewareInvocations++
			return next(ctx)
		}
	})

	targetMsgID := wumpkit.Snowflake(998877)
	targetMsg := wumpkit.Message{ID: targetMsgID, Content: "Hello world to quote!"}

	cmd := &mockMessageCommand{
		MessageContextMenuCommandConfig: MessageContextMenuCommandConfig{
			Name: "QuoteMessage",
		},
		runFunc: func(ctx *MessageContextMenuCommandContext) error {
			executed = true
			receivedMessage = ctx.TargetMessage()
			return nil
		},
	}
	require.NoError(t, r.RegisterMessageContextMenuCommand(cmd))

	r.HandleInteraction(context.Background(), wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.MessageContextMenuCommandInteraction{
			ApplicationCommandInteractionFields: wumpkit.ApplicationCommandInteractionFields{
				Channel: wumpkit.ResolvedChannel{
					ChannelFields: wumpkit.ChannelFields{
						ID:   5555,
						Type: wumpkit.ChannelTypeGuildText,
					},
				},
			},
			Data: wumpkit.MessageContextMenuCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "QuoteMessage",
				},
				TargetID: targetMsgID,
				Resolved: wumpkit.MessageContextMenuCommandInteractionDataResolved{
					Messages: map[wumpkit.Snowflake]wumpkit.Message{
						targetMsgID: targetMsg,
					},
				},
			},
		},
	})

	assert.True(t, executed)
	require.NotNil(t, receivedMessage)
	assert.Equal(t, targetMsgID, receivedMessage.ID)
	assert.Equal(t, "Hello world to quote!", receivedMessage.Content)
	assert.Equal(t, 1, middlewareInvocations)
}

func TestRouter_ArgsMentionParsing(t *testing.T) {
	userMention, err := parseArgType("<@!123456789>", ArgTypeUser)
	require.NoError(t, err)
	assert.Equal(t, wumpkit.Snowflake(123456789), userMention)

	channelMention, err := parseArgType("<#987654321>", ArgTypeChannel)
	require.NoError(t, err)
	assert.Equal(t, wumpkit.Snowflake(987654321), channelMention)

	roleMention, err := parseArgType("<@&555555555>", ArgTypeRole)
	require.NoError(t, err)
	assert.Equal(t, wumpkit.Snowflake(555555555), roleMention)
}

func TestRouter_PrefixAttachmentArgs(t *testing.T) {
	r := New(WithPrefix("!"))

	var (
		executed bool
		helloAtt wumpkit.Attachment
		worldAtt wumpkit.Attachment
		helloID  wumpkit.Snowflake
	)

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "attach",
			Args: []ArgDef{
				{Name: "hello", Type: ArgTypeAttachment, Required: true},
				{Name: "world", Type: ArgTypeAttachment, Required: true},
			},
		},
		runFunc: func(ctx Context) error {
			executed = true
			helloAtt = ctx.Args().Attachment("hello")
			worldAtt = ctx.Args().Attachment("world")
			helloID = ctx.Args().Snowflake("hello")
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	evt := wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			ID:        1001,
			ChannelID: 2001,
			Content:   "!attach",
			Author: wumpkit.User{
				ID:       9999,
				Username: "Admin",
			},
			Attachments: []wumpkit.Attachment{
				{ID: 111, Filename: "first.png"},
				{ID: 222, Filename: "second.txt"},
			},
		},
	}

	r.HandleMessage(context.Background(), evt)
	assert.True(t, executed)
	assert.Equal(t, wumpkit.Snowflake(111), helloAtt.ID)
	assert.Equal(t, "first.png", helloAtt.Filename)
	assert.Equal(t, wumpkit.Snowflake(222), worldAtt.ID)
	assert.Equal(t, "second.txt", worldAtt.Filename)
	assert.Equal(t, wumpkit.Snowflake(111), helloID)
}

func BenchmarkHandleMessage(b *testing.B) {
	r := New(WithPrefix("!"))
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{Name: "ping"},
	}
	_ = r.RegisterCommand(cmd)

	evt := wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			ID:        1001,
			ChannelID: 2001,
			Content:   "!ping",
			Author: wumpkit.User{
				ID:       9999,
				Username: "Admin",
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		r.HandleMessage(ctx, evt)
	}
}

func BenchmarkHandleInteraction(b *testing.B) {
	r := New()
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{Name: "ping"},
	}
	_ = r.RegisterCommand(cmd)

	evt := wumpkit.InteractionCreateEvent{
		Interaction: &wumpkit.ChatInputCommandInteraction{
			Data: wumpkit.ChatInputCommandInteractionData{
				ApplicationCommandInteractionDataFields: wumpkit.ApplicationCommandInteractionDataFields{
					Name: "ping",
				},
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		r.HandleInteraction(ctx, evt)
	}
}

func TestRouter_PermissionResolver(t *testing.T) {
	customUserPerms := wumpkit.PermissionManageChannels | wumpkit.PermissionKickMembers
	customAppPerms := wumpkit.PermissionSendMessages | wumpkit.PermissionEmbedLinks

	var resolverCalled bool
	r := New(
		WithPrefix("!"),
		WithPermissionResolver(func(ctx Context) (userPerms, appPerms wumpkit.Permissions, err error) {
			resolverCalled = true
			return customUserPerms, customAppPerms, nil
		}),
	)

	var (
		capturedUserPerms wumpkit.Permissions
		capturedAppPerms  wumpkit.Permissions
		executed          bool
	)

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "checkperms",
		},
		runFunc: func(ctx Context) error {
			executed = true
			capturedUserPerms = ctx.UserPermissions()
			capturedAppPerms = ctx.AppPermissions()
			return nil
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	guildID := wumpkit.Snowflake(777)
	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!checkperms",
			GuildID: guildID,
			Author: wumpkit.User{
				ID:       1234,
				Username: "Tester",
			},
		},
	})

	assert.True(t, resolverCalled)
	assert.True(t, executed)
	assert.Equal(t, customUserPerms, capturedUserPerms)
	assert.Equal(t, customAppPerms, capturedAppPerms)
}

func TestRouter_ExecutePipelinePanicRecovery(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(
		WithPrefix("!"),
		WithCommandErrorHandler(handler),
	)

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "panic-now",
		},
		runFunc: func(ctx Context) error {
			panic("intentional disaster")
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
		Message: wumpkit.Message{
			Content: "!panic-now",
			Author:  wumpkit.User{ID: 100},
		},
	})

	assert.True(t, handler.executionErrorCalled)
	require.NotNil(t, handler.lastErr)
	assert.Contains(t, handler.lastErr.Error(), "router: panic recovered during command execution: intentional disaster")
}

func TestComputeGuildPermissions(t *testing.T) {
	guildID := wumpkit.Snowflake(100)
	ownerID := wumpkit.Snowflake(200)

	roles := map[wumpkit.Snowflake]wumpkit.Role{
		guildID: {
			ID:          guildID,
			Permissions: wumpkit.PermissionSendMessages | wumpkit.PermissionViewChannel,
		},
		wumpkit.Snowflake(101): {
			ID:          101,
			Permissions: wumpkit.PermissionManageMessages,
		},
		wumpkit.Snowflake(102): {
			ID:          102,
			Permissions: wumpkit.PermissionAdministrator,
		},
	}

	// guild owner gets all permissions
	perms := computeGuildPermissions(ownerID, ownerID, guildID, roles, nil)
	assert.Equal(t, ^wumpkit.Permissions(0), perms)

	// regular member gets @everyone + their roles
	memberID := wumpkit.Snowflake(300)
	perms = computeGuildPermissions(memberID, ownerID, guildID, roles, []wumpkit.Snowflake{101})
	expected := wumpkit.PermissionSendMessages | wumpkit.PermissionViewChannel | wumpkit.PermissionManageMessages
	assert.Equal(t, expected, perms)

	// member with administrator gets all permissions
	adminMemberID := wumpkit.Snowflake(400)
	perms = computeGuildPermissions(adminMemberID, ownerID, guildID, roles, []wumpkit.Snowflake{102})
	assert.Equal(t, ^wumpkit.Permissions(0), perms)
}

func TestApplicationCommandsEqual(t *testing.T) {
	cmdA := wumpkit.NewSlashCommandBuilder("ping", "Replies with pong").
		AddStringOption("msg", "Optional message", false).
		Build()

	// identical
	cmdB := wumpkit.NewSlashCommandBuilder("ping", "Replies with pong").
		AddStringOption("msg", "Optional message", false).
		Build()
	assert.True(t, applicationCommandsEqual([]wumpkit.ApplicationCommand{cmdA}, []wumpkit.ApplicationCommand{cmdB}))

	// different lengths
	assert.False(t, applicationCommandsEqual([]wumpkit.ApplicationCommand{cmdA}, nil))

	// modified description
	cmdC := wumpkit.NewSlashCommandBuilder("ping", "Something different").
		AddStringOption("msg", "Optional message", false).
		Build()
	assert.False(t, applicationCommandsEqual([]wumpkit.ApplicationCommand{cmdA}, []wumpkit.ApplicationCommand{cmdC}))

	// modified options
	cmdD := wumpkit.NewSlashCommandBuilder("ping", "Replies with pong").
		AddIntegerOption("msg", "Optional message", false).
		Build()
	assert.False(t, applicationCommandsEqual([]wumpkit.ApplicationCommand{cmdA}, []wumpkit.ApplicationCommand{cmdD}))

	// modified default member permissions
	cmdE := wumpkit.NewSlashCommandBuilder("ping", "Replies with pong").
		SetDefaultMemberPermissions(wumpkit.PermissionManageChannels).
		AddStringOption("msg", "Optional message", false).
		Build()
	assert.False(t, applicationCommandsEqual([]wumpkit.ApplicationCommand{cmdA}, []wumpkit.ApplicationCommand{cmdE}))

	// multiple commands in different order should still be equal
	cmd2 := wumpkit.NewSlashCommandBuilder("help", "Help command").Build()
	assert.True(t, applicationCommandsEqual(
		[]wumpkit.ApplicationCommand{cmdA, cmd2},
		[]wumpkit.ApplicationCommand{cmd2, cmdA},
	))
}
