package main

import (
	"fmt"
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
