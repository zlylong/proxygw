package main

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func buildXrayDownloadURL(version string) (string, error) {
	base := "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip"
	ver := strings.TrimSpace(version)
	if ver == "" || ver == "latest" {
		return base, nil
	}
	matched, _ := regexp.MatchString(`^v[0-9A-Za-z._-]+$`, ver)
	if !matched {
		return "", fmt.Errorf("invalid version")
	}
	return fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/Xray-linux-64.zip", ver), nil
}

func parseXrayVersionOutput(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return "Unknown"
	}
	parts := strings.Fields(lines[0])
	if len(parts) >= 2 {
		return parts[1]
	}
	return "Unknown"
}

func parsePortValue(v interface{}) int {
	portInt := 443
	switch p := v.(type) {
	case float64:
		portInt = int(p)
	case string:
		if parsed, err := strconv.Atoi(p); err == nil {
			portInt = parsed
		}
	}
	return portInt
}

func isValidIPOrCIDR(v string) bool {
	s := strings.TrimSpace(v)
	if s == "" {
		return false
	}
	if strings.Contains(s, "/") {
		_, _, err := net.ParseCIDR(s)
		return err == nil
	}
	return net.ParseIP(s) != nil
}

func sanitizeUpstreamItem(addr string) (string, bool) {
	a := strings.TrimSpace(addr)
	if a == "" {
		return "", false
	}
	if strings.ContainsAny(a, "\"\n\r") {
		return "", false
	}
	return a, true
}

func normalizeUpstreamCSV(raw string) (string, bool) {
	parts := strings.Split(raw, ",")
	var cleaned []string
	for _, p := range parts {
		item, ok := sanitizeUpstreamItem(p)
		if !ok {
			continue
		}
		cleaned = append(cleaned, item)
	}
	if len(cleaned) == 0 {
		return "", false
	}
	return strings.Join(cleaned, ","), true
}

func isTrustedOrigin(origin, host string) bool {
	if strings.TrimSpace(origin) == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	originHost := strings.Split(u.Host, ":")[0]
	requestHost := strings.Split(host, ":")[0]
	return originHost != "" && requestHost != "" && strings.EqualFold(originHost, requestHost)
}
