package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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

func getRemoteFileContent(urlStr string) (string, error) {
	resp, err := httpClient.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func verifySHA256(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedHash) {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}

func getGeoDataVersionAndHash() (string, string, error) {
	resp, err := http.Get("https://api.github.com/repos/Loyalsoldier/v2ray-rules-dat/releases/latest")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}
	tag := release.TagName

	urlStr := "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/download/" + tag + "/rules.zip.sha256sum"
	content, err := getRemoteFileContent(urlStr)
	if err != nil {
		return tag, "", err
	}
	parts := strings.Fields(content)
	if len(parts) > 0 {
		return tag, parts[0], nil
	}
	return tag, "", fmt.Errorf("invalid hash file")
}

func getXrayHash(version string) (string, error) {
	urlStr := ""
	if version == "" || version == "latest" {
		urlStr = "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip.dgst"
	} else {
		urlStr = fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/Xray-linux-64.zip.dgst", version)
	}
	content, err := getRemoteFileContent(urlStr)
	if err != nil {
		return "", err
	}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SHA256(") {
			parts := strings.Split(line, "= ")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return "", fmt.Errorf("hash not found in dgst")
}

func downloadWithVerification(urlStr, dest, expectedHash string) error {
	resp, err := downloadClient.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	tmpPath := dest + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := verifySHA256(tmpPath, expectedHash); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, dest)
}

var downloadClient = &http.Client{ Timeout: 5 * time.Minute }
var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}
