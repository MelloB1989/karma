package openai

import (
	"context"
	"encoding/json"
	"fmt"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go/v3"
)

const defaultMaxToolPasses = 5

type FuncParams openai.FunctionParameters

func NewFuncParams() FuncParams {
	return FuncParams{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (fp FuncParams) ToOpenAI() openai.FunctionParameters {
	return openai.FunctionParameters(fp)
}

func (fp FuncParams) getProperties() map[string]any {
	if props, ok := fp["properties"].(map[string]any); ok {
		return props
	}
	props := map[string]any{}
	fp["properties"] = props
	return props
}

func (fp FuncParams) SetString(key, description string) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "string",
		"description": description,
	}
	return fp
}

func (fp FuncParams) SetStringEnum(key, description string, enumValues []string) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "string",
		"description": description,
		"enum":        enumValues,
	}
	return fp
}

func (fp FuncParams) SetInt(key, description string) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "integer",
		"description": description,
	}
	return fp
}

func (fp FuncParams) SetIntRange(key, description string, min, max int) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "integer",
		"description": description,
		"minimum":     min,
		"maximum":     max,
	}
	return fp
}

func (fp FuncParams) SetNumber(key, description string) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "number",
		"description": description,
	}
	return fp
}

func (fp FuncParams) SetNumberRange(key, description string, min, max float64) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "number",
		"description": description,
		"minimum":     min,
		"maximum":     max,
	}
	return fp
}

func (fp FuncParams) SetBool(key, description string) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "boolean",
		"description": description,
	}
	return fp
}

func (fp FuncParams) SetArray(key, description, itemType string) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type": itemType,
		},
	}
	return fp
}

func (fp FuncParams) SetArrayWithItems(key, description string, itemSchema map[string]any) FuncParams {
	props := fp.getProperties()
	props[key] = map[string]any{
		"type":        "array",
		"description": description,
		"items":       itemSchema,
	}
	return fp
}

func (fp FuncParams) SetObject(key, description string, nestedParams FuncParams) FuncParams {
	props := fp.getProperties()
	nested := map[string]any{
		"type":        "object",
		"description": description,
	}
	if nestedProps, ok := nestedParams["properties"]; ok {
		nested["properties"] = nestedProps
	}
	if req, ok := nestedParams["required"]; ok {
		nested["required"] = req
	}
	props[key] = nested
	return fp
}

func (fp FuncParams) SetCustom(key string, schema map[string]any) FuncParams {
	props := fp.getProperties()
	props[key] = schema
	return fp
}

func (fp FuncParams) SetRequired(keys ...string) FuncParams {
	fp["required"] = keys
	return fp
}

func (fp FuncParams) AddRequired(keys ...string) FuncParams {
	existing, ok := fp["required"].([]string)
	if !ok {
		existing = []string{}
	}
	fp["required"] = append(existing, keys...)
	return fp
}

func (fp FuncParams) SetAdditionalProperties(allowed bool) FuncParams {
	fp["additionalProperties"] = allowed
	return fp
}

// --- Getter methods on FuncParams for extracting values from tool call arguments ---

func (fp FuncParams) GetString(key string) (string, bool) {
	if v, ok := fp[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

func (fp FuncParams) GetStringDefault(key, defaultValue string) string {
	if s, ok := fp.GetString(key); ok {
		return s
	}
	return defaultValue
}

func (fp FuncParams) GetInt(key string) (int, bool) {
	if v, ok := fp[key]; ok {
		switch n := v.(type) {
		case int:
			return n, true
		case int64:
			return int(n), true
		case float64:
			return int(n), true
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return int(i), true
			}
		}
	}
	return 0, false
}

func (fp FuncParams) GetIntDefault(key string, defaultValue int) int {
	if i, ok := fp.GetInt(key); ok {
		return i
	}
	return defaultValue
}

func (fp FuncParams) GetFloat(key string) (float64, bool) {
	if v, ok := fp[key]; ok {
		switch n := v.(type) {
		case float64:
			return n, true
		case float32:
			return float64(n), true
		case int:
			return float64(n), true
		case int64:
			return float64(n), true
		case json.Number:
			if f, err := n.Float64(); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func (fp FuncParams) GetFloatDefault(key string, defaultValue float64) float64 {
	if f, ok := fp.GetFloat(key); ok {
		return f
	}
	return defaultValue
}

func (fp FuncParams) GetBool(key string) (bool, bool) {
	if v, ok := fp[key]; ok {
		if b, ok := v.(bool); ok {
			return b, true
		}
	}
	return false, false
}

func (fp FuncParams) GetBoolDefault(key string, defaultValue bool) bool {
	if b, ok := fp.GetBool(key); ok {
		return b
	}
	return defaultValue
}

func (fp FuncParams) GetStringArray(key string) ([]string, bool) {
	if v, ok := fp[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result, true
		}
		if arr, ok := v.([]string); ok {
			return arr, true
		}
	}
	return nil, false
}

func (fp FuncParams) GetMap(key string) (FuncParams, bool) {
	if v, ok := fp[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return FuncParams(m), true
		}
	}
	return nil, false
}

type GoFunctionTool struct {
	Name        string
	Description string
	Parameters  openai.FunctionParameters
	Strict      bool
	Handler     func(context.Context, FuncParams) (string, error)
}

func (o *OpenAI) SetMCPServer(serverURL string, authToken string) {
	mcpClient := mcp.NewClient(serverURL, authToken)
	o.MCPManager = mcp.NewManager(mcpClient)
}

func (o *OpenAI) SetMultiMCPManager(multiManager *mcp.MultiManager) {
	o.MultiMCPManager = multiManager
}

func (o *OpenAI) SetMaxToolPasses(max int) {
	o.maxToolPasses = max
}

func (o *OpenAI) AddMCPTool(name, description, mcpToolName string, inputSchema any) error {
	if o.MCPManager == nil {
		return fmt.Errorf("MCP server not configured. Call SetMCPServer first")
	}
	return o.MCPManager.AddToolFromSchema(name, description, mcpToolName, inputSchema)
}

func (o *OpenAI) AddGoFunctionTool(tool GoFunctionTool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name required")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler required")
	}
	tool.Parameters = coerceFunctionParameters(tool.Parameters)
	o.FunctionTools[tool.Name] = tool
	return nil
}

func (o *OpenAI) AddGoFunctionDefinition(def models.OpenAIFunctionDefinition, handler func(context.Context, FuncParams) (string, error)) error {
	tool := GoFunctionTool{
		Name:        def.Name,
		Description: def.Description,
		Parameters:  coerceFunctionParameters(def.Parameters),
		Strict:      def.Strict,
		Handler:     handler,
	}
	return o.AddGoFunctionTool(tool)
}

func (o *OpenAI) ClearGoFunctionTools() {
	o.FunctionTools = make(map[string]GoFunctionTool)
}

func (o *OpenAI) GetMCPManager() *mcp.Manager {
	return o.MCPManager
}
