package middleware

import (
	"net/http"
	"strings"

	"ai-ats-platform/backend/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextUserIDKey    = "user_id"
	ContextCompanyIDKey = "company_id"
	ContextEmailKey     = "email"
	ContextRoleKey      = "role"
)

type AuthMiddleware struct {
	tokenMgr *auth.TokenManager
}

func NewAuthMiddleware(tokenMgr *auth.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{tokenMgr: tokenMgr}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			respondUnauthorized(c, "missing_token", "Authorization header is required")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			respondUnauthorized(c, "invalid_token", "Authorization header must be Bearer token")
			return
		}

		claims, err := m.tokenMgr.Parse(parts[1])
		if err != nil {
			respondUnauthorized(c, "invalid_token", "Invalid or expired token")
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextCompanyIDKey, claims.CompanyID)
		c.Set(ContextEmailKey, claims.Email)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	value, ok := c.Get(ContextUserIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := value.(uuid.UUID)
	return id, ok
}

func GetCompanyID(c *gin.Context) (uuid.UUID, bool) {
	value, ok := c.Get(ContextCompanyIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := value.(uuid.UUID)
	return id, ok
}

func GetRole(c *gin.Context) (string, bool) {
	value, ok := c.Get(ContextRoleKey)
	if !ok {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}

// RequireRoles allows the request only when the JWT role is one of the allowed roles.
func (m *AuthMiddleware) RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}
	return func(c *gin.Context) {
		role, ok := GetRole(c)
		if !ok {
			respondUnauthorized(c, "invalid_token", "Invalid token context")
			return
		}
		if _, ok := allowed[strings.ToLower(role)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "forbidden",
					"message": "You do not have permission to perform this action",
				},
			})
			return
		}
		c.Next()
	}
}

func respondUnauthorized(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
