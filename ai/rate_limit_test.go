package ai

import (
	"errors"
	"testing"
)

func TestInstanceRateLimitErrorsWhenExceeded(t *testing.T) {
	kai := NewKarmaAI(GPT4oMini, OpenAI, WithRateLimit(1, RateLimitBehaviorError))

	if err := kai.enforceRateLimit(); err != nil {
		t.Fatalf("first request should pass: %v", err)
	}

	err := kai.enforceRateLimit()
	if err == nil {
		t.Fatal("second request should be rate limited")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestGlobalRateLimitIsSharedAcrossInstances(t *testing.T) {
	ClearGlobalRateLimits()
	defer ClearGlobalRateLimits()

	SetGlobalRateLimit(OpenAI, 1, RateLimitBehaviorError)

	first := NewKarmaAI(GPT4oMini, OpenAI)
	second := NewKarmaAI(GPT4oMini, OpenAI)

	if err := first.enforceRateLimit(); err != nil {
		t.Fatalf("first instance should pass: %v", err)
	}

	err := second.enforceRateLimit()
	if err == nil {
		t.Fatal("second instance should share and hit global rate limit")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestGlobalRateLimitIsScopedByProvider(t *testing.T) {
	ClearGlobalRateLimits()
	defer ClearGlobalRateLimits()

	SetGlobalRateLimit(OpenAI, 1, RateLimitBehaviorError)

	openAI := NewKarmaAI(GPT4oMini, OpenAI)
	gemini := NewKarmaAI(Gemini25Flash, Google)

	if err := openAI.enforceRateLimit(); err != nil {
		t.Fatalf("openai request should pass: %v", err)
	}
	if err := gemini.enforceRateLimit(); err != nil {
		t.Fatalf("google request should not use openai global limit: %v", err)
	}
}
