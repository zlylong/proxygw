package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var sessionToken string

var (
	cachedGeosite []string
	cachedGeoip   []string
	cacheMutex    sync.Mutex
)
var ospfLogs []string
var ospfLogsMu sync.RWMutex

func addOspfLog(msg string) {
	ospfLogsMu.Lock()
	defer ospfLogsMu.Unlock()
	ospfLogs = append([]string{time.Now().Format("15:04:05") + " " + msg}, ospfLogs...)
	if len(ospfLogs) > 50 {
		ospfLogs = ospfLogs[:50]
	}
}

func getOspfLogsSnapshot() []string {
	ospfLogsMu.RLock()
	defer ospfLogsMu.RUnlock()
	out := make([]string, len(ospfLogs))
	copy(out, ospfLogs)
	return out
}

func parseDatFile(filename string) []string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	tags := make(map[string]bool)
	idx := 0
	for idx < len(data) {
		if data[idx] == 0x0A {
			idx++
			msgLen, shift := 0, 0
			for {
				if idx >= len(data) {
					break
				}
				b := data[idx]
				idx++
				msgLen |= (int(b&0x7F) << shift)
				if (b & 0x80) == 0 {
					break
				}
				shift += 7
			}

			endIdx := idx + msgLen
			if idx < endIdx && idx < len(data) && data[idx] == 0x0A {
				idx++
				strLen, shift := 0, 0
				for {
					if idx >= len(data) {
						break
					}
					b := data[idx]
					idx++
					strLen |= (int(b&0x7F) << shift)
					if (b & 0x80) == 0 {
						break
					}
					shift += 7
				}

				if idx+strLen <= endIdx && idx+strLen <= len(data) && strLen > 0 && strLen < 50 {
					tag := string(data[idx : idx+strLen])
					tag = strings.ToLower(tag)
					valid := true
					for _, c := range tag {
						if c < 32 || c > 126 || c == ' ' {
							valid = false
							break
						}
					}
					if valid {
						tags[tag] = true
					}
				}
			}
			idx = endIdx
		} else {
			idx++
		}
	}

	var res []string
	for k := range tags {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "../config/proxygw.db")
	if err != nil {
		log.Fatal(err)
	}

	tables := []string{
		`CREATE TABLE IF NOT EXISTS routes_table (
			ip TEXT PRIMARY KEY, domain TEXT, source TEXT,
			first_seen DATETIME, last_seen DATETIME, ttl INTEGER, status TEXT, miss_count INTEGER DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, grp TEXT, type TEXT, address TEXT, port INTEGER, uuid TEXT, active BOOLEAN DEFAULT 1, ping INTEGER DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT, type TEXT, value TEXT, policy TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT);`,
	}
	for _, t := range tables {
		db.Exec(t)
	}

	db.Exec("ALTER TABLE nodes ADD COLUMN params TEXT DEFAULT '{}'")
	db.Exec("ALTER TABLE nodes ADD COLUMN ping INTEGER DEFAULT 0")
	db.Exec("ALTER TABLE routes_table ADD COLUMN miss_count INTEGER DEFAULT 0")

	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('mode', 'B')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('dns_local', '114.114.114.114')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('dns_remote', '8.8.8.8')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('dns_lazy', 'true')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('dns_mode', 'smart')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('cron_enabled', 'true')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('cron_time', '04:00')")

	var count int
	db.QueryRow("SELECT count(*) FROM rules").Scan(&count)
	if count == 0 {
		db.Exec("INSERT INTO rules (type, value, policy) VALUES ('geosite', 'cn', 'direct')")
		db.Exec("INSERT INTO rules (type, value, policy) VALUES ('geosite', 'category-ads-all', 'block')")
		db.Exec("INSERT INTO rules (type, value, policy) VALUES ('geolocation', '!cn', 'proxy')")
	}

	ensurePasswordInitialized()
}

func ensurePasswordInitialized() {
	var pwdHash, legacyPwd string
	_ = db.QueryRow("SELECT value FROM settings WHERE key='password_hash'").Scan(&pwdHash)
	_ = db.QueryRow("SELECT value FROM settings WHERE key='password'").Scan(&legacyPwd)
	if strings.TrimSpace(pwdHash) != "" || strings.TrimSpace(legacyPwd) != "" {
		return
	}

	bootstrap := strings.TrimSpace(os.Getenv("PROXYGW_BOOTSTRAP_PASSWORD"))
	generated := false
	if bootstrap == "" {
		b := make([]byte, 12)
		if _, err := rand.Read(b); err == nil {
			bootstrap = fmt.Sprintf("%x", b)
			generated = true
		}
	}
	if strings.TrimSpace(bootstrap) == "" {
		log.Println("[SECURITY] password bootstrap failed: empty bootstrap password")
		return
	}

	hash, err := hashPassword(bootstrap)
	if err != nil {
		log.Printf("[SECURITY] password bootstrap hash failed: %v", err)
		return
	}
	if _, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('password_hash', ?)", hash); err != nil {
		log.Printf("[SECURITY] password bootstrap db write failed: %v", err)
		return
	}

	if generated {
		bootstrapPath := "../config/bootstrap_password.txt"
		if err := os.WriteFile(bootstrapPath, []byte(bootstrap+"\n"), 0600); err != nil {
			log.Printf("[SECURITY] initialized random bootstrap password (save failed: %v)", err)
		} else {
			log.Printf("[SECURITY] initialized random bootstrap password, saved to %s (change it immediately)", bootstrapPath)
		}
	} else {
		log.Println("[SECURITY] initialized password from PROXYGW_BOOTSTRAP_PASSWORD")
	}
}

func ospfController() {
	var lastUpdate time.Time
	const CoolingTime = 30 * time.Second
	const MaxBatch = 50

	for {
		time.Sleep(10 * time.Second)
		var mode string
		db.QueryRow("SELECT value FROM settings WHERE key='mode'").Scan(&mode)
		if mode != "B" {
			continue
		}

		if time.Since(lastUpdate) < CoolingTime {
			continue
		}
		updated := false

		db.Exec("UPDATE routes_table SET miss_count = miss_count + 1 WHERE status='published' AND datetime(last_seen, '+' || ttl || ' seconds') < datetime('now')")

		rowsDel, _ := db.Query("SELECT ip FROM routes_table WHERE status='published' AND miss_count >= 3 LIMIT ?", MaxBatch)
		var toDel []string
		for rowsDel.Next() {
			var ip string
			rowsDel.Scan(&ip)
			toDel = append(toDel, ip)
		}
		rowsDel.Close()

		for _, ip := range toDel {
			addOspfLog("[DEL] " + ip + " (Miss count >= 3)")
			exec.Command("vtysh", "-c", "conf t", "-c", fmt.Sprintf("no ip route %s 127.0.0.1 tag 100", func(s string) string {
				if strings.Contains(s, "/") {
					return s
				}
				return s + "/32"
			}(ip))).Run()
			db.Exec("DELETE FROM routes_table WHERE ip=?", ip)
			updated = true
		}

		rowsAdd, _ := db.Query("SELECT ip FROM routes_table WHERE status='candidate' AND first_seen <= datetime('now', '-60 seconds') LIMIT ?", MaxBatch)
		var toAdd []string
		for rowsAdd.Next() {
			var ip string
			rowsAdd.Scan(&ip)
			toAdd = append(toAdd, ip)
		}
		rowsAdd.Close()

		for _, ip := range toAdd {
			addOspfLog("[ADD] " + ip + " to published_set")
			exec.Command("vtysh", "-c", "conf t", "-c", fmt.Sprintf("ip route %s 127.0.0.1 tag 100", func(s string) string {
				if strings.Contains(s, "/") {
					return s
				}
				return s + "/32"
			}(ip))).Run()
			db.Exec("UPDATE routes_table SET status='published', last_seen=datetime('now'), miss_count=0 WHERE ip=?", ip)
			updated = true
		}

		if updated {
			lastUpdate = time.Now()
		}
	}
}

func watchDnsLogs() {
	var lastSize int64 = 0
	logPath := "/root/proxygw/core/mosdns/mosdns.log"
	for {
		time.Sleep(3 * time.Second)
		info, err := os.Stat(logPath)
		if err != nil {
			continue
		}
		if info.Size() < lastSize {
			lastSize = 0
		}
		if info.Size() == lastSize {
			continue
		}

		f, err := os.Open(logPath)
		if err != nil {
			continue
		}
		f.Seek(lastSize, 0)
		buf := make([]byte, info.Size()-lastSize)
		f.Read(buf)
		f.Close()
		lastSize = info.Size()

		lines := strings.Split(string(buf), "\n")
		for _, line := range lines {
			if strings.Contains(line, "forward_remote") && strings.Contains(line, "query:") {
				parts := strings.Split(line, "query: ")
				if len(parts) > 1 {
					domain := strings.TrimSuffix(strings.Split(parts[1], " ")[0], ".")
					go func(d string) {
						ips, err := net.LookupIP(d)
						if err == nil {
							for _, ip := range ips {
								if ipv4 := ip.To4(); ipv4 != nil {
									db.Exec("INSERT INTO routes_table (ip, domain, source, first_seen, last_seen, ttl, status, miss_count) VALUES (?, ?, 'dns', datetime('now'), datetime('now'), 3600, 'candidate', 0) ON CONFLICT(ip) DO UPDATE SET last_seen=datetime('now'), miss_count=0", ipv4.String(), d)
								}
							}
						}
					}(domain)
				}
			}
		}
	}
}

func cronUpdater() {
	for {
		time.Sleep(24 * time.Hour)
		var enabled string
		db.QueryRow("SELECT value FROM settings WHERE key='cron_enabled'").Scan(&enabled)
		if enabled == "true" {
			log.Println("Running daily cron update for GeoData...")
			if err := updateGeodata(); err != nil {
				log.Printf("[SECURITY] Cron update failed: %v", err)
			} else {
				log.Println("Cron update for GeoData completed securely.")
			}
		}
	}
}

var (
	applyTimer *time.Timer
	applyMutex sync.Mutex
	upgrader   = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return isTrustedOrigin(r.Header.Get("Origin"), r.Host) }}
)

func scheduleApply() {
	applyMutex.Lock()
	defer applyMutex.Unlock()
	if applyTimer != nil {
		applyTimer.Stop()
	}
	applyTimer = time.AfterFunc(3*time.Second, func() {
		applyMosdnsConfig()
		applyXrayConfig()
	})
}

func formatUpstreams(addrs string, useSocks bool) string {
	parts := strings.Split(addrs, ",")
	var items []string
	for _, p := range parts {
		clean, ok := sanitizeUpstreamItem(p)
		if !ok {
			continue
		}
		if useSocks {
			items = append(items, fmt.Sprintf(`{ addr: "%s", socks5: "127.0.0.1:10808" }`, clean))
		} else {
			items = append(items, fmt.Sprintf(`{ addr: "%s" }`, clean))
		}
	}
	if len(items) == 0 {
		if useSocks {
			return `[{ addr: "8.8.8.8", socks5: "127.0.0.1:10808" }]`
		}
		return `[{ addr: "223.5.5.5" }]`
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func applyMosdnsConfig() {
	var local, remote, lazyStr string
	db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazyStr)

	dRows, _ := db.Query("SELECT value FROM rules WHERE type='domain' AND policy LIKE 'proxy%'")
	var proxyDomains []string
	for dRows.Next() {
		var d string
		dRows.Scan(&d)
		proxyDomains = append(proxyDomains, d)
	}
	dRows.Close()
	os.WriteFile("../core/mosdns/proxy_domains.txt", []byte(strings.Join(proxyDomains, "\n")), 0644)

	config := renderMosdnsConfig(local, remote, lazyStr == "true")

	os.WriteFile("../core/mosdns/config.yaml", []byte(config), 0644)
	exec.Command("systemctl", "restart", "mosdns").Run()
}

func applyXrayConfig() error {
	config := buildBaseXrayConfig()

	rows, _ := db.Query("SELECT id, name, type, address, port, uuid, COALESCE(params, '{}') FROM nodes WHERE active=1")
	defer rows.Close()
	var proxyTags []string
	for rows.Next() {
		var name, ntype, address, uuid, paramsStr string
		var port, id int
		rows.Scan(&id, &name, &ntype, &address, &port, &uuid, &paramsStr)

		var outbound map[string]interface{}

		ntypeLow := strings.ToLower(ntype)

		if ntypeLow == "vmess" {
			outbound = map[string]interface{}{
				"protocol": "vmess", "tag": fmt.Sprintf("proxy-%d", id),
				"settings": map[string]interface{}{
					"vnext": []map[string]interface{}{{
						"address": address, "port": port,
						"users": []map[string]interface{}{{"id": uuid, "alterId": 0}},
					}},
				},
			}
		} else if ntypeLow == "vless" {
			var p map[string]string
			json.Unmarshal([]byte(paramsStr), &p)

			user := map[string]interface{}{"id": uuid, "encryption": "none"}
			if p["encryption"] != "" {
				user["encryption"] = p["encryption"]
			}
			if p["flow"] != "" {
				user["flow"] = p["flow"]
			}

			streamSettings := map[string]interface{}{
				"network": p["type"],
			}
			if streamSettings["network"] == "" {
				streamSettings["network"] = "tcp"
			}

			if p["security"] != "" {
				streamSettings["security"] = p["security"]
			}

			if p["security"] == "reality" {
				streamSettings["realitySettings"] = map[string]interface{}{
					"fingerprint": p["fp"],
					"serverName":  p["sni"],
					"publicKey":   p["pbk"],
					"shortId":     p["sid"],
					"spiderX":     "/",
				}
			} else if p["security"] == "tls" {
				streamSettings["tlsSettings"] = map[string]interface{}{
					"serverName": p["sni"],
				}
			}

			outbound = map[string]interface{}{
				"protocol": "vless", "tag": fmt.Sprintf("proxy-%d", id),
				"settings": map[string]interface{}{
					"vnext": []map[string]interface{}{{
						"address": address, "port": port,
						"users": []interface{}{user},
					}},
				},
				"streamSettings": streamSettings,
			}
		}

		if outbound != nil {
			config["outbounds"] = append(config["outbounds"].([]map[string]interface{}), outbound)
			proxyTags = append(proxyTags, fmt.Sprintf("proxy-%d", id))
		}
	}

	if len(proxyTags) > 0 {
		config["outbounds"] = append(config["outbounds"].([]map[string]interface{}), map[string]interface{}{
			"protocol": "freedom", "tag": "proxy",
		})
	}

	rRows, _ := db.Query("SELECT type, value, policy FROM rules")
	defer rRows.Close()
	rules := config["routing"].(map[string]interface{})["rules"].([]map[string]interface{})
	for rRows.Next() {
		var rtype, value, policy string
		rRows.Scan(&rtype, &value, &policy)
		rule := map[string]interface{}{"type": "field", "outboundTag": policy}

		if rtype == "geosite" || rtype == "domain" {
			rule["domain"] = []string{rtype + ":" + value}
			rules = append(rules, rule)
		} else if rtype == "geoip" || rtype == "ip" || rtype == "geolocation" {
			if rtype == "ip" {
				rule["ip"] = []string{value}
				rules = append(rules, rule)
			} else if rtype == "geolocation" {
				if strings.EqualFold(value, "private") {
					rule["ip"] = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "169.254.0.0/16", "100.64.0.0/10"}
				} else {
					rule["ip"] = []string{"geoip:" + value}
				}
				rules = append(rules, rule)
			} else {
				if strings.EqualFold(value, "private") {
					rule["ip"] = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "169.254.0.0/16", "100.64.0.0/10"}
				} else {
					rule["ip"] = []string{"geoip:" + value}
				}
				rules = append(rules, rule)
			}
		} else {
			// Skip invalid rules without domain/ip to prevent Xray crash
			continue
		}
	}

	if len(proxyTags) > 0 {
		for _, r := range rules {
			if r["outboundTag"] == "proxy" {
				r["outboundTag"] = proxyTags[0]
			}
		}
	}
	config["routing"].(map[string]interface{})["rules"] = rules

	// Sync static IP rules to OSPF
	staticRows, _ := db.Query("SELECT value FROM rules WHERE type='ip' AND policy LIKE 'proxy%'")
	var staticIPs []string
	staticIPsMap := make(map[string]bool)
	for staticRows.Next() {
		var ip string
		staticRows.Scan(&ip)
		staticIPs = append(staticIPs, ip)
		staticIPsMap[ip] = true
	}
	staticRows.Close()

	// Mark removed static rules for deletion by ospfController
	oldRows, _ := db.Query("SELECT ip FROM routes_table WHERE source='static'")
	var toDelete []string
	for oldRows.Next() {
		var ip string
		oldRows.Scan(&ip)
		if !staticIPsMap[ip] {
			toDelete = append(toDelete, ip)
		}
	}
	oldRows.Close()

	for _, ipStr := range toDelete {
		db.Exec("UPDATE routes_table SET miss_count=99, ttl=0, last_seen=datetime('now', '-1 hour') WHERE ip=?", ipStr)
	}

	for _, ipStr := range staticIPs {
		db.Exec("INSERT INTO routes_table (ip, domain, source, first_seen, last_seen, ttl, status, miss_count) VALUES (?, 'static_rule', 'static', datetime('now', '-61 seconds'), datetime('now'), 999999999, 'candidate', 0) ON CONFLICT(ip) DO UPDATE SET source='static', status='candidate', first_seen=datetime('now', '-61 seconds'), ttl=999999999, miss_count=0", ipStr)
	}

	configData, _ := json.MarshalIndent(config, "", "  ")

	os.WriteFile("../core/xray/config.json", configData, 0644)
	return exec.Command("systemctl", "restart", "xray").Run()
}


func getPrimarySubnet(ipStr string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ipnet.IP.String() == ipStr {
					network := ipnet.IP.Mask(ipnet.Mask)
					maskSize, _ := ipnet.Mask.Size()
					return fmt.Sprintf("%s/%d", network.String(), maskSize)
				}
			}
		}
	}
	return ""
}

func syncFRRConfig() {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return
	}
	ip := conn.LocalAddr().(*net.UDPAddr).IP.String()
	conn.Close()

	subnet := getPrimarySubnet(ip)
	if subnet == "" {
		return
	}

	b, err := os.ReadFile("/etc/frr/frr.conf")
	if err != nil {
		return
	}
	content := string(b)

	reRouter := regexp.MustCompile(`(?m)^\s*ospf router-id\s+\S+`)
	reNetwork := regexp.MustCompile(`(?m)^\s*network\s+\S+\s+area\s+0`)

	newContent := reRouter.ReplaceAllString(content, " ospf router-id "+ip)
	newContent = reNetwork.ReplaceAllString(newContent, " network "+subnet+" area 0")

	if newContent != content {
		log.Printf("[OSPF] Auto-updating FRR config: router-id=%s, network=%s", ip, subnet)
		os.WriteFile("/etc/frr/frr.conf", []byte(newContent), 0644)
		exec.Command("systemctl", "restart", "frr").Run()
	}
}

func main() {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		sessionToken = fmt.Sprintf("%x", b)
	} else {
		sessionToken = fmt.Sprintf("proxygw-%d", time.Now().UnixNano())
	}

	syncFRRConfig()
	initDB()
	go ospfController()
	go watchDnsLogs()
	go cronUpdater()

	r := gin.Default()
	registerAPIRoutes(r)

	r.Static("/ui", "../frontend/dist")
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/ui/") })

	log.Println("ProxyGW backend starting on :80")
	r.Run(":80")
}
