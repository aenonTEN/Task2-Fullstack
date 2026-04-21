package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func whoAmI(c *gin.Context) {
	userID := c.GetString("userId")

	scopeAny, _ := c.Get("scope")
	scope, _ := scopeAny.(dataScope)

	roleAny, _ := c.Get("roleIds")
	roles, _ := roleAny.([]string)

	c.JSON(http.StatusOK, gin.H{
		"userId":  userID,
		"roleIds": roles,
		"scope":   scope,
	})
}

