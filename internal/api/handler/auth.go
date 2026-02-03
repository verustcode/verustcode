// Package handler provides HTTP handlers for the API.
package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
	"go.uber.org/zap"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	config     *config.Config
	configPath string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		config:     cfg,
		configPath: config.BootstrapConfigPath,
	}
}

// NewAuthHandlerWithConfigPath creates a new auth handler with custom config path
func NewAuthHandlerWithConfigPath(cfg *config.Config, configPath string) *AuthHandler {
	return &AuthHandler{
		config:     cfg,
		configPath: configPath,
	}
}

// RememberMeExpirationHours is the token expiration time when "remember me" is enabled (7 days)
const RememberMeExpirationHours = 168

// LoginRequest represents the login request body
type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"` // If true, token expires in 7 days
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Claims represents JWT claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body",
		})
		return
	}

	// Check if admin is configured
	if h.config.Admin == nil || !h.config.Admin.Enabled {
		logger.Error("Admin console is not enabled")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    errors.ErrCodeUnauthorized,
			"message": "Admin console is not enabled",
		})
		return
	}

	// Validate username
	if req.Username != h.config.Admin.Username {
		logger.Warn("Invalid login attempt", zap.String("username", req.Username))
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    errors.ErrCodeUnauthorized,
			"message": "Invalid username or password",
		})
		return
	}

	// Validate password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(h.config.Admin.PasswordHash), []byte(req.Password)); err != nil {
		logger.Warn("Invalid login attempt", zap.String("username", req.Username))
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    errors.ErrCodeUnauthorized,
			"message": "Invalid username or password",
		})
		return
	}

	// Generate JWT token
	// Determine token expiration: 7 days if "remember me" is enabled, otherwise use configured expiry_hours
	var expirationHours int
	if req.RememberMe {
		expirationHours = RememberMeExpirationHours
		logger.Debug("Using remember me expiration", zap.Int("hours", expirationHours))
	} else {
		expirationHours = h.config.Admin.TokenExpiration
		if expirationHours <= 0 {
			expirationHours = 24 // default 24 hours
		}
	}

	expiresAt := time.Now().Add(time.Duration(expirationHours) * time.Hour)

	claims := &Claims{
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "verustcode",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// JWT secret is validated at startup - no fallback needed
	jwtSecret := h.config.Admin.JWTSecret

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		logger.Error("Failed to generate JWT token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to generate token",
		})
		return
	}

	logger.Info("User logged in", zap.String("username", req.Username))

	c.JSON(http.StatusOK, LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	// Get username from context (set by auth middleware)
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    errors.ErrCodeUnauthorized,
			"message": "Not authenticated",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": username,
	})
}

// ValidateToken validates a JWT token and returns the username
// Implements middleware.TokenValidator interface
func (h *AuthHandler) ValidateToken(tokenString string) (string, error) {
	// Check if admin config exists
	if h.config.Admin == nil {
		return "", fmt.Errorf("admin configuration not available")
	}

	// Explicit check for empty JWT secret (P1-1 security improvement)
	jwtSecret := h.config.Admin.JWTSecret
	if jwtSecret == "" {
		return "", fmt.Errorf("JWT secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.Username, nil
	}

	return "", jwt.ErrSignatureInvalid
}

// SetupStatusResponse represents the setup status response
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

// SetupPasswordRequest represents the setup password request body
type SetupPasswordRequest struct {
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// GetSetupStatus handles GET /api/v1/auth/setup/status
// Returns 404 if password is already set (to hide API existence)
func (h *AuthHandler) GetSetupStatus(c *gin.Context) {
	// If password is already set, return 404 to hide API existence
	if h.config.Admin != nil && h.config.Admin.PasswordHash != "" {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Not found",
		})
		return
	}

	// Password needs to be set
	c.JSON(http.StatusOK, SetupStatusResponse{
		NeedsSetup: true,
	})
}

// SetupPassword handles POST /api/v1/auth/setup
// Allows setting password for the first time without authentication
// Returns 404 if password is already set (to hide API existence)
func (h *AuthHandler) SetupPassword(c *gin.Context) {
	// If password is already set, return 404 to hide API existence
	if h.config.Admin != nil && h.config.Admin.PasswordHash != "" {
		logger.Warn("Attempt to access setup API when password already set",
			zap.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Not found",
		})
		return
	}

	var req SetupPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body",
		})
		return
	}

	// Check passwords match
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Passwords do not match",
		})
		return
	}

	// Validate password complexity
	passwordReq := config.DefaultPasswordRequirements()
	if err := config.ValidatePassword(req.Password, passwordReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": fmt.Sprintf("Password validation failed: %v", err),
		})
		return
	}

	// Generate bcrypt hash
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Failed to generate password hash", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to generate password hash",
		})
		return
	}

	// Update config file with new password hash
	if err := updatePasswordHashInConfig(h.configPath, string(hash)); err != nil {
		logger.Error("Failed to update config file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to save password",
		})
		return
	}

	// Reload bootstrap configuration
	newBootstrap, err := config.LoadBootstrap(h.configPath)
	if err != nil {
		logger.Error("Failed to reload bootstrap config after password setup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to reload configuration",
		})
		return
	}

	// Update the admin config reference
	h.config.Admin = newBootstrap.Admin

	logger.Info("Admin password set successfully via web UI")

	c.JSON(http.StatusOK, gin.H{
		"message": "Password set successfully",
	})
}
