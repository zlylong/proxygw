package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func updateGeodata() error {
	hashZip, err := getGeoDataHash()
	if err != nil {
		return fmt.Errorf("failed to fetch geodata hash: %v", err)
	}

	err = downloadWithVerification("https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/rules.zip", "/tmp/rules.zip", hashZip)
	if err != nil {
		return fmt.Errorf("geodata validation failed: %v", err)
	}

	cmds := [][]string{
		{"unzip", "-qo", "/tmp/rules.zip", "direct-list.txt", "geoip.dat", "-d", "/tmp/"},
		{"cp", "/tmp/direct-list.txt", "/root/proxygw/core/mosdns/geosite_cn.txt"},
		{"cp", "/tmp/geoip.dat", "/root/proxygw/core/mosdns/geoip.dat"},
	}
	for _, c := range cmds {
		if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
			return fmt.Errorf("extraction/copy failed: %v", err)
		}
	}
	if err := exec.Command("systemctl", "restart", "mosdns", "xray").Run(); err != nil {
		return fmt.Errorf("service restart failed: %v", err)
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
			hash, err := getXrayHash(req.Version)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to fetch hash"})
				return
			}

			if err := exec.Command("cp", "/usr/local/bin/xray", "/usr/local/bin/xray.bak").Run(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "backup failed"})
				return
			}

			if err := downloadWithVerification(downloadURL, "/tmp/xray.zip", hash); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("xray validation failed: %v", err)})
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
