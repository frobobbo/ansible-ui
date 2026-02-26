package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ClaimsKey = "claims"

func Middleware(jwtSvc *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtSvc.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func RequireAdmin(c *gin.Context) {
	claims, ok := c.MustGet(ClaimsKey).(*Claims)
	if !ok || claims.Role != "admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}
	c.Next()
}

// RequireEditor allows admin and editor roles; viewers are rejected.
func RequireEditor(c *gin.Context) {
	claims, ok := c.MustGet(ClaimsKey).(*Claims)
	if !ok || (claims.Role != "admin" && claims.Role != "editor") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "editor role required"})
		return
	}
	c.Next()
}

func GetClaims(c *gin.Context) *Claims {
	v, _ := c.Get(ClaimsKey)
	claims, _ := v.(*Claims)
	return claims
}
