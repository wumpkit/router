package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/wumpkit-go/router"
)

// cooldownWithStore creates a middleware enforcing rate limits via a custom CooldownStore
func CooldownWithStore(store router.CooldownStore, customHandler ...router.CommandErrorHandler) router.Middleware {
	var eh router.CommandErrorHandler
	if len(customHandler) > 0 {
		eh = customHandler[0]
	}

	return func(next router.CommandHandler) router.CommandHandler {
		return func(ctx router.Context) error {
			cmd := ctx.Command()
			if cmd == nil || store == nil {
				return next(ctx)
			}

			cfg := router.ExtractCommonConfig(cmd)
			rules := cfg.Cooldowns
			if len(rules) == 0 {
				return next(ctx)
			}

			for idx, rule := range rules {
				key := getCooldownBucketKey(cfg.Name, rule.Scope, idx, ctx)
				if key == "" {
					continue
				}

				limit := rule.Limit
				if limit <= 0 {
					limit = 1
				}

				allowed, remaining, err := store.Allow(key, limit, rule.Duration)
				if err != nil {
					return err
				}
				if !allowed {
					if eh != nil {
						eh.OnCooldown(ctx, remaining, rule.Scope)
						return nil
					}
					return &router.CooldownError{
						Remaining: remaining,
						Scope:     rule.Scope,
					}
				}
			}

			return next(ctx)
		}
	}
}

// Cooldown creates a middleware enforcing command execution rate limits using the Generic Cell Rate Algorithm (GCRA).
// When a cooldown is exceeded, it notifies the customHandler (if provided) or returns a *router.CooldownError.
func Cooldown(customHandler ...router.CommandErrorHandler) router.Middleware {
	var eh router.CommandErrorHandler
	if len(customHandler) > 0 {
		eh = customHandler[0]
	}

	var (
		// buckets stores theoretical arrival time (tat) in unix nanoseconds for each active bucket
		buckets   = make(map[string]int64)
		mu        sync.Mutex
		opCount   int
		lastSweep = time.Now().UnixNano()
	)

	return func(next router.CommandHandler) router.CommandHandler {
		return func(ctx router.Context) error {
			cmd := ctx.Command()
			if cmd == nil {
				return next(ctx)
			}

			cfg := router.ExtractCommonConfig(cmd)
			rules := cfg.Cooldowns
			if len(rules) == 0 {
				return next(ctx)
			}

			nowNano := time.Now().UnixNano()
			mu.Lock()

			// periodically sweep expired keys based on operation count or time elapsed
			opCount++
			if opCount > 512 || (nowNano-lastSweep > int64(2*time.Minute) && len(buckets) > 0) {
				opCount = 0
				lastSweep = nowNano
				for k, tat := range buckets {
					if tat < nowNano {
						delete(buckets, k)
					}
				}
			}

			// fast-path for single rule with 0 heap allocations
			if len(rules) == 1 {
				rule := rules[0]
				if rule.Duration > 0 {
					emissionInterval, burstTolerance := ruleParams(rule)
					bucketKey := getCooldownBucketKey(cfg.Name, rule.Scope, 0, ctx)
					nextTAT, remaining, exceeded := evalGCRA(nowNano, emissionInterval, burstTolerance, buckets[bucketKey])

					if exceeded {
						mu.Unlock()
						if eh != nil {
							eh.OnCooldown(ctx, remaining, rule.Scope)
							return nil
						}
						return &router.CooldownError{
							Remaining: remaining,
							Scope:     rule.Scope,
						}
					}

					buckets[bucketKey] = nextTAT
				}
				mu.Unlock()
				return next(ctx)
			}

			// multi-rule: verify all rules pass before updating any tats
			var (
				stackKeys  [4]string
				stackTATs  [4]int64
				validCount int
			)

			for ruleIdx, rule := range rules {
				if rule.Duration <= 0 {
					continue
				}

				emissionInterval, burstTolerance := ruleParams(rule)
				bucketKey := getCooldownBucketKey(cfg.Name, rule.Scope, ruleIdx, ctx)
				nextTAT, remaining, exceeded := evalGCRA(nowNano, emissionInterval, burstTolerance, buckets[bucketKey])

				if exceeded {
					mu.Unlock()
					if eh != nil {
						eh.OnCooldown(ctx, remaining, rule.Scope)
						return nil
					}
					return &router.CooldownError{
						Remaining: remaining,
						Scope:     rule.Scope,
					}
				}

				if validCount < 4 {
					stackKeys[validCount] = bucketKey
					stackTATs[validCount] = nextTAT
					validCount++
				} else {
					buckets[bucketKey] = nextTAT
				}
			}

			// commit updated tats for multi-rule stack keys
			for i := 0; i < validCount; i++ {
				buckets[stackKeys[i]] = stackTATs[i]
			}

			mu.Unlock()
			return next(ctx)
		}
	}
}

func ruleParams(rule router.CooldownRule) (int64, int64) {
	limit := rule.Limit
	if limit <= 0 {
		limit = 1
	}
	emissionInterval := int64(rule.Duration / time.Duration(limit))
	burstTolerance := int64(rule.Duration)
	return emissionInterval, burstTolerance
}

func evalGCRA(nowNano, emissionInterval, burstTolerance, storedTAT int64) (int64, time.Duration, bool) {
	tat := max(storedTAT, nowNano)
	nextTAT := tat + emissionInterval
	if nextTAT-nowNano > burstTolerance {
		remaining := time.Duration(nextTAT - nowNano - burstTolerance)
		return nextTAT, remaining, true
	}
	return nextTAT, 0, false
}

func getCooldownBucketKey(cmdName string, scope router.CooldownScope, ruleIdx int, ctx router.Context) string {
	idxStr := strconv.Itoa(ruleIdx)
	switch scope {
	case router.CooldownScopeGuild:
		gID := "dm"
		if g := ctx.GuildID(); g != nil {
			gID = g.String()
		}
		return cmdName + ":g:" + gID + ":" + idxStr
	case router.CooldownScopeChannel:
		return cmdName + ":c:" + ctx.ChannelID().String() + ":" + idxStr
	case router.CooldownScopeMember:
		gID := "dm"
		if g := ctx.GuildID(); g != nil {
			gID = g.String()
		}
		return cmdName + ":m:" + gID + ":" + ctx.Author().ID.String() + ":" + idxStr
	case router.CooldownScopeGlobal:
		return cmdName + ":global:" + idxStr
	default: // CooldownScopeUser
		return cmdName + ":u:" + ctx.Author().ID.String() + ":" + idxStr
	}
}
