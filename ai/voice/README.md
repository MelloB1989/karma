# ai/voice

`ai/voice` adds voice I/O on top of your existing Karma AI setup.

- Text reasoning still goes through your configured `ai.KarmaAI` instance.
- MCP, Go function tools, analytics, and provider/model behavior stay intact.
- Only speech I/O (STT + TTS) is provider-specific.

## Providers

- `openai` (SDK: `github.com/openai/openai-go/v3`)
- `together` (SDK: `github.com/togethercomputer/together-go`)
- `elevenlabs` (WebSocket APIs)

## Quick Start

```go
package main

import (
    "context"

    "github.com/MelloB1989/karma/ai"
    "github.com/MelloB1989/karma/ai/voice"
)

func main() {
    textAI := ai.NewKarmaAI(
        ai.GPT4oMini,
        ai.OpenAI,
        ai.WithToolsEnabled(),
        // ai.SetGoFunctionTools(...)
        // ai.SetMCPServers(...)
    )

    agent, _ := voice.NewOpenAIAgent(textAI)

    _, _ = agent.Converse(context.Background(), voice.ConverseRequest{
        Audio: []byte("...audio bytes..."),
    })
}
```

## Model Selection (Hard-Coded Catalog)

Models are not loaded from env. Use provider model constants, similar to `ai`:

```go
agent, _ := voice.NewOpenAIAgent(
    textAI,
    voice.WithOpenAIModels(
        voice.OpenAIGPT4oMiniTranscribe,
        voice.OpenAIGPT4oMiniTTS,
    ),
)
```

You can inspect available models:

```go
voice.GetAvailableSTTModels(voice.ProviderOpenAI)
voice.GetAvailableTTSModels(voice.ProviderOpenAI)
```

## WebSocket Utility

Use `voice.WebSocketHandler` for message loops and typed send helpers:

```go
handler, _ := voice.NewWebSocketHandler(conn,
    voice.WithWSMessageHandler(func(ctx context.Context, msg voice.WSMessage) error {
        // msg.Data, msg.JSON
        return nil
    }),
)

_ = handler.SendJSON(ctx, map[string]any{"event": "ready"})
_ = handler.Run(ctx)
```

## Environment Defaults

- OpenAI: `OPENAI_API_KEY` (fallback `OPENAI_KEY`)
- Together: `TOGETHER_API_KEY`
- ElevenLabs: `ELEVENLABS_API_KEY`

Optional defaults can be customized with `voice.WithOpenAIConfig`,
`voice.WithTogetherConfig`, and `voice.WithElevenLabsConfig`.
