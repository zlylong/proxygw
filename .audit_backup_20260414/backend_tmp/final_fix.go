package main
import "os"
import "strings"
func main() {
    b, _ := os.ReadFile("main.go")
    s := string(b)
    s = strings.ReplaceAll(s, "{ addr: \\\"%s\\\", socks5: \\\"127.0.0.1:10808\\\" }", "{ addr: \"%s\", socks5: \"127.0.0.1:10808\" }")
    s = strings.ReplaceAll(s, "{ addr: \\\"%s\\\" }", "{ addr: \"%s\" }")
    s = strings.ReplaceAll(s, "[{ addr: \\\"114.114.114.114\\\" }]", "[{ addr: \"114.114.114.114\" }]")
    os.WriteFile("main.go.new", []byte(s), 0644)
}
