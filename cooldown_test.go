package router

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCooldownError_Error(t *testing.T) {
	err := &CooldownError{
		Remaining: 3500 * time.Millisecond,
		Scope:     CooldownScopeUser,
	}
	msg := err.Error()
	assert.Contains(t, msg, "3.5s")
	assert.Contains(t, msg, "user")
}

func TestCooldownScope_String(t *testing.T) {
	tests := []struct {
		scope    CooldownScope
		expected string
	}{
		{CooldownScopeUser, "user"},
		{CooldownScopeGuild, "guild"},
		{CooldownScopeChannel, "channel"},
		{CooldownScopeMember, "member"},
		{CooldownScopeGlobal, "global"},
		{CooldownScope(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.scope.String())
	}
}
