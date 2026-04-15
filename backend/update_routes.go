package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func updateGeodata() error {
	cmds := [][]string{
		{"wget", "-qO", "/usr/local/bin/geosite_cn.txt", "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"},
		{"wget", "-qO", "/usr/local/bin/geoip.dat", "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"},
		{"cp", "/usr/local/bin/geosite_cn.txt", "/root/proxygw/core/mosdns/"},
		{"cp", "/usr/local/bin/geoip.dat", "/root/proxygw/core/mosdns/"},
	}
	for _, c := range cmds {
		if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
			return err
		}
	}
	if err := exec.Command("systemctl", "restart", "mosdns", "xray").Run(); err != nil {
		return err
	}
	cacheMutex.Lock()
	cachedGeosite = nil
	cachedGeoip = nil
	cacheMutex.Unlock()
	return nil
}

func registerUpdateRoutes(api *gin.RouterGroup) {
	api.GET("/xray/versions", func(c *gin.Context) {
		resp, err := http.Get("https://api.github.com/repos/XTLS/Xray-core/releases")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch releases"})
			return
		}
		defer resp.Body.Close()

		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse releases"})
			return
		}

		var tags []string
		for _, r := range releases {
			if strings.TrimSpace(r.TagName) != "" {
				tags = append(tags, r.TagName)
			}
		}
		c.JSON(http.StatusOK, gin.H{"versions": tags})
	})

	api.POST("/update/:component", func(c *gin.Context) {
		comp := c.Param("component")
		switch comp {
		case "geodata":
			if err := updateGeodata(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
				return
			}
		case "xray":
			var req struct {
				Version string `json:"version"`
			}
			_ = c.ShouldBindJSON(&req)
			downloadURL, err := buildXrayDownloadURL(req.Version)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid version"})
				return
			}

			if err := exec.Command("cp", "/usr/local/bin/xray", "/usr/local/bin/xray.bak").Run(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "backup failed"})
				return
			}
			if err := exec.Command("wget", "-qO", "/tmp/xray.zip", downloadURL).Run(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "download failed"})
				return
			}
			if err := exec.Command("unzip", "-qo", "/tmp/xray.zip", "-d", "/tmp/xray").Run(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "unzip failed"})
				return
			}
			if err := exec.Command("install", "-m", "755", "/tmp/xray/xray", "/usr/local/bin/xray").Run(); err != nil {
				_ = exec.Command("cp", "/usr/local/bin/xray.bak", "/usr/local/bin/xray").Run()
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "install failed"})
				return
			}
			if err := exec.Command("systemctl", "restart", "xray").Run(); err != nil {
				_ = exec.Command("cp", "/usr/local/bin/xray.bak", "/usr/local/bin/xray").Run()
				_ = exec.Command("systemctl", "restart", "xray").Run()
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "restart failed, rolled back"})
				return
			}
		case "rollback_xray":
			if err := exec.Command("cp", "/usr/local/bin/xray.bak", "/usr/local/bin/xray").Run(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "rollback copy failed"})
				return
			}
			if err := exec.Command("systemctl", "restart", "xray").Run(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "rollback restart failed"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "unsupported component"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}
