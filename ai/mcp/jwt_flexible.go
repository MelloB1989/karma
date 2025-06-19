package mcp

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/golang-jwt/jwt"
)

var (
	JWTSecret   = config.DefaultConfig().JWTSecret
	TokenExpiry = 24 * time.Hour
)

// JWTClaims interface that any custom claims struct must implement
type JWTClaims interface {
	jwt.Claims
	GetUserID() string
	GetRole() string
	GetEmail() string
	IsExpired() bool
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Validator  *DefaultJWTValidator
	ClaimsType reflect.Type
	ContextKey string
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Name  string `json:"name"`
}

type DefaultClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

func (c *DefaultClaims) GetUserID() string {
	return c.UserID
}

func (c *DefaultClaims) GetRole() string {
	return c.Role
}

func (c *DefaultClaims) GetEmail() string {
	return c.Email
}

func (c *DefaultClaims) IsExpired() bool {
	return c.ExpiresAt < time.Now().Unix()
}

type DefaultJWTValidator struct {
	SigningKey  []byte
	TokenExpiry time.Duration
	ClaimsType  reflect.Type
}

func NewDefaultJWTValidator() *DefaultJWTValidator {
	return &DefaultJWTValidator{
		SigningKey:  []byte(JWTSecret),
		TokenExpiry: TokenExpiry,
		ClaimsType:  reflect.TypeOf((*DefaultClaims)(nil)).Elem(),
	}
}

func NewCustomJWTValidator(claimsType reflect.Type) *DefaultJWTValidator {
	return &DefaultJWTValidator{
		SigningKey:  []byte(JWTSecret),
		TokenExpiry: TokenExpiry,
		ClaimsType:  claimsType,
	}
}

func (v *DefaultJWTValidator) GetSigningKey() any {
	return v.SigningKey
}

func (v *DefaultJWTValidator) ValidateToken(tokenString string) (JWTClaims, error) {
	// Create a new instance of the claims type
	claims := reflect.New(v.ClaimsType).Interface().(JWTClaims)

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.SigningKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	if claims.IsExpired() {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

func NewDefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		Validator:  NewDefaultJWTValidator(),
		ClaimsType: reflect.TypeOf((*DefaultClaims)(nil)).Elem(),
		ContextKey: "jwt_claims",
	}
}

func NewCustomJWTConfig(claimsType reflect.Type) *JWTConfig {
	return &JWTConfig{
		Validator:  NewCustomJWTValidator(claimsType),
		ClaimsType: claimsType,
		ContextKey: "jwt_claims",
	}
}

func GetClaimsFromContextFlexible(ctx context.Context) JWTClaims {
	if claims, ok := ctx.Value("jwt_claims").(JWTClaims); ok {
		return claims
	}
	return nil
}

func GetRawClaims(ctx context.Context) any {
	return ctx.Value("jwt_claims")
}

func GetUserFromContextFlexible(ctx context.Context) any {
	return ctx.Value("user")
}

func GetDefaultClaims(ctx context.Context) *DefaultClaims {
	if claims := GetClaimsFromContextFlexible(ctx); claims != nil {
		if defaultClaims, ok := claims.(*DefaultClaims); ok {
			return defaultClaims
		}
	}
	return nil
}

func GetDefaultUser(ctx context.Context) *User {
	if user := GetUserFromContextFlexible(ctx); user != nil {
		if defaultUser, ok := user.(*User); ok {
			return defaultUser
		}
	}
	return nil
}
