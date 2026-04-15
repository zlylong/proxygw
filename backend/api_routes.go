package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func authMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if !validateSession(token) {
		c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	c.Next()
}

func registerAPIRoutes(r *gin.Engine) {
	api := r.Group("/api")

	authed := api.Group("")
	authed.Use(authMiddleware)
	registerAuthRoutes(api, authed)

	registerConfigRoutes(authed)
	registerSystemRoutes(authed)
	registerNodeRoutes(authed)
	registerRuleRoutes(authed)
	registerDNSRoutes(authed)
	registerUpdateRoutes(authed)

	authed.POST("/apply", func(c *gin.Context) {
		applyMosdnsConfig()
		applyXrayConfig()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}
