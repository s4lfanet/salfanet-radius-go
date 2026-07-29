// Package middleware provides Fiber middleware for the API server.
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/config"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// nextAuthHTTPClient is a shared HTTP client for NextAuth session validation calls.
// Uses persistent connections to avoid TCP handshake overhead on every request.
var nextAuthHTTPClient = &http.Client{
	Timeout: 3 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

type nextAuthSessionResult struct {
	UserID string
	Email  string
	Role   string
	Name   string
}

// validateNextAuthSession calls the internal NextAuth session endpoint to validate
// a browser session cookie. Returns user info on success, error on invalid/expired session.
func validateNextAuthSession(cookieHeader string) (*nextAuthSessionResult, error) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:3000/api/auth/session", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", cookieHeader)
	// Set Host to ensure NextAuth can find the correct NEXTAUTH_URL config
	req.Host = "localhost"

	resp, err := nextAuthHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nextauth unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nextauth returned %d", resp.StatusCode)
	}

	var data struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Role  string `json:"role"`
			Name  string `json:"name"`
		} `json:"user"`
		Expires string `json:"expires"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("nextauth decode error: %w", err)
	}

	if data.User.ID == "" {
		return nil, fmt.Errorf("no active nextauth session")
	}

	return &nextAuthSessionResult{
		UserID: data.User.ID,
		Email:  data.User.Email,
		Role:   data.User.Role,
		Name:   data.User.Name,
	}, nil
}

// Claims is the JWT payload structure.
type Claims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens in the Authorization header.
func AuthMiddleware(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
	}
	tokenStr := parts[1]

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(config.C.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
	}

	c.Locals("userID", claims.UserID)
	c.Locals("email", claims.Email)
	c.Locals("role", claims.Role)
	return c.Next()
}

// RequireAdmin rejects requests from non-admin users.
// Accepts both ADMIN and SUPER_ADMIN roles.
func RequireAdmin(c fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if role != "ADMIN" && role != "SUPER_ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	return c.Next()
}

// RequireRole returns a middleware that checks the user has one of the allowed roles.
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		if _, ok := allowed[role]; !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
		}
		return c.Next()
	}
}

// loginRateLimit is a simple in-memory rate limiter for auth endpoints.
// Allows maxAttempts per IP per window.
var (
	rateLimitMu     sync.Mutex
	rateLimitMap    = make(map[string]*rateLimitEntry)
	rateLimitWindow = 15 * time.Minute
	rateLimitMax    = 10
)

type rateLimitEntry struct {
	count    int
	expireAt time.Time
}

// LoginRateLimit middleware limits login attempts per IP.
func LoginRateLimit(c fiber.Ctx) error {
	ip := c.IP()
	if ip == "" {
		ip = "unknown"
	}
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	now := time.Now()
	entry, exists := rateLimitMap[ip]
	if !exists || now.After(entry.expireAt) {
		rateLimitMap[ip] = &rateLimitEntry{count: 1, expireAt: now.Add(rateLimitWindow)}
		return c.Next()
	}
	entry.count++
	if entry.count > rateLimitMax {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many login attempts, try again later"})
	}
	return c.Next()
}

// AdminPathGuard checks if the request path starts with /admin/, /cron/, or
// /backup/ and enforces RequireAdmin on those routes. This catches both the
// admin group and standalone api.Get/Post/etc routes registered with those
// prefixes.
func AdminPathGuard(c fiber.Ctx) error {
	path := c.Path()
	if strings.HasPrefix(path, "/admin/") ||
		strings.HasPrefix(path, "/cron/") ||
		strings.HasPrefix(path, "/backup/") {
		return RequireAdmin(c)
	}
	return c.Next()
}

// CombinedAuthMiddleware validates admin panel requests using either:
//  1. JWT Bearer token (used by mobile / API clients sending Authorization header)
//  2. NextAuth session cookie (used by admin panel browser requests)
//
// This allows nginx to route ALL /api/ traffic to the Go backend without the
// admin panel needing to change how it authenticates.
func CombinedAuthMiddleware(c fiber.Ctx) error {
	// Attempt 1: JWT Bearer token
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fiber.ErrUnauthorized
				}
				return []byte(config.C.JWTSecret), nil
			})
			if err == nil && token.Valid {
				c.Locals("userID", claims.UserID)
				c.Locals("email", claims.Email)
				c.Locals("role", claims.Role)
				return c.Next()
			}
		}
	}

	// Attempt 2: NextAuth session cookie (admin panel browser requests)
	cookieHeader := c.Get("Cookie")
	if strings.Contains(cookieHeader, "next-auth") {
		session, err := validateNextAuthSession(cookieHeader)
		if err == nil {
			c.Locals("userID", session.UserID)
			c.Locals("email", session.Email)
			c.Locals("role", session.Role)
			return c.Next()
		}
	}

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
}

// NewCustomerAuthMiddleware returns a Fiber middleware that validates customer session
// tokens against the customer_sessions table in the database.
func NewCustomerAuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
		}
		tokenStr := parts[1]

		var session models.CustomerSession
		if err := db.Where("token = ? AND verified = 1", tokenStr).First(&session).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired session"})
		}
		if session.ExpiresAt != nil && session.ExpiresAt.Before(time.Now()) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session expired"})
		}

		c.Locals("customerToken", tokenStr)
		c.Locals("customerID", session.UserID)
		return c.Next()
	}
}
