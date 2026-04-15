package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func registerConfigRoutes(api *gin.RouterGroup) {
	api.GET("/config/xray", func(c *gin.Context) {
		data, err := os.ReadFile("../core/xray/config.json")
		if err != nil {
			c.String(http.StatusInternalServerError, "read config failed")
			return
		}
		c.String(http.StatusOK, string(data))
	})

	api.GET("/config/mosdns", func(c *gin.Context) {
		data, err := os.ReadFile("../core/mosdns/config.yaml")
		if err != nil {
			c.String(http.StatusInternalServerError, "read config failed")
			return
		}
		c.String(http.StatusOK, string(data))
	})
}
