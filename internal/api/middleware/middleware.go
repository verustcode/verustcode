// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
)

// LoggerConfig holds the configuration for the Logger middleware
type LoggerConfig struct {
	// AccessLog determines if HTTP request logs should be printed at info level
	// When true, successful requests (status < 400) are logged; when false, they are not
	AccessLog bool
}

// Logger returns a middleware that logs HTTP requests
// If cfg is nil, defaults to not logging access requests (accessLog = false)
func Logger(cfg *LoggerConfig) gin.HandlerFunc {
	// Default to not logging access logs if no config provided
	accessLog := false
	if cfg != nil {
		accessLog = cfg.AccessLog
	}

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := c.Writer.Status()

		// Build log fields
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case status >= 500:
			logger.Error("Server error", fields...)
		case status >= 400:
			logger.Warn("Client error", fields...)
		default:
			// Only log successful requests if accessLog is enabled
			if accessLog {
				logger.Info("Request", fields...)
			}
		}
	}
}

// Recovery returns a middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stack := debug.Stack()
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.ByteString("stack", stack),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// Return internal server error
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    errors.ErrCodeInternal,
					"message": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}

// CORS returns a middleware that handles CORS headers with origin whitelist validation
func CORS(allowedOrigins []string) gin.HandlerFunc {
	// Build a set for O(1) lookup
	originSet := make(map[string]bool)
	for _, origin := range allowedOrigins {
		originSet[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Only set CORS headers if origin is in the whitelist
		if origin != "" && originSet[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if origin != "" && originSet[origin] {
				c.AbortWithStatus(http.StatusNoContent)
			} else {
				c.AbortWithStatus(http.StatusForbidden)
			}
			return
		}

		c.Next()
	}
}

// RequestID returns a middleware that adds a request ID to the context
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or generate request ID
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID generates a unique request ID for HTTP request tracking
func generateRequestID() string {
	return idgen.NewRequestID()
}

// ErrorHandler returns a middleware that handles errors uniformly
// P2-1 Security improvement: In production mode (debugMode=false), sensitive error details are hidden
func ErrorHandler(debugMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Try to convert to AppError
			if appErr, ok := errors.AsAppError(err); ok {
				response := gin.H{
					"code": appErr.Code,
				}
				// P2-1: Hide sensitive error messages in production mode
				// For internal errors, always hide message; for other errors, show message
				if appErr.HTTPStatus() >= http.StatusInternalServerError && !debugMode {
					response["message"] = "Internal server error"
				} else {
					response["message"] = appErr.Message
				}
				// Only include details in debug mode
				if debugMode && appErr.Details != "" {
					response["details"] = appErr.Details
				}
				c.JSON(appErr.HTTPStatus(), response)
				return
			}

			// Default error response
			// P2-1: Hide raw error message in production mode
			msg := "Internal server error"
			if debugMode {
				msg = err.Error()
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeInternal,
				"message": msg,
			})
		}
	}
}

// TokenValidator is an interface for validating JWT tokens
type TokenValidator interface {
	ValidateToken(token string) (username string, err error)
}

// JWTAuth returns a middleware that validates JWT tokens
func JWTAuth(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errors.ErrCodeUnauthorized,
				"message": "Authorization header required",
			})
			return
		}

		// Check Bearer prefix
		const bearerPrefix = "Bearer "
		if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errors.ErrCodeUnauthorized,
				"message": "Invalid authorization format",
			})
			return
		}

		// Extract token
		token := authHeader[len(bearerPrefix):]

		// Validate token
		username, err := validator.ValidateToken(token)
		if err != nil {
			logger.Debug("JWT validation failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errors.ErrCodeUnauthorized,
				"message": "Invalid or expired token",
			})
			return
		}

		// Set username in context
		c.Set("username", username)
		c.Next()
	}
}
