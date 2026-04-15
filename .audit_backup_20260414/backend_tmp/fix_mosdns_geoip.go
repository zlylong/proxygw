package main
import (
	"os"
	"strings"
)
func main() {
	b, _ := os.ReadFile("main.go.bak")
	code := string(b)
	
	// Add formatUpstreams with socks
	fu_old := "func applyMosdnsConfig() {"
	fu_new := "func formatUpstreams(addrs string, useSocks bool) string {\n" +
		"\tparts := strings.Split(addrs, \",\")\n" +
		"\tvar res []string\n" +
		"\tfor _, p := range parts {\n" +
		"\t\tp = strings.TrimSpace(p)\n" +
		"\t\tif p != \"\" {\n" +
		"\t\t\tif useSocks {\n" +
		"\t\t\t\tres = append(res, \"{\" + \" addr: \\\"\" + p + \"\\\", socks5: \\\"127.0.0.1:10808\\\" }\")\n" +
		"\t\t\t} else {\n" +
		"\t\t\t\tres = append(res, \"{\" + \" addr: \\\"\" + p + \"\\\" }\")\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"\tif len(res) == 0 { return \"[{ addr: \\\"114.114.114.114\\\" }]\" }\n" +
		"\treturn \"[\" + strings.Join(res, \", \") + \"]\"\n" +
		"}\n\n" +
		"func applyMosdnsConfig() {"
	code = strings.Replace(code, fu_old, fu_new, 1)

	// Fix mosdns body config
	mb_old := "var local, remote, lazyStr string\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_local'\").Scan(&local)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_remote'\").Scan(&remote)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_lazy'\").Scan(&lazyStr)"
	mb_new := "var local, remote, lazyStr, mode string\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_local'\").Scan(&local)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_remote'\").Scan(&remote)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_lazy'\").Scan(&lazyStr)\n\tdb.QueryRow(\"SELECT value FROM settings WHERE key='dns_mode'\").Scan(&mode)\n\tif mode == \"\" { mode = \"smart\" }"
	code = strings.Replace(code, mb_old, mb_new, 1)

	// Fix fallback logic
	seq_old := "      - matches: [ qname  ]\n        exec: \n      - matches: [ qname  ]\n        exec: \n      - exec: \n\n\tsmartPlugins := \"\"\n\tif mode == \"strict\" {\n\t\tseqStr += \n\t} else if mode == \"fast\" {\n\t\tseqStr += \n\t} else {\n\t\tseqStr += \n\t\tsmartPlugins = \n\t}"
	code = strings.Replace(code, seq_old, seq_new, 1)

	// Fix args substitution
	args_old := "        - \"geosite_cn.txt\"\n%s  - tag: forward_local\n    type: forward\n    args: { upstreams: [{ addr: \"%s\" }] }\n  - tag: forward_remote\n    type: forward\n    args: { upstreams: [{ addr: \"%s\" }] }\n  - tag: udp_server\n    type: udp_server\n    args:\n      entry: main_sequence\n      listen: 0.0.0.0:53\n, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)"
	code = strings.Replace(code, args_old, args_new, 1)

	// Fix file path
	code = strings.ReplaceAll(code, "\"proxy_domains.txt\"", "\"/root/proxygw/core/mosdns/proxy_domains.txt\"")

	os.WriteFile("main.go", []byte(code), 0644)
}
