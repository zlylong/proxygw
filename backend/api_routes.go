package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func authMiddleware(c *gin.Context) {
	if c.GetHeader("Authorization") != "Bearer "+sessionToken {
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
