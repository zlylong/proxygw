package main

import (
	"fmt"
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func registerSyslogsRoutes(r *gin.RouterGroup) {
	r.GET("/logs/:service", func(c *gin.Context) {
		service := c.Param("service")
		if service != "proxygw" && service != "xray" && service != "mosdns" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service name"})
			return
		}

		// Get the last 200 lines
		cmd := exec.Command("journalctl", "-u", service, "-n", "200", "--no-pager")
		out, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch logs: %v\nOutput: %s", err, string(out))})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"logs":    string(out),
		})
	})
}
