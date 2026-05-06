package voice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	vapi "github.com/VapiAI/server-sdk-go"
	vapiclient "github.com/VapiAI/server-sdk-go/client"
	"github.com/VapiAI/server-sdk-go/option"
)

// VapiConfig configures Vapi as a call-agent provider.
//
// Vapi handles STT, LLM, and TTS internally, so it intentionally does not
// implement SpeechProvider.
type VapiConfig struct {
	APIKey  string
	BaseURL string

	// AssistantID and PhoneNumberID are reusable defaults for outbound calls.
	AssistantID   string
	PhoneNumberID string

	AssistantName string
	FirstMessage  string
	SystemPrompt  string

	ModelProvider string
	Model         string
	Temperature   *float64
	MaxTokens     *float64

	EmotionRecognitionEnabled *bool
	NumFastTurns              *float64
	ToolIDs                   []string

	VoiceProvider string
	VoiceID       string
	VoiceModel    string

	TranscriberProvider string
	TranscriberModel    string
	TranscriberLanguage string

	MaxDurationSeconds *float64
	ServerURL          string
	ServerHeaders      map[string]any
	Metadata           map[string]any
	CallMetadata       map[string]any

	// Assistant bypasses the convenience builder when you need full SDK coverage.
	Assistant *vapi.CreateAssistantDto

	// Extra body properties are merged by the generated SDK before requests are sent.
	ExtraAssistantBodyProperties map[string]any
	ExtraCallBodyProperties      map[string]any
}

// VapiMessage configures the initial LLM message state for a Vapi assistant.
type VapiMessage struct {
	Role    string
	Content string
}

// VapiAssistantRequest overrides configured Vapi assistant defaults.
type VapiAssistantRequest struct {
	Name         string
	FirstMessage string
	SystemPrompt string
	Messages     []VapiMessage

	ModelProvider string
	Model         string
	Temperature   *float64
	MaxTokens     *float64

	EmotionRecognitionEnabled *bool
	NumFastTurns              *float64
	ToolIDs                   []string

	VoiceProvider string
	VoiceID       string
	VoiceModel    string

	TranscriberProvider string
	TranscriberModel    string
	TranscriberLanguage string

	MaxDurationSeconds *float64
	ServerURL          string
	ServerHeaders      map[string]any
	Metadata           map[string]any

	Assistant               *vapi.CreateAssistantDto
	ExtraBodyProperties     map[string]any
	AdditionalModelMessages []VapiMessage
}

// VapiCallRequest creates an outbound Vapi call.
type VapiCallRequest struct {
	Name string

	AssistantID   string
	Assistant     *vapi.CreateAssistantDto
	UseTransient  bool
	PhoneNumberID string

	CustomerID                     string
	CustomerNumber                 string
	CustomerSipURI                 string
	CustomerName                   string
	CustomerEmail                  string
	CustomerExternalID             string
	CustomerExtension              string
	CustomerNumberE164CheckEnabled *bool

	AssistantOverrides  *vapi.AssistantOverrides
	Metadata            map[string]any
	ExtraBodyProperties map[string]any
}

// CallProvider defines providers that own a complete voice call lifecycle.
type CallProvider interface {
	CreateAssistant(ctx context.Context, req VapiAssistantRequest) (*vapi.Assistant, error)
	CreateOutboundCall(ctx context.Context, req VapiCallRequest) (*vapi.CreateCallsResponse, error)
	GetCall(ctx context.Context, id string) (*vapi.Call, error)
}

// CallAgent orchestrates voice providers that own STT, LLM, and TTS internally.
type CallAgent struct {
	provider Provider
	calls    CallProvider
}

// NewVapiAgent creates a call agent backed by Vapi.
func NewVapiAgent(options ...Option) (*CallAgent, error) {
	return NewCallAgent(ProviderVapi, options...)
}

// NewCallAgent creates a call agent using one of the built-in call providers.
func NewCallAgent(provider Provider, options ...Option) (*CallAgent, error) {
	cfg := defaultConfig()
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}

	callProvider, err := newBuiltInCallProvider(provider, cfg)
	if err != nil {
		return nil, err
	}

	return &CallAgent{
		provider: provider,
		calls:    callProvider,
	}, nil
}

// NewCallAgentWithProvider creates a call agent with a custom call provider.
func NewCallAgentWithProvider(provider Provider, callProvider CallProvider) (*CallAgent, error) {
	if callProvider == nil {
		return nil, errors.New("callProvider is required")
	}
	return &CallAgent{
		provider: provider,
		calls:    callProvider,
	}, nil
}

func newBuiltInCallProvider(provider Provider, cfg Config) (CallProvider, error) {
	switch provider {
	case ProviderVapi:
		return newVapiCallProvider(cfg.Vapi, cfg.HTTPClient)
	default:
		return nil, fmt.Errorf("unsupported call provider: %s", provider)
	}
}

// Provider returns the active call provider name.
func (a *CallAgent) Provider() Provider {
	if a == nil {
		return ""
	}
	return a.provider
}

// CreateAssistant creates a reusable Vapi assistant.
func (a *CallAgent) CreateAssistant(ctx context.Context, req VapiAssistantRequest) (*vapi.Assistant, error) {
	if a == nil || a.calls == nil {
		return nil, errors.New("call agent is not initialized")
	}
	return a.calls.CreateAssistant(ctx, req)
}

// CreateOutboundCall starts an outbound Vapi call.
func (a *CallAgent) CreateOutboundCall(ctx context.Context, req VapiCallRequest) (*vapi.CreateCallsResponse, error) {
	if a == nil || a.calls == nil {
		return nil, errors.New("call agent is not initialized")
	}
	return a.calls.CreateOutboundCall(ctx, req)
}

// GetCall fetches the latest Vapi call state.
func (a *CallAgent) GetCall(ctx context.Context, id string) (*vapi.Call, error) {
	if a == nil || a.calls == nil {
		return nil, errors.New("call agent is not initialized")
	}
	return a.calls.GetCall(ctx, id)
}

type vapiCallProvider struct {
	cfg    VapiConfig
	client *vapiclient.Client
}

func newVapiCallProvider(cfg VapiConfig, httpClient *http.Client) (*vapiCallProvider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("vapi api key is required (set VAPI_API_KEY or use WithVapiAPIKey)")
	}

	opts := []option.RequestOption{option.WithToken(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if httpClient != nil {
		opts = append(opts, option.WithHTTPClient(httpClient))
	}

	return &vapiCallProvider{
		cfg:    cfg,
		client: vapiclient.NewClient(opts...),
	}, nil
}

func (p *vapiCallProvider) CreateAssistant(ctx context.Context, req VapiAssistantRequest) (*vapi.Assistant, error) {
	dto, bodyProperties, err := buildVapiAssistantDTO(p.cfg, req)
	if err != nil {
		return nil, err
	}
	return p.client.Assistants.Create(ctx, dto, vapiBodyOptions(bodyProperties)...)
}

func (p *vapiCallProvider) CreateOutboundCall(ctx context.Context, req VapiCallRequest) (*vapi.CreateCallsResponse, error) {
	dto, bodyProperties, err := buildVapiCallDTO(p.cfg, req)
	if err != nil {
		return nil, err
	}
	return p.client.Calls.Create(ctx, dto, vapiBodyOptions(bodyProperties)...)
}

func (p *vapiCallProvider) GetCall(ctx context.Context, id string) (*vapi.Call, error) {
	if id == "" {
		return nil, errors.New("call id is required")
	}
	return p.client.Calls.Get(ctx, &vapi.GetCallsRequest{Id: id})
}

func buildVapiAssistantDTO(cfg VapiConfig, req VapiAssistantRequest) (*vapi.CreateAssistantDto, map[string]any, error) {
	if req.Assistant != nil {
		return req.Assistant, mergeStringAnyMaps(cfg.ExtraAssistantBodyProperties, req.ExtraBodyProperties), nil
	}
	if cfg.Assistant != nil {
		return cfg.Assistant, mergeStringAnyMaps(cfg.ExtraAssistantBodyProperties, req.ExtraBodyProperties), nil
	}

	merged := mergeVapiAssistantRequest(cfg, req)
	dto := &vapi.CreateAssistantDto{
		Name:               stringPtrIfNotEmpty(merged.Name),
		FirstMessage:       stringPtrIfNotEmpty(merged.FirstMessage),
		MaxDurationSeconds: merged.MaxDurationSeconds,
		Server:             buildVapiServer(merged.ServerURL, merged.ServerHeaders),
		Metadata:           merged.Metadata,
	}

	model, err := buildVapiModel(merged)
	if err != nil {
		return nil, nil, err
	}
	dto.Model = model

	voice, err := buildVapiVoice(merged)
	if err != nil {
		return nil, nil, err
	}
	dto.Voice = voice

	transcriber, err := buildVapiTranscriber(merged)
	if err != nil {
		return nil, nil, err
	}
	dto.Transcriber = transcriber

	return dto, mergeStringAnyMaps(cfg.ExtraAssistantBodyProperties, req.ExtraBodyProperties), nil
}

func buildVapiCallDTO(cfg VapiConfig, req VapiCallRequest) (*vapi.CreateCallDto, map[string]any, error) {
	phoneNumberID := firstNonEmpty(req.PhoneNumberID, cfg.PhoneNumberID)
	if phoneNumberID == "" {
		return nil, nil, errors.New("vapi phone number id is required for outbound calls")
	}

	dto := &vapi.CreateCallDto{
		Name:               stringPtrIfNotEmpty(req.Name),
		PhoneNumberId:      &phoneNumberID,
		AssistantOverrides: req.AssistantOverrides,
		CustomerId:         stringPtrIfNotEmpty(req.CustomerID),
	}

	if req.Assistant != nil {
		dto.Assistant = req.Assistant
	} else {
		assistantID := firstNonEmpty(req.AssistantID, cfg.AssistantID)
		if req.UseTransient || assistantID == "" {
			assistant, _, err := buildVapiAssistantDTO(cfg, VapiAssistantRequest{})
			if err != nil {
				return nil, nil, err
			}
			dto.Assistant = assistant
		} else {
			dto.AssistantId = &assistantID
		}
	}

	if dto.CustomerId == nil {
		customer := buildVapiCustomer(req)
		if customer == nil {
			return nil, nil, errors.New("vapi customer id, customer number, or customer sip uri is required")
		}
		dto.Customer = customer
	}

	bodyProperties := mergeStringAnyMaps(cfg.ExtraCallBodyProperties, req.ExtraBodyProperties)
	metadata := mergeStringAnyMaps(cfg.CallMetadata, req.Metadata)
	if len(metadata) > 0 {
		if bodyProperties == nil {
			bodyProperties = make(map[string]any, 1)
		}
		bodyProperties["metadata"] = metadata
	}

	return dto, bodyProperties, nil
}

func buildVapiModel(req VapiAssistantRequest) (*vapi.CreateAssistantDtoModel, error) {
	if req.ModelProvider == "" && req.Model == "" && req.SystemPrompt == "" && len(req.Messages) == 0 {
		return nil, nil
	}
	if req.ModelProvider == "" {
		return nil, errors.New("vapi model provider is required")
	}
	if req.Model == "" {
		return nil, errors.New("vapi model is required")
	}

	payload := map[string]any{
		"provider": req.ModelProvider,
		"model":    req.Model,
	}
	messages := buildVapiMessages(req)
	if len(messages) > 0 {
		payload["messages"] = messages
	}
	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		payload["maxTokens"] = *req.MaxTokens
	}
	if req.EmotionRecognitionEnabled != nil {
		payload["emotionRecognitionEnabled"] = *req.EmotionRecognitionEnabled
	}
	if req.NumFastTurns != nil {
		payload["numFastTurns"] = *req.NumFastTurns
	}
	if len(req.ToolIDs) > 0 {
		payload["toolIds"] = req.ToolIDs
	}

	var model vapi.CreateAssistantDtoModel
	if err := unmarshalVapiPayload(payload, &model); err != nil {
		return nil, fmt.Errorf("invalid vapi model config: %w", err)
	}
	if _, err := json.Marshal(model); err != nil {
		return nil, fmt.Errorf("invalid vapi model provider %q: %w", req.ModelProvider, err)
	}
	return &model, nil
}

func buildVapiVoice(req VapiAssistantRequest) (*vapi.CreateAssistantDtoVoice, error) {
	if req.VoiceProvider == "" {
		return nil, nil
	}

	payload := map[string]any{"provider": req.VoiceProvider}
	if req.VoiceID != "" {
		payload["voiceId"] = req.VoiceID
	}
	if req.VoiceModel != "" {
		payload["model"] = req.VoiceModel
	}

	var voice vapi.CreateAssistantDtoVoice
	if err := unmarshalVapiPayload(payload, &voice); err != nil {
		return nil, fmt.Errorf("invalid vapi voice config: %w", err)
	}
	if _, err := json.Marshal(voice); err != nil {
		return nil, fmt.Errorf("invalid vapi voice provider %q: %w", req.VoiceProvider, err)
	}
	return &voice, nil
}

func buildVapiTranscriber(req VapiAssistantRequest) (*vapi.CreateAssistantDtoTranscriber, error) {
	if req.TranscriberProvider == "" {
		return nil, nil
	}

	payload := map[string]any{"provider": req.TranscriberProvider}
	if req.TranscriberModel != "" {
		payload["model"] = req.TranscriberModel
	}
	if req.TranscriberLanguage != "" {
		payload["language"] = req.TranscriberLanguage
	}

	var transcriber vapi.CreateAssistantDtoTranscriber
	if err := unmarshalVapiPayload(payload, &transcriber); err != nil {
		return nil, fmt.Errorf("invalid vapi transcriber config: %w", err)
	}
	if _, err := json.Marshal(transcriber); err != nil {
		return nil, fmt.Errorf("invalid vapi transcriber provider %q: %w", req.TranscriberProvider, err)
	}
	return &transcriber, nil
}

func buildVapiMessages(req VapiAssistantRequest) []map[string]string {
	messages := make([]map[string]string, 0, 1+len(req.Messages)+len(req.AdditionalModelMessages))
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}
	for _, msg := range req.Messages {
		if msg.Role == "" || msg.Content == "" {
			continue
		}
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	for _, msg := range req.AdditionalModelMessages {
		if msg.Role == "" || msg.Content == "" {
			continue
		}
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return messages
}

func buildVapiServer(url string, headers map[string]any) *vapi.Server {
	if url == "" && len(headers) == 0 {
		return nil
	}
	return &vapi.Server{
		Url:     stringPtrIfNotEmpty(url),
		Headers: headers,
	}
}

func buildVapiCustomer(req VapiCallRequest) *vapi.CreateCustomerDto {
	if req.CustomerNumber == "" && req.CustomerSipURI == "" {
		return nil
	}
	return &vapi.CreateCustomerDto{
		NumberE164CheckEnabled: req.CustomerNumberE164CheckEnabled,
		Extension:              stringPtrIfNotEmpty(req.CustomerExtension),
		Number:                 stringPtrIfNotEmpty(req.CustomerNumber),
		SipUri:                 stringPtrIfNotEmpty(req.CustomerSipURI),
		Name:                   stringPtrIfNotEmpty(req.CustomerName),
		Email:                  stringPtrIfNotEmpty(req.CustomerEmail),
		ExternalId:             stringPtrIfNotEmpty(req.CustomerExternalID),
	}
}

func mergeVapiConfig(dst *VapiConfig, src VapiConfig) {
	if src.APIKey != "" {
		dst.APIKey = src.APIKey
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if src.AssistantID != "" {
		dst.AssistantID = src.AssistantID
	}
	if src.PhoneNumberID != "" {
		dst.PhoneNumberID = src.PhoneNumberID
	}
	if src.AssistantName != "" {
		dst.AssistantName = src.AssistantName
	}
	if src.FirstMessage != "" {
		dst.FirstMessage = src.FirstMessage
	}
	if src.SystemPrompt != "" {
		dst.SystemPrompt = src.SystemPrompt
	}
	if src.ModelProvider != "" {
		dst.ModelProvider = src.ModelProvider
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.Temperature != nil {
		dst.Temperature = src.Temperature
	}
	if src.MaxTokens != nil {
		dst.MaxTokens = src.MaxTokens
	}
	if src.EmotionRecognitionEnabled != nil {
		dst.EmotionRecognitionEnabled = src.EmotionRecognitionEnabled
	}
	if src.NumFastTurns != nil {
		dst.NumFastTurns = src.NumFastTurns
	}
	if len(src.ToolIDs) > 0 {
		dst.ToolIDs = append([]string(nil), src.ToolIDs...)
	}
	if src.VoiceProvider != "" {
		dst.VoiceProvider = src.VoiceProvider
	}
	if src.VoiceID != "" {
		dst.VoiceID = src.VoiceID
	}
	if src.VoiceModel != "" {
		dst.VoiceModel = src.VoiceModel
	}
	if src.TranscriberProvider != "" {
		dst.TranscriberProvider = src.TranscriberProvider
	}
	if src.TranscriberModel != "" {
		dst.TranscriberModel = src.TranscriberModel
	}
	if src.TranscriberLanguage != "" {
		dst.TranscriberLanguage = src.TranscriberLanguage
	}
	if src.MaxDurationSeconds != nil {
		dst.MaxDurationSeconds = src.MaxDurationSeconds
	}
	if src.ServerURL != "" {
		dst.ServerURL = src.ServerURL
	}
	if len(src.ServerHeaders) > 0 {
		dst.ServerHeaders = mergeStringAnyMaps(dst.ServerHeaders, src.ServerHeaders)
	}
	if len(src.Metadata) > 0 {
		dst.Metadata = mergeStringAnyMaps(dst.Metadata, src.Metadata)
	}
	if len(src.CallMetadata) > 0 {
		dst.CallMetadata = mergeStringAnyMaps(dst.CallMetadata, src.CallMetadata)
	}
	if src.Assistant != nil {
		dst.Assistant = src.Assistant
	}
	if len(src.ExtraAssistantBodyProperties) > 0 {
		dst.ExtraAssistantBodyProperties = mergeStringAnyMaps(dst.ExtraAssistantBodyProperties, src.ExtraAssistantBodyProperties)
	}
	if len(src.ExtraCallBodyProperties) > 0 {
		dst.ExtraCallBodyProperties = mergeStringAnyMaps(dst.ExtraCallBodyProperties, src.ExtraCallBodyProperties)
	}
}

func mergeVapiAssistantRequest(cfg VapiConfig, req VapiAssistantRequest) VapiAssistantRequest {
	merged := VapiAssistantRequest{
		Name:                      firstNonEmpty(req.Name, cfg.AssistantName),
		FirstMessage:              firstNonEmpty(req.FirstMessage, cfg.FirstMessage),
		SystemPrompt:              firstNonEmpty(req.SystemPrompt, cfg.SystemPrompt),
		ModelProvider:             firstNonEmpty(req.ModelProvider, cfg.ModelProvider),
		Model:                     firstNonEmpty(req.Model, cfg.Model),
		Temperature:               firstNonNilFloat(req.Temperature, cfg.Temperature),
		MaxTokens:                 firstNonNilFloat(req.MaxTokens, cfg.MaxTokens),
		EmotionRecognitionEnabled: firstNonNilBool(req.EmotionRecognitionEnabled, cfg.EmotionRecognitionEnabled),
		NumFastTurns:              firstNonNilFloat(req.NumFastTurns, cfg.NumFastTurns),
		ToolIDs:                   firstNonEmptyStrings(req.ToolIDs, cfg.ToolIDs),
		VoiceProvider:             firstNonEmpty(req.VoiceProvider, cfg.VoiceProvider),
		VoiceID:                   firstNonEmpty(req.VoiceID, cfg.VoiceID),
		VoiceModel:                firstNonEmpty(req.VoiceModel, cfg.VoiceModel),
		TranscriberProvider:       firstNonEmpty(req.TranscriberProvider, cfg.TranscriberProvider),
		TranscriberModel:          firstNonEmpty(req.TranscriberModel, cfg.TranscriberModel),
		TranscriberLanguage:       firstNonEmpty(req.TranscriberLanguage, cfg.TranscriberLanguage),
		MaxDurationSeconds:        firstNonNilFloat(req.MaxDurationSeconds, cfg.MaxDurationSeconds),
		ServerURL:                 firstNonEmpty(req.ServerURL, cfg.ServerURL),
		ServerHeaders:             mergeStringAnyMaps(cfg.ServerHeaders, req.ServerHeaders),
		Metadata:                  mergeStringAnyMaps(cfg.Metadata, req.Metadata),
		Messages:                  req.Messages,
		AdditionalModelMessages:   req.AdditionalModelMessages,
	}
	return merged
}

func unmarshalVapiPayload(payload map[string]any, target any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func vapiBodyOptions(bodyProperties map[string]any) []option.RequestOption {
	if len(bodyProperties) == 0 {
		return nil
	}
	return []option.RequestOption{option.WithBodyProperties(bodyProperties)}
}

func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func firstNonNilFloat(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonNilBool(values ...*bool) *bool {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmptyStrings(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return append([]string(nil), value...)
		}
	}
	return nil
}

func mergeStringAnyMaps(values ...map[string]any) map[string]any {
	var merged map[string]any
	for _, value := range values {
		if len(value) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[string]any, len(value))
		}
		for key, item := range value {
			merged[key] = item
		}
	}
	return merged
}
