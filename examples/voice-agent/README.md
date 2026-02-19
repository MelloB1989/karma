# Voice AI Agent

A real-time voice assistant built with the `karma/ai/voice` package.  
It records audio in the browser, streams it to a Go WebSocket server that runs
STT → LLM → TTS, and plays the synthesised reply back automatically.

```
Browser (MediaRecorder)
  └─ binary audio ──► Go WebSocket ──► Provider STT (OpenAI/Together/ElevenLabs)
                                    ──► Groq Llama 3.3 70B (text AI)
                                    ──► Provider TTS (OpenAI/Together/ElevenLabs)
  ◄─ JSON { transcript, text, audio } ──┘
```

---

## Features

- **Push-to-talk** — hold the 🎤 button (or press `Space`) to record
- **Live waveform** — canvas visualiser while recording
- **Streaming-safe** — `voice.WebSocketHandler` handles read / write deadlines
- **Session memory** — conversation history is kept per WebSocket connection
- **Reset** — clear history with the ↺ button or send `{"event":"reset"}`
- **Auto-reconnect** — browser reconnects on disconnect with 2 s back-off
- **Touch-friendly** — pointer + touch events supported

---

## Prerequisites

| Requirement | Notes |
|---|---|
| Go ≥ 1.21 | part of the karma module |
| `GROQ_API_KEY` env var | used for text reasoning in `ai.KarmaAI` |
| `KARMA_VOICE_PROVIDER` (optional) | `openai`, `together`, or `elevenlabs` (default `elevenlabs`) |
| `KARMA_VOICE_STT_LANGUAGE` (optional) | STT language hint (default `en` for ElevenLabs, unset for others) |
| provider API key(s) | `OPENAI_API_KEY`, `TOGETHER_API_KEY`, or `ELEVENLABS_API_KEY` |
| provider voice ID (when required) | e.g. `ELEVENLABS_VOICE_ID` for ElevenLabs TTS |
| A modern browser | Chrome, Edge, Firefox, Safari 16+ |

---

## Running

```bash
# from the repo root
cd examples/voice-agent

# export keys (or put them in a .env and source it)
export GROQ_API_KEY=gsk-...
export KARMA_VOICE_PROVIDER=openai
export OPENAI_API_KEY=sk-...
export KARMA_VOICE_STT_LANGUAGE=en

# or ElevenLabs
# export KARMA_VOICE_PROVIDER=elevenlabs
# export ELEVENLABS_API_KEY=...
# export ELEVENLABS_VOICE_ID=...

go run .
```

Then open **http://localhost:8080** in your browser.

Set a custom port with `PORT=9090 go run .`

---

## How it works

### Backend (`main.go`)

1. `http.FileServer` serves `static/index.html`.
2. `/ws` upgrades to a WebSocket and creates:
   - A `*ai.KarmaAI` text brain (Llama 3.3 70B on Groq, concise system prompt).
   - A `*voice.Agent` selected from `KARMA_VOICE_PROVIDER`.
   - A `*voice.WebSocketHandler` with a message callback.
3. On each binary message the handler calls `agent.Converse`, which:
   - Transcribes audio with the selected provider.
   - Runs the transcript through the text AI (with full conversation history).
   - Synthesises the reply with the selected provider.
4. The JSON response is sent back:
   ```json
   {
     "event":      "response",
     "transcript": "what the user said",
     "text":       "what the AI replied",
     "audio":      "<base64 mp3>",
     "format":     "mp3"
   }
   ```

### Frontend (`static/index.html`)

- Connects to `ws[s]://<host>/ws` on load.
- `mousedown` / `touchstart` on the mic button → `MediaRecorder.start()`.
- `mouseup` / `touchend` → `MediaRecorder.stop()` and sends provider-specific audio:
  - `webm` for OpenAI/Together
  - `pcm16_16000` for ElevenLabs realtime STT
- Receives JSON responses, appends chat bubbles, and plays audio via the Web Audio API.
- `Space` key works as a keyboard shortcut for push-to-talk.

---

## WebSocket message protocol

| Direction | Format | Meaning |
|---|---|---|
| client → server | **binary** | `webm` (OpenAI/Together) or mono PCM16LE 16 kHz (ElevenLabs) |
| client → server | `{"event":"ping"}` | Keep-alive |
| client → server | `{"event":"reset"}` | Clear conversation history |
| server → client | `{"event":"ready","provider":"...","input_audio_format":"..."}` | Server is ready + expected audio format |
| server → client | `{"event":"processing"}` | Server received audio, working |
| server → client | `{"event":"response", ...}` | Full turn result |
| server → client | `{"event":"reset_ok"}` | History cleared |
| server → client | `{"event":"pong"}` | Ping reply |
| server → client | `{"event":"error","message":"..."}` | Error detail |

---

## Switching providers

The voice agent supports OpenAI, Together, and ElevenLabs out of the box.
Switch providers with env vars:

```bash
# OpenAI
export KARMA_VOICE_PROVIDER=openai
export OPENAI_API_KEY=sk-...

# Together
export KARMA_VOICE_PROVIDER=together
export TOGETHER_API_KEY=...

# ElevenLabs
export KARMA_VOICE_PROVIDER=elevenlabs
export ELEVENLABS_API_KEY=...
export ELEVENLABS_VOICE_ID=...
```

---

## Customising the AI personality

Edit the `WithSystemMessage` call in `main.go`:

```go
ai.WithSystemMessage("You are a snarky pirate assistant. Respond in exactly one sentence."),
```

---

## Project structure

```
examples/voice-agent/
├── main.go          ← Go WebSocket server + voice agent wiring
├── README.md
└── static/
    └── index.html   ← Self-contained HTML/JS/CSS frontend
```
