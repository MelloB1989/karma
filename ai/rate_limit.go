package ai

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// RateLimitBehavior controls what KarmaAI does when a configured request limit is reached.
type RateLimitBehavior string

const (
	// RateLimitBehaviorWait blocks until a new request can be made.
	RateLimitBehaviorWait RateLimitBehavior = "wait"
	// RateLimitBehaviorError returns ErrRateLimited immediately when the limit is reached.
	RateLimitBehaviorError RateLimitBehavior = "error"
)

// ErrRateLimited is returned when a rate limit is reached and the behavior is RateLimitBehaviorError.
var ErrRateLimited = errors.New("karma ai rate limit exceeded")

// RateLimitError includes the retry duration for callers that want to back off and retry.
type RateLimitError struct {
	Provider   Provider
	Model      string
	Scope      string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e == nil {
		return ErrRateLimited.Error()
	}
	return fmt.Sprintf("%s: provider=%s model=%s scope=%s retry_after=%s", ErrRateLimited, e.Provider, e.Model, e.Scope, e.RetryAfter.Round(time.Millisecond))
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

// RateLimitConfig configures requests-per-minute limiting for a KarmaAI instance.
type RateLimitConfig struct {
	RequestsPerMinute int               `json:"requests_per_minute"`
	Behavior          RateLimitBehavior `json:"behavior"`

	limiter *rateLimiter
}

type globalRateLimitKey struct {
	provider Provider
}

var globalRateLimits = struct {
	sync.RWMutex
	limits map[globalRateLimitKey]*RateLimitConfig
}{
	limits: make(map[globalRateLimitKey]*RateLimitConfig),
}

// WithRateLimit applies an instance-local requests-per-minute limit.
func WithRateLimit(requestsPerMinute int, behavior RateLimitBehavior) Option {
	return func(kai *KarmaAI) {
		kai.RateLimit = newRateLimitConfig(requestsPerMinute, behavior)
	}
}

// SetGlobalRateLimit applies a requests-per-minute limit shared by all KarmaAI instances for a provider.
func SetGlobalRateLimit(provider Provider, requestsPerMinute int, behavior RateLimitBehavior) {
	globalRateLimits.Lock()
	defer globalRateLimits.Unlock()
	globalRateLimits.limits[globalRateLimitKey{provider: provider}] = newRateLimitConfig(requestsPerMinute, behavior)
}

// ClearGlobalRateLimit removes the shared rate limit for a provider.
func ClearGlobalRateLimit(provider Provider) {
	globalRateLimits.Lock()
	defer globalRateLimits.Unlock()
	delete(globalRateLimits.limits, globalRateLimitKey{provider: provider})
}

// ClearGlobalRateLimits removes all shared provider limits.
func ClearGlobalRateLimits() {
	globalRateLimits.Lock()
	defer globalRateLimits.Unlock()
	globalRateLimits.limits = make(map[globalRateLimitKey]*RateLimitConfig)
}

func newRateLimitConfig(requestsPerMinute int, behavior RateLimitBehavior) *RateLimitConfig {
	if requestsPerMinute <= 0 {
		return nil
	}
	if behavior == "" {
		behavior = RateLimitBehaviorWait
	}
	return &RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		Behavior:          behavior,
		limiter:           newRateLimiter(requestsPerMinute),
	}
}

func getGlobalRateLimit(provider Provider) *RateLimitConfig {
	globalRateLimits.RLock()
	defer globalRateLimits.RUnlock()
	return globalRateLimits.limits[globalRateLimitKey{provider: provider}]
}

func (kai *KarmaAI) enforceRateLimit() error {
	provider := kai.Model.GetModelProvider()
	model := kai.Model.GetModelString()
	limits := []rateLimitEntry{
		{config: kai.RateLimit, scope: "instance"},
		{config: getGlobalRateLimit(provider), scope: "global"},
	}
	activeLimits := limits[:0]
	for _, limit := range limits {
		if limit.config != nil && limit.config.limiter != nil {
			activeLimits = append(activeLimits, limit)
		}
	}
	if len(activeLimits) == 0 {
		return nil
	}

	for {
		for _, limit := range activeLimits {
			limit.config.limiter.mu.Lock()
		}

		now := time.Now()
		var blocked *rateLimitEntry
		var waitFor time.Duration
		for i := range activeLimits {
			limit := &activeLimits[i]
			limiter := limit.config.limiter
			limiter.purgeLocked(now)
			retryAfter := limiter.retryAfterLocked(now)
			if retryAfter > 0 && (blocked == nil || retryAfter > waitFor || limit.config.Behavior == RateLimitBehaviorError) {
				blocked = limit
				waitFor = retryAfter
			}
		}

		if blocked == nil {
			for _, limit := range activeLimits {
				limit.config.limiter.recordLocked(now)
			}
			for i := len(activeLimits) - 1; i >= 0; i-- {
				activeLimits[i].config.limiter.mu.Unlock()
			}
			return nil
		}

		behavior := blocked.config.Behavior
		scope := blocked.scope
		for i := len(activeLimits) - 1; i >= 0; i-- {
			activeLimits[i].config.limiter.mu.Unlock()
		}
		if behavior == RateLimitBehaviorError {
			return &RateLimitError{
				Provider:   provider,
				Model:      model,
				Scope:      scope,
				RetryAfter: waitFor,
			}
		}
		time.Sleep(waitFor)
	}
}

type rateLimitEntry struct {
	config *RateLimitConfig
	scope  string
}

type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	requests []time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		limit:  limit,
		window: time.Minute,
	}
}

func (rl *rateLimiter) purgeLocked(now time.Time) {
	cutoff := now.Add(-rl.window)
	writeIndex := 0
	for _, requestedAt := range rl.requests {
		if requestedAt.After(cutoff) {
			rl.requests[writeIndex] = requestedAt
			writeIndex++
		}
	}
	rl.requests = rl.requests[:writeIndex]
}

func (rl *rateLimiter) retryAfterLocked(now time.Time) time.Duration {
	if len(rl.requests) < rl.limit {
		return 0
	}

	oldest := rl.requests[0]
	retryAfter := oldest.Add(rl.window).Sub(now)
	if retryAfter < 0 {
		return 0
	}
	return retryAfter
}

func (rl *rateLimiter) recordLocked(now time.Time) {
	rl.requests = append(rl.requests, now)
}
