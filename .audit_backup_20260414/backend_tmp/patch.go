package main
import "os"
import "strings"
func main() {
    b, _ := os.ReadFile("main.go")
    s := string(b)
    
    // Fix files in args
    s = strings.ReplaceAll(s, "\"proxy_domains.txt\"", "\"/root/proxygw/core/mosdns/proxy_domains.txt\"")
    s = strings.ReplaceAll(s, "        - \"geosite_cn.txt\"", "        - \"/root/proxygw/core/mosdns/geosite_cn.txt\"")
    s = strings.ReplaceAll(s, "        - \"geoip.dat\"", "        - \"/root/proxygw/core/mosdns/geoip_cn.txt\"")

    os.WriteFile("main.go", []byte(s), 0644)
}
