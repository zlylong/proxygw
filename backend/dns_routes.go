package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var dnsLogWSConnections int32

func registerDNSRoutes(api *gin.RouterGroup) {
	api.GET("/dns", func(c *gin.Context) {
		var local, remote, lazy, mode string
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazy); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_mode'").Scan(&mode); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
		if strings.TrimSpace(mode) == "" {
			mode = "smart"
			db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('dns_mode', 'smart')")
		}
		c.JSON(http.StatusOK, gin.H{"local": local, "remote": remote, "lazy": lazy == "true", "mode": mode})
	})

	api.POST("/dns", func(c *gin.Context) {
		var req struct {
			Local, Remote, Mode string
			Lazy                bool
		}
		if c.BindJSON(&req) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		local, ok := normalizeUpstreamCSV(req.Local)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid local upstream"})
			return
		}
		remote, ok := normalizeUpstreamCSV(req.Remote)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid remote upstream"})
			return
		}

		mode := strings.TrimSpace(req.Mode)
		if mode == "" {
			mode = "smart"
		}
		if _, err := db.Exec("UPDATE settings SET value=? WHERE key='dns_local'", local); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err := db.Exec("UPDATE settings SET value=? WHERE key='dns_remote'", remote); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err := db.Exec("UPDATE settings SET value=? WHERE key='dns_lazy'", fmt.Sprintf("%t", req.Lazy)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('dns_mode', ?)", mode); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if err := applyMosdnsConfig(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Mosdns failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})


}
