import re

with open('main.go', 'r') as f:
    code = f.read()

code = code.replace('''		c.JSON(200, gin.H{"success": true})
	})
	})''', '''		c.JSON(200, gin.H{"success": true})
	})''')

old_update = '''	api.POST("/update/:component", func(c *gin.Context) {
		comp := c.Param("component")
		if comp == "geodata" {
			exec.Command("sh", "-c", "wget -qO /usr/local/bin/geosite_cn.txt https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt && wget -qO /usr/local/bin/geoip.dat https://mirror.ghproxy.com/https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat && cp /usr/local/bin/geosite_cn.txt /root/proxygw/core/mosdns/ && cp /usr/local/bin/geoip.dat /root/proxygw/core/mosdns/ && systemctl restart mosdns xray").Run()
			cacheMutex.Lock()
			cachedGeosite = nil
			cachedGeoip = nil
			cacheMutex.Unlock()
		} else if comp == "xray" {
			exec.Command("cp", "/usr/local/bin/xray", "/usr/local/bin/xray.bak").Run()
			exec.Command("sh", "-c", "wget -qO /tmp/xray.zip https://mirror.ghproxy.com/https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip && unzip -qo /tmp/xray.zip -d /tmp/xray && install -m 755 /tmp/xray/xray /usr/local/bin/xray && systemctl restart xray").Run()
		} else if comp == "rollback_xray" {
			exec.Command("cp", "/usr/local/bin/xray.bak", "/usr/local/bin/xray").Run()
			exec.Command("systemctl", "restart", "xray").Run()
		}
		c.JSON(200, gin.H{"success": true})
	})'''

new_update = '''	api.GET("/xray/versions", func(c *gin.Context) {
		resp, err := http.Get("https://api.github.com/repos/XTLS/Xray-core/releases")
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch releases"})
			return
		}
		defer resp.Body.Close()
		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			c.JSON(500, gin.H{"error": "Failed to parse releases"})
			return
		}
		var tags []string
		for _, r := range releases {
			tags = append(tags, r.TagName)
		}
		c.JSON(200, gin.H{"versions": tags})
	})

	api.POST("/components/update", func(c *gin.Context) {
		var req struct{ Component string }
		c.BindJSON(&req)
		comp := req.Component
		if comp == "geodata" {
			err := updateGeoData()
			if err == nil { c.JSON(200, gin.H{"success": true}) } else { c.JSON(500, gin.H{"success": false, "error": err.Error()}) }
		} else if comp == "xray" {
			var req struct { Version string `json:"version"` }
			c.ShouldBindJSON(&req)
			downloadUrl := "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip"
			if req.Version != "" && req.Version != "latest" { downloadUrl = fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/Xray-linux-64.zip", req.Version) }
			err := downloadAndVerify(downloadUrl, "/tmp/xray.zip", 5*1024*1024)
			if err == nil {
				exec.Command("cp", "/usr/local/bin/xray", "/usr/local/bin/xray.bak").Run()
				exec.Command("unzip", "-qo", "/tmp/xray.zip", "-d", "/tmp/xray").Run()
				exec.Command("install", "-m", "755", "/tmp/xray/xray", "/usr/local/bin/xray").Run()
				exec.Command("systemctl", "restart", "xray").Run()
				c.JSON(200, gin.H{"success": true})
			} else { c.JSON(500, gin.H{"success": false, "error": "download failed"}) }
		} else if comp == "rollback_xray" {
			exec.Command("cp", "/usr/local/bin/xray.bak", "/usr/local/bin/xray").Run()
			exec.Command("systemctl", "restart", "xray").Run()
			c.JSON(200, gin.H{"success": true})
		}
	})'''
code = code.replace(old_update, new_update)

code = code.replace('"crypto/md5"\n\t"crypto/rand"\n\t"io"\n\t"database/sql"', '"crypto/md5"\n\t"crypto/rand"\n\t"io"\n\t"database/sql"')
if '"crypto/rand"' not in code:
    code = code.replace('"crypto/md5"\n\t"database/sql"', '"crypto/md5"\n\t"crypto/rand"\n\t"io"\n\t"database/sql"')

if 'smartPlugins :=' in code and 'formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)' not in code:
    code = code.replace('lazyCache, local, remote, seqStr)', 'lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)')

if 'var sessionToken string' not in code:
    code = code.replace('var maxGeositeLogLines = 100', 'var maxGeositeLogLines = 100\nvar sessionToken string')

with open('main.go', 'w') as f:
    f.write(code)
print("patch_final done")
