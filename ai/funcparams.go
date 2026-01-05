package ai

import (
	"context"

	internalopenai "github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go/v3"
)

// FuncParams is a helper type for building OpenAI function parameters.
// It provides a fluent API for defining tool parameter schemas and
// extracting typed values from tool call arguments.
//
// Schema Building Methods:
//   - SetString, SetStringEnum - Add string parameters
//   - SetInt, SetIntRange - Add integer parameters
//   - SetNumber, SetNumberRange - Add number (float) parameters
//   - SetBool - Add boolean parameters
//   - SetArray, SetArrayWithItems - Add array parameters
//   - SetObject - Add nested object parameters
//   - SetRequired, AddRequired - Set required fields
//
// Value Extraction Methods (for use in handlers):
//   - GetString, GetStringDefault
//   - GetInt, GetIntDefault
//   - GetFloat, GetFloatDefault
//   - GetBool, GetBoolDefault
//   - GetStringArray
//   - GetMap
type FuncParams = internalopenai.FuncParams

// GoFunctionTool represents a Go function that can be called by the AI model.
// The Handler receives a FuncParams which provides helper methods to extract
// typed values from the tool call arguments.
//
// Example:
//
//	tool := ai.NewGoFunctionTool(
//		"get_weather",
//		"Get the current weather",
//		ai.NewFuncParams().
//			SetString("location", "The city name").
//			SetRequired("location"),
//		func(ctx context.Context, args ai.FuncParams) (string, error) {
//			location := args.GetStringDefault("location", "Unknown")
//			return fmt.Sprintf(`{"weather": "sunny", "location": "%s"}`, location), nil
//		},
//	)
type GoFunctionTool = internalopenai.GoFunctionTool

// NewFuncParams creates a new FuncParams with default object type.
//
// Example:
//
//	params := ai.NewFuncParams().
//		SetString("name", "User's full name").
//		SetInt("age", "User's age in years").
//		SetStringEnum("status", "Account status", []string{"active", "inactive", "pending"}).
//		SetRequired("name", "age")
func NewFuncParams(history ...*models.AIChatHistory) FuncParams {
	if len(history) != 0 {
		return internalopenai.NewFuncParams(history[0])
	}
	return internalopenai.NewFuncParams(nil)
}

// --- Helper functions to create GoFunctionTool easily ---

// NewGoFunctionTool creates a new GoFunctionTool with the given parameters.
// The handler receives FuncParams which provides helper methods like
// GetString, GetInt, GetFloat, etc. for extracting typed values.
//
// Example:
//
//	tool := ai.NewGoFunctionTool(
//		"calculate",
//		"Perform arithmetic operations",
//		ai.NewFuncParams().
//			SetNumber("a", "First operand").
//			SetNumber("b", "Second operand").
//			SetStringEnum("op", "Operation", []string{"add", "subtract", "multiply", "divide"}).
//			SetRequired("a", "b", "op"),
//		func(ctx context.Context, args ai.FuncParams) (string, error) {
//			a := args.GetFloatDefault("a", 0)
//			b := args.GetFloatDefault("b", 0)
//			op := args.GetStringDefault("op", "add")
//
//			var result float64
//			switch op {
//			case "add":
//				result = a + b
//			case "subtract":
//				result = a - b
//			case "multiply":
//				result = a * b
//			case "divide":
//				result = a / b
//			}
//			return fmt.Sprintf(`{"result": %f}`, result), nil
//		},
//	)
func NewGoFunctionTool(
	name string,
	description string,
	params FuncParams,
	handler func(context.Context, FuncParams) (string, error),
) GoFunctionTool {
	return GoFunctionTool{
		Name:        name,
		Description: description,
		Parameters:  openai.FunctionParameters(params),
		Strict:      false,
		Handler:     handler,
	}
}

// NewStrictGoFunctionTool creates a new GoFunctionTool with strict mode enabled.
// Strict mode ensures the model follows the parameter schema exactly.
// The handler receives FuncParams which provides helper methods for extracting typed values.
func NewStrictGoFunctionTool(
	name string,
	description string,
	params FuncParams,
	handler func(context.Context, FuncParams) (string, error),
) GoFunctionTool {
	return GoFunctionTool{
		Name:        name,
		Description: description,
		Parameters:  openai.FunctionParameters(params),
		Strict:      true,
		Handler:     handler,
	}
}
