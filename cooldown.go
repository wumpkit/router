package router

import (
	"fmt"
	"time"
)

// CooldownScope defines the rate limiting boundary for a cooldown rule.
type CooldownScope int

const (
	// CooldownScopeUser limits command usage per individual user.
	CooldownScopeUser CooldownScope = iota
	// CooldownScopeGuild limits command usage per guild.
	CooldownScopeGuild
	// CooldownScopeChannel limits command usage per channel.
	CooldownScopeChannel
	// CooldownScopeMember limits command usage per user within a specific guild.
	CooldownScopeMember
	// CooldownScopeGlobal limits command usage across the entire application.
	CooldownScopeGlobal
)

// String returns the string representation of a CooldownScope.
func (s CooldownScope) String() string {
	switch s {
	case CooldownScopeUser:
		return "user"
	case CooldownScopeGuild:
		return "guild"
	case CooldownScopeChannel:
		return "channel"
	case CooldownScopeMember:
		return "member"
	case CooldownScopeGlobal:
		return "global"
	default:
		return "unknown"
	}
}

// CooldownRule defines a rate limiting rule for a command.
type CooldownRule struct {
	Duration time.Duration
	Scope    CooldownScope
	Limit    int
}

// CooldownError represents an error returned when a command is invoked while on cooldown.
type CooldownError struct {
	Remaining time.Duration
	Scope     CooldownScope
}

func (e *CooldownError) Error() string {
	return fmt.Sprintf("command is on cooldown for %v (scope: %v)", e.Remaining, e.Scope)
}

// cooldownStore manages custom rate limit storage
type CooldownStore interface {
	Allow(key string, limit int, duration time.Duration) (allowed bool, remaining time.Duration, err error)
}
