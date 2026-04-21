package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// requireRole is a minimal RBAC guard for the scaffold.
// Next steps will replace this with DB-backed permissions and action/resource policies.
func requireRole(roleID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesAny, ok := c.Get("roleIds")
		if !ok {
			c.JSON(http.StatusForbidden, errorResponse{
				Code:    "authorization_error",
				Message: "Missing role context.",
				TraceID: c.GetString("traceId"),
			})
			c.Abort()
			return
		}

		roles, _ := rolesAny.([]string)
		for _, r := range roles {
			if r == roleID {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, errorResponse{
			Code:    "authorization_error",
			Message: "Insufficient permissions.",
			Details: gin.H{"requiredRoleId": roleID},
			TraceID: c.GetString("traceId"),
		})
		c.Abort()
	}
}

