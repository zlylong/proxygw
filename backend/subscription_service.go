package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func decodeBase64Safe(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	for len(s)%4 != 0 {
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}

func syncSubscription(subID string, subURL string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(subURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	content := string(body)
	if !strings.Contains(content, "://") {
		if decoded, err := decodeBase64Safe(content); err == nil {
			content = string(decoded)
		}
	}

	lines := strings.Split(content, "\n")

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Soft delete old nodes
	tx.Exec("DELETE FROM nodes WHERE subscription_id = ?", subID)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "vmess://") {
			b64 := strings.TrimPrefix(line, "vmess://")
			decoded, err := decodeBase64Safe(b64)
			if err != nil {
				continue
			}
			var v map[string]interface{}
			if err := json.Unmarshal(decoded, &v); err != nil {
				continue
			}

			name := fmt.Sprintf("%v", v["ps"])
			address := fmt.Sprintf("%v", v["add"])
			uuid := fmt.Sprintf("%v", v["id"])
			
			portStr := fmt.Sprintf("%v", v["port"])
			port, _ := strconv.Atoi(portStr)

			params := map[string]interface{}{
				"type": "tcp",
			}
			if netType, ok := v["net"].(string); ok && netType != "" {
				params["type"] = netType
			}
			if tlsType, ok := v["tls"].(string); ok && tlsType != "" {
				params["security"] = tlsType
			}
			if sni, ok := v["sni"].(string); ok && sni != "" {
				params["sni"] = sni
			}
			if path, ok := v["path"].(string); ok && path != "" {
				params["path"] = path
			}
			if host, ok := v["host"].(string); ok && host != "" {
				params["host"] = host
			}

			pBytes, _ := json.Marshal(params)
			tx.Exec("INSERT INTO nodes (name, type, address, port, uuid, active, subscription_id, params) VALUES (?, ?, ?, ?, ?, 1, ?, ?)",
				name, "vmess", address, port, uuid, subID, string(pBytes))


		} else if strings.HasPrefix(line, "hy2://") {
			u, err := url.Parse(line)
			if err != nil {
				continue
			}
			uuid := u.User.Username()
			if uuid == "" {
				uuid, _ = u.User.Password()
			}
			address := u.Hostname()
			portStr := u.Port()
			port, _ := strconv.Atoi(portStr)
			name, _ := url.QueryUnescape(u.Fragment)
			if name == "" {
				name = address
			}

			q := u.Query()
			params := map[string]interface{}{
				"type": "tcp", // Hysteria2 is UDP based but Xray routing might need this
			}
			
			if sni := q.Get("sni"); sni != "" {
				params["sni"] = sni
			}
			if insecure := q.Get("insecure"); insecure != "" {
				params["insecure"] = insecure
			}
			if obfs := q.Get("obfs"); obfs != "" {
				params["obfs"] = obfs
			}
			if obfsPassword := q.Get("obfs-password"); obfsPassword != "" {
				params["obfsPassword"] = obfsPassword
			}

			pBytes, _ := json.Marshal(params)
			tx.Exec("INSERT INTO nodes (name, type, address, port, uuid, active, subscription_id, params) VALUES (?, ?, ?, ?, ?, 1, ?, ?)",
				name, "hysteria2", address, port, uuid, subID, string(pBytes))

		} else if strings.HasPrefix(line, "vless://") || strings.HasPrefix(line, "trojan://") {
			u, err := url.Parse(line)
			if err != nil {
				continue
			}
			protocol := u.Scheme
			uuid := u.User.Username()
			if protocol == "trojan" && uuid == "" {
				uuid, _ = u.User.Password()
			}
			address := u.Hostname()
			portStr := u.Port()
			port, _ := strconv.Atoi(portStr)
			name, _ := url.QueryUnescape(u.Fragment)
			if name == "" {
				name = address
			}

			q := u.Query()
			params := map[string]interface{}{}
			
			if t := q.Get("type"); t != "" {
				params["type"] = t
			} else {
				params["type"] = "tcp"
			}
			
			if sec := q.Get("security"); sec != "" {
				params["security"] = sec
			}
			if flow := q.Get("flow"); flow != "" {
				params["flow"] = flow
			}
			if sni := q.Get("sni"); sni != "" {
				params["sni"] = sni
			}
			if pbk := q.Get("pbk"); pbk != "" {
				params["pbk"] = pbk
			}
			if sid := q.Get("sid"); sid != "" {
				params["sid"] = sid
			}
			if fp := q.Get("fp"); fp != "" {
				params["fp"] = fp
			}

			pBytes, _ := json.Marshal(params)
			tx.Exec("INSERT INTO nodes (name, type, address, port, uuid, active, subscription_id, params) VALUES (?, ?, ?, ?, ?, 1, ?, ?)",
				name, protocol, address, port, uuid, subID, string(pBytes))
		}
	}

	return tx.Commit()
}
