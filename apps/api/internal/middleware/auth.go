package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/speedplus/api/internal/service"
)

const (
	CtxUserID   = "user_id"
	CtxUserRole = "user_role"
)

func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Session expired. Please log in again."},
			})
			return
		}

		claims, err := authSvc.ValidateAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Session expired. Please log in again."},
			})
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUserRole, claims.Role)
		c.Next()
	}
}

// RequireRole enforces vertical RBAC.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role := c.GetString(CtxUserRole)
		if _, ok := allowed[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "You do not have permission to do this."},
			})
			return
		}
		c.Next()
	}
}

// OwnerOrAdmin enforces row-level ownership: the :id param must match the
// authenticated user's ID, unless the caller is an admin.
func OwnerOrAdmin(paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString(CtxUserRole)
		if role == "admin" {
			c.Next()
			return
		}
		if c.Param(paramKey) != c.GetString(CtxUserID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "You do not have permission to do this."},
			})
			return
		}
		c.Next()
	}
}
