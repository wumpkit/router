package router

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wumpkit/wumpkit"
)

func TestRegistry_CommandRegistrationAndLookup(t *testing.T) {
	r := New()

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:        "ban",
			Description: "Bans a member",
			Aliases:     []string{"b", "banish"},
			Args: []ArgDef{
				{Name: "user", Type: ArgTypeUser, Required: true},
				{Name: "reason", Type: ArgTypeString, Endless: true},
			},
			Flags: []FlagDef{
				{Name: "delete-days", Short: "d", Type: FlagTypeInt, Default: 7},
				{Name: "silent", Short: "s", Type: FlagTypeBool},
			},
		},
	}

	pingCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:      "ping",
			Aliases:   []string{"p", "latency"},
			Cooldowns: []CooldownRule{{Duration: 5 * time.Second}},
		},
	}

	require.NoError(t, r.RegisterCommand(cmd))
	require.NoError(t, r.RegisterCommand(pingCmd))

	found, ok := r.GetCommand("ban")
	require.True(t, ok)
	assert.Equal(t, "ban", found.Config().Name)

	foundAlias, ok := r.GetCommand("b")
	require.True(t, ok)
	assert.Equal(t, "ban", foundAlias.Config().Name)

	foundPing, ok := r.GetCommand("latency")
	require.True(t, ok)
	assert.Equal(t, "ping", foundPing.Config().Name)

	cmdsMap := r.Commands()
	assert.Contains(t, cmdsMap, "ban")
	assert.Contains(t, cmdsMap, "b")
	assert.Contains(t, cmdsMap, "banish")
	assert.Contains(t, cmdsMap, "ping")
	assert.Contains(t, cmdsMap, "p")
	assert.Contains(t, cmdsMap, "latency")
}

func TestRegistry_NilAndEmptyNameErrors(t *testing.T) {
	r := New()

	assert.ErrorIs(t, r.RegisterCommand(nil), ErrNilCommand)

	emptyCmd := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: ""}}
	assert.ErrorIs(t, r.RegisterCommand(emptyCmd), ErrEmptyCommandName)
}

func TestRegistry_DuplicateCommandAndAliasErrors(t *testing.T) {
	r := New()

	pingCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:    "ping",
			Aliases: []string{"latency", "p"},
		},
	}
	require.NoError(t, r.RegisterCommand(pingCmd))

	dupPing := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "ping"}}
	assert.ErrorIs(t, r.RegisterCommand(dupPing), ErrCommandAlreadyExists)

	// command whose name collides with an existing alias
	aliasCollision := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "latency"}}
	assert.ErrorIs(t, r.RegisterCommand(aliasCollision), ErrCommandAlreadyExists)

	// command whose alias collides with an existing command name
	cmdWithDupAlias := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:    "status",
			Aliases: []string{"ping"},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(cmdWithDupAlias), ErrAliasAlreadyExists)

	// command whose alias collides with an existing alias
	cmdWithDupAlias2 := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:    "echo",
			Aliases: []string{"p"},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(cmdWithDupAlias2), ErrAliasAlreadyExists)
}

func TestRegistry_RegisterAliasErrors(t *testing.T) {
	r := New()

	assert.ErrorIs(t, r.RegisterAlias("alias1", "nonexistent"), ErrTargetCommandNotFound)

	pingCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:    "ping",
			Aliases: []string{"p"},
		},
	}
	require.NoError(t, r.RegisterCommand(pingCmd))

	infoCmd := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "info"}}
	require.NoError(t, r.RegisterCommand(infoCmd))

	// "p" already maps to ping, so it cannot be re-aliased to info
	assert.ErrorIs(t, r.RegisterAlias("p", "info"), ErrAliasAlreadyExists)
}

func TestRegistry_ContextMenuCommandErrors(t *testing.T) {
	r := New()

	uCmd := &mockUserCommand{
		UserContextMenuCommandConfig: UserContextMenuCommandConfig{Name: "inspect"},
	}
	require.NoError(t, r.RegisterUserContextMenuCommand(uCmd))
	assert.ErrorIs(
		t,
		r.RegisterUserContextMenuCommand(uCmd),
		ErrUserContextMenuCommandAlreadyExists,
	)

	mCmd := &mockMessageCommand{
		MessageContextMenuCommandConfig: MessageContextMenuCommandConfig{Name: "quote"},
	}
	require.NoError(t, r.RegisterMessageContextMenuCommand(mCmd))
	assert.ErrorIs(
		t,
		r.RegisterMessageContextMenuCommand(mCmd),
		ErrMessageContextMenuCommandAlreadyExists,
	)
}

func TestRegistry_ComponentAndModalCommandErrors(t *testing.T) {
	r := New()

	assert.ErrorIs(t, r.RegisterButtonCommand(nil), ErrNilCommand)
	assert.ErrorIs(
		t,
		r.RegisterButtonCommand(
			&mockButtonCommand{ButtonCommandConfig: ButtonCommandConfig{CustomID: ""}},
		),
		ErrEmptyCustomID,
	)

	btnCmd := &mockButtonCommand{ButtonCommandConfig: ButtonCommandConfig{CustomID: "btn:ok"}}
	require.NoError(t, r.RegisterButtonCommand(btnCmd))
	assert.ErrorIs(t, r.RegisterButtonCommand(btnCmd), ErrButtonCommandAlreadyExists)

	assert.ErrorIs(t, r.RegisterSelectMenuCommand(nil), ErrNilCommand)
	assert.ErrorIs(
		t,
		r.RegisterSelectMenuCommand(
			&mockSelectMenuCommand{SelectMenuCommandConfig: SelectMenuCommandConfig{CustomID: ""}},
		),
		ErrEmptyCustomID,
	)

	selectCmd := &mockSelectMenuCommand{
		SelectMenuCommandConfig: SelectMenuCommandConfig{CustomID: "menu:choose"},
	}
	require.NoError(t, r.RegisterSelectMenuCommand(selectCmd))
	assert.ErrorIs(t, r.RegisterSelectMenuCommand(selectCmd), ErrSelectMenuCommandAlreadyExists)

	assert.ErrorIs(t, r.RegisterModalCommand(nil), ErrNilCommand)
	assert.ErrorIs(
		t,
		r.RegisterModalCommand(
			&mockModalCommand{ModalCommandConfig: ModalCommandConfig{CustomID: ""}},
		),
		ErrEmptyCustomID,
	)

	modalCmd := &mockModalCommand{ModalCommandConfig: ModalCommandConfig{CustomID: "form:submit"}}
	require.NoError(t, r.RegisterModalCommand(modalCmd))
	assert.ErrorIs(t, r.RegisterModalCommand(modalCmd), ErrModalCommandAlreadyExists)
}

func TestRegistry_MustPanics(t *testing.T) {
	r := New()
	r.MustRegisterCommand(&mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "help"}})

	assert.Panics(t, func() {
		r.MustRegisterCommand(&mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "help"}})
	})

	r.MustRegisterUserContextMenuCommand(
		&mockUserCommand{
			UserContextMenuCommandConfig: UserContextMenuCommandConfig{Name: "user_help"},
		},
	)
	assert.Panics(t, func() {
		r.MustRegisterUserContextMenuCommand(
			&mockUserCommand{
				UserContextMenuCommandConfig: UserContextMenuCommandConfig{Name: "user_help"},
			},
		)
	})

	r.MustRegisterMessageContextMenuCommand(
		&mockMessageCommand{
			MessageContextMenuCommandConfig: MessageContextMenuCommandConfig{Name: "msg_help"},
		},
	)
	assert.Panics(t, func() {
		r.MustRegisterMessageContextMenuCommand(
			&mockMessageCommand{
				MessageContextMenuCommandConfig: MessageContextMenuCommandConfig{Name: "msg_help"},
			},
		)
	})

	r.MustRegisterButtonCommand(
		&mockButtonCommand{ButtonCommandConfig: ButtonCommandConfig{CustomID: "btn_help"}},
	)
	assert.Panics(t, func() {
		r.MustRegisterButtonCommand(
			&mockButtonCommand{ButtonCommandConfig: ButtonCommandConfig{CustomID: "btn_help"}},
		)
	})

	r.MustRegisterSelectMenuCommand(
		&mockSelectMenuCommand{
			SelectMenuCommandConfig: SelectMenuCommandConfig{CustomID: "menu_help"},
		},
	)
	assert.Panics(t, func() {
		r.MustRegisterSelectMenuCommand(
			&mockSelectMenuCommand{
				SelectMenuCommandConfig: SelectMenuCommandConfig{CustomID: "menu_help"},
			},
		)
	})

	r.MustRegisterModalCommand(
		&mockModalCommand{ModalCommandConfig: ModalCommandConfig{CustomID: "modal_help"}},
	)
	assert.Panics(t, func() {
		r.MustRegisterModalCommand(
			&mockModalCommand{ModalCommandConfig: ModalCommandConfig{CustomID: "modal_help"}},
		)
	})
}

func TestRegistry_WordLimitValidation(t *testing.T) {
	r := New()

	cmdTooLong := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "one two three four"}}
	assert.ErrorIs(t, r.RegisterCommand(cmdTooLong), ErrCommandNameTooLong)

	cmd3Words := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "one two three"}}
	require.NoError(t, r.RegisterCommand(cmd3Words))

	cmdWithLongAlias := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:    "valid",
			Aliases: []string{"one two three four"},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(cmdWithLongAlias), ErrAliasTooLong)

	assert.ErrorIs(t, r.RegisterAlias("one two three four", "one two three"), ErrAliasTooLong)
}

func TestRegistry_ArgValidation(t *testing.T) {
	r := New()

	reqWithDefault := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "invalid-default",
			Args: []ArgDef{
				{Name: "opt", Type: ArgTypeString, Required: true, Default: "fallback"},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(reqWithDefault), ErrRequiredArgWithDefault)

	optBeforeReq := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "invalid-order",
			Args: []ArgDef{
				{Name: "first_optional", Type: ArgTypeString, Required: false},
				{Name: "second_required", Type: ArgTypeString, Required: true},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(optBeforeReq), ErrOptionalArgBeforeRequired)

	endlessNotLast := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "invalid-endless",
			Args: []ArgDef{
				{Name: "endless_text", Type: ArgTypeString, Endless: true},
				{Name: "another_arg", Type: ArgTypeString},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(endlessNotLast), ErrEndlessArgNotLast)

	emptyArgName := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "empty-arg-name",
			Args: []ArgDef{
				{Name: "   ", Type: ArgTypeString},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(emptyArgName), ErrEmptyArgName)

	dupArgName := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "dup-arg",
			Args: []ArgDef{
				{Name: "target", Type: ArgTypeUser},
				{Name: "TARGET", Type: ArgTypeString},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(dupArgName), ErrDuplicateArgName)

	// validation propagates to sub-commands
	invalidSubCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "bad-child",
			Args: []ArgDef{
				{Name: "bad_arg", Type: ArgTypeString, Required: true, Default: "error"},
			},
		},
	}
	parentGroup := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:        "parent",
			Subcommands: []ChatCommand{invalidSubCmd},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(parentGroup), ErrRequiredArgWithDefault)
}

func TestRegistry_FlagValidation(t *testing.T) {
	r := New()

	emptyFlagName := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "empty-flag",
			Flags: []FlagDef{
				{Name: "   ", Type: FlagTypeString},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(emptyFlagName), ErrEmptyFlagName)

	dupFlagName := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "dup-flag",
			Flags: []FlagDef{
				{Name: "silent", Short: "s", Type: FlagTypeBool},
				{Name: "SILENT", Short: "x", Type: FlagTypeBool},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(dupFlagName), ErrDuplicateFlagName)

	dupShortFlag := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "dup-short-flag",
			Flags: []FlagDef{
				{Name: "silent", Short: "s", Type: FlagTypeBool},
				{Name: "slow", Short: "S", Type: FlagTypeBool},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(dupShortFlag), ErrDuplicateFlagName)
}

func TestRegistry_InvalidLengthRange(t *testing.T) {
	r := New()

	minLen := 10
	maxLen := 5
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "bad-length-range",
			Args: []ArgDef{
				{Name: "text", Type: ArgTypeString, MinLength: &minLen, MaxLength: &maxLen},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(cmd), ErrInvalidLengthRange)

	negativeMin := -1
	cmd2 := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "neg-min-length",
			Args: []ArgDef{
				{Name: "text", Type: ArgTypeString, MinLength: &negativeMin},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(cmd2), ErrInvalidLengthRange)
}

func TestRegistry_InvalidValueRange(t *testing.T) {
	r := New()

	minVal := 100.0
	maxVal := 10.0
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "bad-value-range",
			Args: []ArgDef{
				{Name: "amount", Type: ArgTypeInt, MinValue: &minVal, MaxValue: &maxVal},
			},
		},
	}
	assert.ErrorIs(t, r.RegisterCommand(cmd), ErrInvalidValueRange)
}

func TestRegistry_RegistrationErrorMessage(t *testing.T) {
	err := &CommandRegistrationError{
		Err:     ErrCommandAlreadyExists,
		Command: "ping",
	}
	assert.Contains(t, err.Error(), "ping")
	assert.ErrorIs(t, err, ErrCommandAlreadyExists)

	errWithAlias := &CommandRegistrationError{
		Err:   ErrAliasAlreadyExists,
		Alias: "p",
	}
	assert.Contains(t, errWithAlias.Error(), "p")

	errWithArg := &CommandRegistrationError{
		Err:      ErrDuplicateArgName,
		Command:  "ban",
		Argument: "target",
	}
	assert.Contains(t, errWithArg.Error(), "ban")
	assert.Contains(t, errWithArg.Error(), "target")
}

func TestRegistry_StringLengthConstraints(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	minLen, maxLen := 3, 6
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "tag",
			Args: []ArgDef{
				{Name: "name", Type: ArgTypeString, MinLength: &minLen, MaxLength: &maxLen},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	tests := []struct {
		input   string
		invalid bool
	}{
		{"!tag ab", true},          // below MinLength
		{"!tag toolongname", true}, // above MaxLength
		{"!tag golang", false},     // within range
	}

	for _, tt := range tests {
		handler.invalidArgCalled = false
		r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
			Message: wumpkit.Message{
				Content: tt.input,
				Author:  wumpkit.User{ID: 1},
			},
		})
		assert.Equal(t, tt.invalid, handler.invalidArgCalled, "input: %q", tt.input)
	}
}

func TestRegistry_IntLengthConstraints(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	exactLen := 4
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "pin",
			Args: []ArgDef{
				{Name: "code", Type: ArgTypeInt, MinLength: &exactLen, MaxLength: &exactLen},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	tests := []struct {
		input   string
		invalid bool
	}{
		{"!pin 123", true},   // too short
		{"!pin 12345", true}, // too long
		{"!pin 1234", false}, // exact
	}

	for _, tt := range tests {
		handler.invalidArgCalled = false
		r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
			Message: wumpkit.Message{
				Content: tt.input,
				Author:  wumpkit.User{ID: 1},
			},
		})
		assert.Equal(t, tt.invalid, handler.invalidArgCalled, "input: %q", tt.input)
	}
}

func TestRegistry_ValueConstraints(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	minVal, maxVal := 13.0, 120.0
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "age",
			Args: []ArgDef{
				{Name: "years", Type: ArgTypeInt, MinValue: &minVal, MaxValue: &maxVal},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	tests := []struct {
		input   string
		invalid bool
	}{
		{"!age 5", true},   // below MinValue
		{"!age 150", true}, // above MaxValue
		{"!age 25", false}, // within range
	}

	for _, tt := range tests {
		handler.invalidArgCalled = false
		r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
			Message: wumpkit.Message{
				Content: tt.input,
				Author:  wumpkit.User{ID: 1},
			},
		})
		assert.Equal(t, tt.invalid, handler.invalidArgCalled, "input: %q", tt.input)
	}
}

func TestRegistry_ChoiceConstraints(t *testing.T) {
	handler := &trackingErrorHandler{}
	r := New(WithPrefix("!"), WithCommandErrorHandler(handler))

	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name: "theme",
			Args: []ArgDef{
				{
					Name: "mode",
					Type: ArgTypeString,
					Choices: []Choice{
						{Name: "Dark", Value: "dark"},
						{Name: "Light", Value: "light"},
					},
				},
			},
		},
	}
	require.NoError(t, r.RegisterCommand(cmd))

	tests := []struct {
		input   string
		invalid bool
	}{
		{"!theme blue", true},   // not a valid choice
		{"!theme dark", false},  // valid value match
		{"!theme Light", false}, // valid name match (case-insensitive)
	}

	for _, tt := range tests {
		handler.invalidArgCalled = false
		r.HandleMessage(context.Background(), wumpkit.MessageCreateEvent{
			Message: wumpkit.Message{
				Content: tt.input,
				Author:  wumpkit.User{ID: 1},
			},
		})
		assert.Equal(t, tt.invalid, handler.invalidArgCalled, "input: %q", tt.input)
	}
}

func BenchmarkGetCommand(b *testing.B) {
	r := New()
	cmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:    "ban",
			Aliases: []string{"b", "banish"},
		},
	}
	_ = r.RegisterCommand(cmd)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.GetCommand("ban")
	}
}
