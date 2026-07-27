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
		// Browsers cannot set an Authorization header on a WebSocket handshake.
		// Preferred carrier is the Sec-WebSocket-Protocol header ("bearer,
		// <token>"), which keeps the credential out of the URL entirely.
		// ?token= remains as a deprecated fallback for older clients: query
		// strings are commonly captured by upstream proxy/CDN access logs
		// (our own logger records only URL.Path). Remove the fallback once
		// every client has moved to the subprotocol form.
		var raw string
		if isWSUpgrade(c.Request) {
			raw = wsTokenFromSubprotocol(c.Request)
			if raw == "" {
				raw = c.Query("token")
			}
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
