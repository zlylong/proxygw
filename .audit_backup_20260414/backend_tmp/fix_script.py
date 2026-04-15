import re

with open('main.go.bak', 'r') as f:
    code = f.read()

code = code.replace('"crypto/md5"\n\t"database/sql"', '"crypto/md5"\n\t"crypto/rand"\n\t"io"\n\t"database/sql"')
code = code.replace('var maxGeositeLogLines = 100', 'var maxGeositeLogLines = 100\nvar sessionToken string')

code = code.replace('func cronUpdater() {', '''
func downloadAndVerify(urlStr, dest string, minSize int64) error {
	resp, err := http.Get(urlStr)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return fmt.Errorf("bad status: %d", resp.StatusCode) }
	out, err := os.Create(dest)
	if err != nil { return err }
	defer out.Close()
	size, err := io.Copy(out, resp.Body)
	if err != nil { return err }
	if size < minSize { return fmt.Errorf("file too small: %d", size) }
	return nil
}

func updateGeoData() error {
	resp, err := http.Get("https://api.github.com/repos/Loyalsoldier/v2ray-rules-dat/releases/latest")
	if err != nil { return err }
	defer resp.Body.Close()
	var rel struct { TagName string `json:"tag_name"` }
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil { return err }
	tag := rel.TagName
	if tag == "" { return fmt.Errorf("no tag found") }

	baseUrl := "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/download/" + tag + "/"
	err1 := downloadAndVerify(baseUrl+"direct-list.txt", "/tmp/geosite_cn.txt", 10*1024)
	err2 := downloadAndVerify(baseUrl+"geoip.dat", "/tmp/geoip.dat", 1024*1024)
	err3 := downloadAndVerify(baseUrl+"geosite.dat", "/tmp/geosite.dat", 1024*1024)

	if err1 == nil && err2 == nil && err3 == nil {
		exec.Command("cp", "/tmp/geosite_cn.txt", "/usr/local/bin/geosite_cn.txt").Run()
		exec.Command("cp", "/tmp/geoip.dat", "/usr/local/bin/geoip.dat").Run()
		exec.Command("cp", "/tmp/geosite.dat", "/usr/local/bin/geosite.dat").Run()
		exec.Command("cp", "/usr/local/bin/geosite_cn.txt", "/root/proxygw/core/mosdns/").Run()
		exec.Command("cp", "/usr/local/bin/geoip.dat", "/root/proxygw/core/mosdns/").Run()
		os.WriteFile("/usr/local/bin/geodata.ver", []byte(tag), 0644)
		cacheMutex.Lock()
		cachedGeosite = nil
		cachedGeoip = nil
		cacheMutex.Unlock()
		applyMosdnsConfig()
		applyXrayConfig()
		return nil
	}
	return fmt.Errorf("download failed")
}
func cronUpdater() {''')

cron_loop = '''	for {
		time.Sleep(1 * time.Minute)
		var enabled, cronTime string
		db.QueryRow("SELECT value FROM settings WHERE key='cron_enabled'").Scan(&enabled)
		db.QueryRow("SELECT value FROM settings WHERE key='cron_time'").Scan(&cronTime)
		if cronTime == "" { cronTime = "04:00" }
		now := time.Now()
		if enabled == "true" && now.Format("15:04") == cronTime {
			log.Println("Running daily cron update for GeoData...")
			err1 := downloadAndVerify("https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt", "/tmp/geosite_cn.txt", 10*1024)
			err2 := downloadAndVerify("https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat", "/tmp/geoip.dat", 1024*1024)
			if err1 == nil && err2 == nil {
				exec.Command("cp", "/tmp/geosite_cn.txt", "/usr/local/bin/geosite_cn.txt").Run()
				exec.Command("cp", "/tmp/geoip.dat", "/usr/local/bin/geoip.dat").Run()
				exec.Command("cp", "/usr/local/bin/geosite_cn.txt", "/root/proxygw/core/mosdns/").Run()
				exec.Command("cp", "/usr/local/bin/geoip.dat", "/root/proxygw/core/mosdns/").Run()
				applyMosdnsConfig()
				applyXrayConfig()
				cacheMutex.Lock()
				cachedGeosite = nil
				cachedGeoip = nil
				cacheMutex.Unlock()
			}
			time.Sleep(61 * time.Minute)
		}
	}'''
code = re.sub(r'for \{\n\t\ttime\.Sleep\(24 \* time\.Hour\).*?cacheMutex\.Unlock\(\)\n\t\t\}\n\t\}', cron_loop, code, flags=re.DOTALL)

code = code.replace('func scheduleApply() {\n\tapplyMutex.Lock()', 'func scheduleApply() {\n\tapplyMosdnsConfig()\n\tapplyXrayConfig()\n\treturn\n\tapplyMutex.Lock()')

format_upstreams = '''func formatUpstreams(addrs string, useSocks bool) string {
	parts := strings.Split(addrs, ",")
	var res []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			if useSocks {
				res = append(res, fmt.Sprintf(`{ addr: "%s", socks5: "127.0.0.1:10808" }`, p))
			} else {
				res = append(res, fmt.Sprintf(`{ addr: "%s" }`, p))
			}
		}
	}
	if len(res) == 0 { return "[{ addr: \\"114.114.114.114\\" }]" }
	return "[" + strings.Join(res, ", ") + "]"
}

func applyMosdnsConfig() {'''
code = code.replace('func applyMosdnsConfig() {', format_upstreams)

mosdns_body_old = '''	var local, remote, lazyStr string
	db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazyStr)'''
mosdns_body_new = '''	var local, remote, lazyStr, mode string
	db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazyStr)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_mode'").Scan(&mode)
	if mode == "" { mode = "smart" }'''
code = code.replace(mosdns_body_old, mosdns_body_new)

seq_old = '''      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $forward_remote
`'''
seq_new = '''`
	if mode == "strict" {
		seqStr += `      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $forward_remote
`
	} else if mode == "fast" {
		seqStr += `      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $forward_local
`
	} else {
		seqStr += `      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $fallback
`
	}

	smartPlugins := ""
	if mode == "smart" {
		smartPlugins = `  - tag: geoip_cn
    type: ip_set
    args:
      files:
        - "geoip.dat"
  - tag: local_sequence
    type: sequence
    args:
      - exec: $forward_local
      - matches: [ "!resp_ip $geoip_cn" ]
        exec: drop_resp
  - tag: fallback
    type: fallback
    args:
      primary: local_sequence
      secondary: forward_remote
      threshold: 500
      always_standby: true
`
	}'''
code = code.replace(seq_old, seq_new)

mosdns_args_old = '''        - "geosite_cn.txt"
%s  - tag: forward_local
    type: forward
    args: { upstreams: [{ addr: "%s" }] }
  - tag: forward_remote
    type: forward
    args: { upstreams: [{ addr: "%s" }] }
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, local, remote, seqStr)'''
mosdns_args_new = '''        - "geosite_cn.txt"
%s%s  - tag: forward_local
    type: forward
    args: { upstreams: %s }
  - tag: forward_remote
    type: forward
    args: { upstreams: %s }
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)'''
code = code.replace(mosdns_args_old, mosdns_args_new)

xray_old = '''	"inbounds": []map[string]interface{}{
			{
				"port": 12345, "protocol": "dokodemo-door",
				"settings":       map[string]interface{}{"network": "tcp,udp", "followRedirect": true},
				"streamSettings": map[string]interface{}{"sockopt": map[string]string{"tproxy": "tproxy"}},
				"sniffing":       map[string]interface{}{"enabled": true, "destOverride": []string{"http", "tls"}},
			},
			{
				"listen": "127.0.0.1", "port": 10085, "protocol": "dokodemo-door",
				"settings": map[string]interface{}{"address": "127.0.0.1"},
				"tag":      "api_inbound",
			},
		},'''
xray_new = '''	"inbounds": []map[string]interface{}{
			{
				"port": 12345, "protocol": "dokodemo-door",
				"settings":       map[string]interface{}{"network": "tcp,udp", "followRedirect": true},
				"streamSettings": map[string]interface{}{"sockopt": map[string]string{"tproxy": "tproxy"}},
				"sniffing":       map[string]interface{}{"enabled": true, "destOverride": []string{"http", "tls"}},
			},
			{
				"listen": "127.0.0.1", "port": 10085, "protocol": "dokodemo-door",
				"settings": map[string]interface{}{"address": "127.0.0.1"},
				"tag":      "api_inbound",
			},
			{
				"listen": "127.0.0.1", "port": 10808, "protocol": "socks",
				"settings": map[string]interface{}{"auth": "noauth", "udp": true},
				"tag":      "dns_socks_inbound",
			},
		},'''
code = code.replace(xray_old, xray_new)

xray_routing_old = '''"rules": []map[string]interface{}{
				{"inboundTag": []string{"api_inbound"}, "outboundTag": "api", "type": "field"},
			},'''
xray_routing_new = '''"rules": []map[string]interface{}{
				{"inboundTag": []string{"api_inbound"}, "outboundTag": "api", "type": "field"},
				{"inboundTag": []string{"dns_socks_inbound"}, "outboundTag": "proxy", "type": "field"},
			},'''
code = code.replace(xray_routing_old, xray_routing_new)

xray_custom = '''		ntypeLow := strings.ToLower(ntype)

		if ntypeLow == "custom" {
			var customOutbound map[string]interface{}
			if err := json.Unmarshal([]byte(paramsStr), &customOutbound); err == nil && customOutbound != nil {
				outbound = customOutbound
				outbound["tag"] = fmt.Sprintf("proxy-%d", id)
				if _, ok := outbound["protocol"]; !ok { outbound["protocol"] = "freedom" }
			}
		} else if ntypeLow == "vmess" {'''
code = code.replace('ntypeLow := strings.ToLower(ntype)\n\n\t\tif ntypeLow == "vmess" {', xray_custom)

xray_vmess = '''			outbound = map[string]interface{}{
				"protocol": "vmess", "tag": fmt.Sprintf("proxy-%d", id),
				"settings": map[string]interface{}{
					"vnext": []map[string]interface{}{{
						"address": address, "port": port,
						"users": []map[string]interface{}{{"id": uuid, "alterId": 0}},
					}},
				},
			}
			var p map[string]interface{}
			if err := json.Unmarshal([]byte(paramsStr), &p); err == nil && len(p) > 0 {
				outbound["streamSettings"] = p
			}
		} else if ntypeLow == "vless" {'''
code = re.sub(r'outbound = map\[string\]interface\{\}\{\n\t\t\t\t"protocol": "vmess".*?\}\n\t\t\} else if ntypeLow == "vless" \{', xray_vmess, code, flags=re.DOTALL)

code = code.replace('b := make([]byte, 16)\n\trand.Read(b)\n\tsessionToken = fmt.Sprintf("%x", b)', '')
code = code.replace('func main() {', 'func main() {\n\tb := make([]byte, 16)\n\trand.Read(b)\n\tsessionToken = fmt.Sprintf("%x", b)')
code = code.replace('c.JSON(200, gin.H{"token": "proxygw-token-secret"})', 'c.JSON(200, gin.H{"token": sessionToken})')
code = code.replace('if c.GetHeader("Authorization") != "Bearer proxygw-token-secret" {', 'if c.GetHeader("Authorization") != "Bearer "+sessionToken {')

status_vars = '''		xrayVer := "Unknown"
		xrayVersionOut, err := exec.Command("xray", "version").Output()
		if err == nil {
			lines := strings.Split(string(xrayVersionOut), "\\n")
			if len(lines) > 0 {
				parts := strings.Split(lines[0], " ")
				if len(parts) >= 2 { xrayVer = parts[1] }
			}
		}
		geoVer := "Unknown"
		if data, err := os.ReadFile("/usr/local/bin/geodata.ver"); err == nil && len(data) > 0 {
			geoVer = strings.TrimSpace(string(data))
		} else if info, err := os.Stat("/usr/local/bin/geoip.dat"); err == nil {
			geoVer = info.ModTime().Format("2006-01-02")
		}'''
code = code.replace('upStats, _ := exec.Command("curl",', status_vars + '\n\t\tupStats, _ := exec.Command("curl",')
code = code.replace('"xray": xray, "ospf": frr, "mosdns": mosdns,', '"xray": xray, "ospf": frr, "mosdns": mosdns, "xrayVersion": xrayVer, "geoVersion": geoVer,')

geo_end = '''		comp := req.Component
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
		}'''
code = re.sub(r'comp := req\.Component\n\t\tif comp == "geodata".*?c\.JSON\(200, gin\.H\{"success": true\}\)\n\t\t\}', geo_end, code, flags=re.DOTALL)

mode_a_fixes = '''	// Sync systemd services with Mode
	var mode string
	db.QueryRow("SELECT value FROM settings WHERE key='mode'").Scan(&mode)
	if mode == "A" {
		exec.Command("systemctl", "start", "nftables").Run()
		exec.Command("systemctl", "stop", "frr").Run()
	} else {
		exec.Command("systemctl", "stop", "nftables").Run()
		exec.Command("systemctl", "start", "frr").Run()
	}
'''
code = code.replace('\tgo ospfController()', mode_a_fixes + '\n\tgo ospfController()')

code = code.replace('go watchDnsLogs()', 'go watchDnsLogs()\n\tgo initGeoCache()\n\tgo initGeoCache()')

init_geo = '''func initGeoCache() {
	cacheMutex.Lock()
	if len(cachedGeosite) == 0 { cachedGeosite = parseDatFile("/usr/local/bin/geosite.dat") }
	if len(cachedGeoip) == 0 { cachedGeoip = parseDatFile("/usr/local/bin/geoip.dat") }
	cacheMutex.Unlock()
}
func main() {'''
code = code.replace('func main() {', init_geo)

mode_a_endpoint = '''	api.POST("/mode", func(c *gin.Context) {
		var req struct{ Mode string }
		c.BindJSON(&req)
		db.Exec("UPDATE settings SET value=? WHERE key='mode'", req.Mode)
		if req.Mode == "A" {
			exec.Command("systemctl", "start", "nftables").Run()
			exec.Command("systemctl", "stop", "frr").Run()
		} else {
			exec.Command("systemctl", "stop", "nftables").Run()
			exec.Command("systemctl", "start", "frr").Run()
		}
		applyMosdnsConfig()
		applyXrayConfig()
		c.JSON(200, gin.H{"success": true})
	})'''
code = re.sub(r'\tapi\.POST\("/mode", func\(c \*gin\.Context\) \{.*?\t\}\)', mode_a_endpoint, code, flags=re.DOTALL)

with open('main.go', 'w') as f:
    f.write(code)
print("done")
