// Package voice provides voice-agent orchestration on top of Karma AI text agents.
//
// It keeps text reasoning in the existing ai package (including MCP and Go function
// tools) while swapping the I/O layer to STT + TTS providers.
//
// Vapi is modeled separately as a call agent because Vapi owns the full voice
// pipeline inside the call instead of exposing standalone transcription and
// synthesis operations.
//
// Provider-specific note: ElevenLabs realtime STT expects raw PCM/uLaw audio chunks
// (for example pcm_16000) over websocket, not container formats like webm/ogg.
package voice
