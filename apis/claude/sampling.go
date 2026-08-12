package claude

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

// Sampling parameters, applied the way each backend will actually accept them.
//
// Anthropic on Bedrock rejects a request carrying both `temperature` and
// `top_p`: "cannot both be specified for this model". The direct API and most
// proxies tolerate it, so a config that worked everywhere else fails the moment
// it is pointed at Bedrock — and it fails as a 400 on every single call, which
// looks like a broken deployment rather than one surplus field.
//
// Callers should not have to know that. The library sends temperature when it
// has one and falls back to top_p otherwise, so the same SessionConfig works on
// either transport.

// applySampling sets the sampling fields that this model and transport accept.
func applySampling(temp, topP float64, topK int64, thinking bool) (t, p param.Opt[float64], k param.Opt[int64]) {
	// Thinking models take neither: sampling is not meaningful when the model
	// controls its own decoding.
	if thinking {
		return t, p, k
	}
	switch {
	case temp > 0:
		// Temperature wins when both are configured. It is the one callers
		// actually set deliberately; top_p is usually a default they never chose.
		t = param.NewOpt(temp)
	case topP > 0:
		p = param.NewOpt(topP)
	}
	if topK > 0 {
		k = param.NewOpt(topK)
	}
	return t, p, k
}

// sampling returns this client's parameters, ready to assign.
func (cc *ClaudeClient) sampling() (t, p param.Opt[float64], k param.Opt[int64]) {
	return applySampling(cc.Temp, cc.TopP, cc.TopK, cc.isThinkingModel())
}

// applyTo sets the sampling fields on a message request.
func (cc *ClaudeClient) applyTo(m *anthropic.MessageNewParams) {
	m.Temperature, m.TopP, m.TopK = cc.sampling()
}
