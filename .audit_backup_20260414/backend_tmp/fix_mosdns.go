package main

import (
	"os"
	"strings"
	"regexp"
)

func main() {
	b, _ := os.ReadFile("main.go.bak")
	code := string(b)
	
	// formatUpstreams
	oldFu := "func applyMosdnsConfig() {"
	newFu := "func formatUpstreams(addrs string, useSocks bool) string {\n\tparts := strings.Split(addrs, \",\")\n\tvar res []string\n\tfor _, p := range parts {\n\t\tp = strings.TrimSpace(p)\n\t\tif p != \"\" {\n\t\t\tif useSocks {\n\t\t\t\tres = append(res, \"{\"+\" addr: \\\"\"+p+\"\\\", socks5: \\\"127.0.0.1:10808\\\" }\")\n\t\t\t} else {\n\t\t\t\tres = append(res, \"{\"+\" addr: \\\"\"+p+\"\\\" }\")\n\t\t\t}\n\t\t}\n\t}\n\tif len(res) == 0 { return \"[{ addr: \\\"114.114.114.114\\\" }]\" }\n\treturn \"[\" + strings.Join(res, \", \") + \"]\"\n}\n\nfunc applyMosdnsConfig() {"
	code = strings.Replace(code, oldFu, newFu, 1)

	// Mode and smartPlugins
	oldMd := "var local, remote, lazyStr string\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_local'\").Scan(&local)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_remote'\").Scan(&remote)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_lazy'\").Scan(&lazyStr)"
	newMd := "var local, remote, lazyStr, mode string\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_local'\").Scan(&local)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_remote'\").Scan(&remote)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_lazy'\").Scan(&lazyStr)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_mode'\").Scan(&mode)\n\tif mode == \"\" { mode = \"smart\" }"
	code = strings.Replace(code, oldMd, newMd, 1)

	oldSeq := "      - matches: [ qname  ]\n        exec: \n      - matches: [ qname  ]\n        exec: \n      - exec: \n\n\tsmartPlugins := \"\"\n\tif mode == \"strict\" {\n\t\tseqStr += \n\t} else if mode == \"fast\" {\n\t\tseqStr += \n\t} else {\n\t\tseqStr += \n\t\tsmartPlugins = \n\t}"
	code = strings.Replace(code, oldSeq, newSeq, 1)

	oldCfg := "        - \"geosite_cn.txt\"\n%s  - tag: forward_local\n    type: forward\n    args: { upstreams: [{ addr: \"%s\" }] }\n  - tag: forward_remote\n    type: forward\n    args: { upstreams: [{ addr: \"%s\" }] }\n  - tag: udp_server\n    type: udp_server\n    args:\n      entry: main_sequence\n      listen: 0.0.0.0:53\n, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)"
	code = strings.Replace(code, oldCfg, newCfg, 1)
	
	code = strings.ReplaceAll(code, "\"proxy_domains.txt\"", "\"/root/proxygw/core/mosdns/proxy_domains.txt\"")

	// 2. Add Xray inbound
	xInOld := "\"inbounds\": []map[string]interface{}{\n\t\t\t{\n\t\t\t\t\"port\": 12345, \"protocol\": \"dokodemo-door\",\n\t\t\t\t\"settings\":       map[string]interface{}{\"network\": \"tcp,udp\", \"followRedirect\": true},\n\t\t\t\t\"streamSettings\": map[string]interface{}{\"sockopt\": map[string]string{\"tproxy\": \"tproxy\"}},\n\t\t\t\t\"sniffing\":       map[string]interface{}{\"enabled\": true, \"destOverride\": []string{\"http\", \"tls\"}},\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"listen\": \"127.0.0.1\", \"port\": 10085, \"protocol\": \"dokodemo-door\",\n\t\t\t\t\"settings\": map[string]interface{}{\"address\": \"127.0.0.1\"},\n\t\t\t\t\"tag\":      \"api_inbound\",\n\t\t\t},\n\t\t},"
	xInNew := "\"inbounds\": []map[string]interface{}{\n\t\t\t{\n\t\t\t\t\"port\": 12345, \"protocol\": \"dokodemo-door\",\n\t\t\t\t\"settings\":       map[string]interface{}{\"network\": \"tcp,udp\", \"followRedirect\": true},\n\t\t\t\t\"streamSettings\": map[string]interface{}{\"sockopt\": map[string]string{\"tproxy\": \"tproxy\"}},\n\t\t\t\t\"sniffing\":       map[string]interface{}{\"enabled\": true, \"destOverride\": []string{\"http\", \"tls\"}},\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"listen\": \"127.0.0.1\", \"port\": 10085, \"protocol\": \"dokodemo-door\",\n\t\t\t\t\"settings\": map[string]interface{}{\"address\": \"127.0.0.1\"},\n\t\t\t\t\"tag\":      \"api_inbound\",\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"listen\": \"127.0.0.1\", \"port\": 10808, \"protocol\": \"socks\",\n\t\t\t\t\"settings\": map[string]interface{}{\"auth\": \"noauth\", \"udp\": true},\n\t\t\t\t\"tag\":      \"dns_socks_inbound\",\n\t\t\t},\n\t\t},"
	code = strings.Replace(code, xInOld, xInNew, 1)

	xRtOld := "\"rules\": []map[string]interface{}{\n\t\t\t\t{\"inboundTag\": []string{\"api_inbound\"}, \"outboundTag\": \"api\", \"type\": \"field\"},\n\t\t\t},"
	xRtNew := "\"rules\": []map[string]interface{}{\n\t\t\t\t{\"inboundTag\": []string{\"api_inbound\"}, \"outboundTag\": \"api\", \"type\": \"field\"},\n\t\t\t\t{\"inboundTag\": []string{\"dns_socks_inbound\"}, \"outboundTag\": \"proxy\", \"type\": \"field\"},\n\t\t\t},"
	code = strings.Replace(code, xRtOld, xRtNew, 1)

	// Custom
	xcOld := "ntypeLow := strings.ToLower(ntype)\n\n\t\tif ntypeLow == \"vmess\" {"
	xcNew := "ntypeLow := strings.ToLower(ntype)\n\n\t\tif ntypeLow == \"custom\" {\n\t\t\tvar customOutbound map[string]interface{}\n\t\t\tif err := json.Unmarshal([]byte(paramsStr), &customOutbound); err == nil && customOutbound != nil {\n\t\t\t\toutbound = customOutbound\n\t\t\t\toutbound[\"tag\"] = fmt.Sprintf(\"proxy-%d\", id)\n\t\t\t\tif _, ok := outbound[\"protocol\"]; !ok { outbound[\"protocol\"] = \"freedom\" }\n\t\t\t}\n\t\t} else if ntypeLow == \"vmess\" {"
	code = strings.Replace(code, xcOld, xcNew, 1)

	// Stream settings for vmess
	reVmess := regexp.MustCompile()
	xvmNew := 
	code = reVmess.ReplaceAllString(code, xvmNew)

	os.WriteFile("main.go", []byte(code), 0644)
}
