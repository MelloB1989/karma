package tests

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MelloB1989/karma/ai/mcp"
	"github.com/golang-jwt/jwt"
	mc "github.com/mark3labs/mcp-go/mcp"
)

type MyClaims struct {
	UserID      string `json:"uid"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Gender      string `json:"gender"`
	DateOfBirth string `json:"date_of_birth"`
	Name        string `json:"name"`
	Pfp         string `json:"pfp"`
	Role        string `json:"role"`
	jwt.StandardClaims
}

func (c *MyClaims) GetUserID() string {
	return c.UserID
}

func (c *MyClaims) GetRole() string {
	return c.Role
}

func (c *MyClaims) GetEmail() string {
	return c.Email
}

func (c *MyClaims) IsExpired() bool {
	return c.ExpiresAt < time.Now().Unix()
}

func TestMCPServer(auth bool) {
	myMcp := mcp.NewMCPServer("Complex Server", "1.0.0",
		mcp.WithDebug(true),
		mcp.WithRateLimit(mcp.RateLimit{Limit: 10, Window: time.Minute * 1}),
		mcp.WithAuthentication(auth),
		mcp.WithLogging(true), // Explicitly enable logging
		mcp.WithPort(8086),
		mcp.WithEndpoint("mcp"),
		mcp.WithTools(exampleCalcTool()),
		// mcp.WithCustomJWT(MyClaims{}),
	)
	myMcp.Start()
}

func exampleCalcTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"calculate",
			mc.WithDescription("Perform basic arithmetic operations"),
			mc.WithString("operation",
				mc.Required(),
				mc.Description("The operation to perform (add, subtract, multiply, divide)"),
				mc.Enum("add", "subtract", "multiply", "divide"),
			),
			mc.WithNumber("x",
				mc.Required(),
				mc.Description("First number"),
			),
			mc.WithNumber("y",
				mc.Required(),
				mc.Description("Second number"),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			// Get JWT claims for additional security context
			// claims := mcp.GetRawClaims(ctx)

			// Extract parameters
			op, err := request.RequireString("operation")
			if err != nil {
				log.Printf("Failed to get operation parameter: %v", err)
				return mc.NewToolResultError(err.Error()), nil
			}

			x, err := request.RequireFloat("x")
			if err != nil {
				log.Printf("Failed to get x parameter: %v", err)
				return mc.NewToolResultError(err.Error()), nil
			}

			y, err := request.RequireFloat("y")
			if err != nil {
				log.Printf("Failed to get y parameter: %v", err)
				return mc.NewToolResultError(err.Error()), nil
			}

			log.Printf("Calculating: %f %s %f", x, op, y)

			var result float64
			switch op {
			case "add":
				result = x + y
			case "subtract":
				result = x - y
			case "multiply":
				result = x * y
			case "divide":
				if y == 0 {
					return mc.NewToolResultError("cannot divide by zero"), nil
				}
				result = x / y
			default:
				return mc.NewToolResultError(fmt.Sprintf("unsupported operation: %s", op)), nil
			}

			responseText := fmt.Sprintf("Result: %.2f (operation: %s, x: %.2f, y: %.2f, timestamp: %s)",
				result, op, x, y, time.Now().Format(time.RFC3339))

			log.Printf("Calculator result: %s", responseText)
			return mc.NewToolResultText(responseText), nil
		},
	}
}
