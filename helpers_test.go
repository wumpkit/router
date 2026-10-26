package router

import (
	"context"
	"time"

	"github.com/wumpkit-go/wumpkit"
)

type mockCommand struct {
	ChatCommandConfig
	runFunc func(ctx Context) error
}

func (m *mockCommand) Run(ctx Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	return nil
}

type mockAutoCommand struct {
	mockCommand
	autocompleteCalled bool
}

func (a *mockAutoCommand) Autocomplete(ctx context.Context, interaction *wumpkit.AutoCompleteInteraction) error {
	a.autocompleteCalled = true
	return nil
}

type mockUserCommand struct {
	UserContextMenuCommandConfig
	runFunc func(ctx Context) error
}

func (m *mockUserCommand) Run(ctx Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	return nil
}

type mockMessageCommand struct {
	MessageContextMenuCommandConfig
	runFunc func(ctx Context) error
}

func (m *mockMessageCommand) Run(ctx Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	return nil
}

type mockButtonCommand struct {
	ButtonCommandConfig
	runFunc func(ctx Context) error
}

func (b *mockButtonCommand) Run(ctx Context) error {
	if b.runFunc != nil {
		return b.runFunc(ctx)
	}
	return nil
}

type mockSelectMenuCommand struct {
	SelectMenuCommandConfig
	runFunc func(ctx Context) error
}

func (s *mockSelectMenuCommand) Run(ctx Context) error {
	if s.runFunc != nil {
		return s.runFunc(ctx)
	}
	return nil
}

type mockModalCommand struct {
	ModalCommandConfig
	runFunc func(ctx Context) error
}

func (m *mockModalCommand) Run(ctx Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	return nil
}

type trackingErrorHandler struct {
	DefaultCommandErrorHandler
	missingArgCalled      bool
	missingArgName        string
	invalidArgCalled      bool
	invalidArgReceived    string
	userPermMissingCalled bool
	userPermMissingValue  wumpkit.Permissions
	appPermMissingCalled  bool
	appPermMissingValue   wumpkit.Permissions
	contextExcludedCalled bool
	contextExcludedValue  ExcludedIn
	cooldownCalled        bool
	cooldownRemaining     time.Duration
	cooldownScope         CooldownScope
	nsfwRequiredCalled    bool
	executionErrorCalled  bool
	lastErr               error
}

func (t *trackingErrorHandler) OnMissingArgument(ctx Context, name string, expected string) {
	t.missingArgCalled = true
	t.missingArgName = name
}

func (t *trackingErrorHandler) OnInvalidArgument(ctx Context, name string, expected string, received string) {
	t.invalidArgCalled = true
	t.invalidArgReceived = received
}

func (t *trackingErrorHandler) OnUserPermissionMissing(ctx Context, missing wumpkit.Permissions) {
	t.userPermMissingCalled = true
	t.userPermMissingValue = missing
}

func (t *trackingErrorHandler) OnAppPermissionMissing(ctx Context, missing wumpkit.Permissions) {
	t.appPermMissingCalled = true
	t.appPermMissingValue = missing
}

func (t *trackingErrorHandler) OnCooldown(ctx Context, remaining time.Duration, scope CooldownScope) {
	t.cooldownCalled = true
	t.cooldownRemaining = remaining
	t.cooldownScope = scope
}

func (t *trackingErrorHandler) OnContextExcluded(ctx Context, excluded ExcludedIn) {
	t.contextExcludedCalled = true
	t.contextExcludedValue = excluded
}

func (t *trackingErrorHandler) OnNSFWRequired(ctx Context) {
	t.nsfwRequiredCalled = true
}

func (t *trackingErrorHandler) OnCommandError(ctx Context, cmd any, err error) {
	t.executionErrorCalled = true
	t.lastErr = err
}

type mockLogger struct {
	entries []string
}

func (m *mockLogger) Debug(msg string, keysAndValues ...any) {
	m.entries = append(m.entries, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, keysAndValues ...any) {
	m.entries = append(m.entries, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, keysAndValues ...any) {
	m.entries = append(m.entries, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, keysAndValues ...any) {
	m.entries = append(m.entries, "ERROR: "+msg)
}
