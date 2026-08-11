package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/service"
)

const (
	CtxUserID     = "user_id"
	CtxUserRole   = "user_role"
	CtxIsVerified = "is_verified"
)

func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var raw string
		if isWSUpgrade(c.Request) {
			raw = wsTokenFromSubprotocol(c.Request)
		}
		if raw == "" {
			header := c.GetHeader("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{"code": "UNAUTHORIZED", "message": "Session expired. Please log in again."},
				})
				return
			}
			raw = strings.TrimPrefix(header, "Bearer ")
		}

		claims, err := authSvc.ValidateAccessToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Session expired. Please log in again."},
			})
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUserRole, claims.Role)
		c.Set(CtxIsVerified, claims.IsVerified)
		c.Next()
	}
}

// isWSUpgrade returns true when the request is a WebSocket upgrade.
func isWSUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// WSBearerSubprotocol is the sentinel value a client sends alongside its token
// in Sec-WebSocket-Protocol: new WebSocket(url, ["bearer", token]).
const WSBearerSubprotocol = "bearer"

// wsTokenFromSubprotocol extracts the access token from the
// Sec-WebSocket-Protocol header. The browser API sends the requested
// subprotocols as a comma-separated list, so ["bearer", token] arrives as
// "bearer, <token>". Returns "" when the header is absent or malformed, so
// the caller can fall back to the deprecated query parameter.
//
// The server must echo the accepted subprotocol back on the handshake or the
// browser closes the connection — see ws.upgrader.
func wsTokenFromSubprotocol(r *http.Request) string {
	header := r.Header.Get("Sec-WebSocket-Protocol")
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	for i, p := range parts {
		if !strings.EqualFold(strings.TrimSpace(p), WSBearerSubprotocol) {
			continue
		}
		if i+1 < len(parts) {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

// RequireActiveUser checks users.is_active on every request.
// This is a DB hit but necessary: access tokens are stateless JWTs (15 min TTL)
// so revoking refresh tokens alone does not provide immediate lockout for AML/fraud cases.
func RequireActiveUser(users userFinder) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.GetString(CtxUserID))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Session expired. Please log in again."},
			})
			return
		}
		u, err := users.FindByID(c.Request.Context(), userID)
		if err != nil || !u.IsActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "ACCOUNT_INACTIVE", "message": "Your account has been deactivated. Please contact support."},
			})
			return
		}
		c.Next()
	}
}

type userFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}
// IsVerified is baked into the JWT at issue time — no DB hit required.
func RequireVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool(CtxIsVerified) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "UNVERIFIED",
					"message": "Phone verification required before placing orders.",
				},
			})
			return
		}
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
