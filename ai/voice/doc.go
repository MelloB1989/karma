// Package voice provides voice-agent orchestration on top of Karma AI text agents.
//
// It keeps text reasoning in the existing ai package (including MCP and Go function
// tools) while swapping the I/O layer to STT + TTS providers.
//
// Provider-specific note: ElevenLabs realtime STT expects raw PCM/uLaw audio chunks
// (for example pcm_16000) over websocket, not container formats like webm/ogg.
package voice
