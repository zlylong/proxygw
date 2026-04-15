package main

import (
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("main.go")
	code := string(b)
	
	// fix the syntax error
	code = strings.ReplaceAll(code, "[{ addr: \"114.114.114.114\" }]", "[{ addr: \\\"114.114.114.114\\\" }]")
	
	code = strings.ReplaceAll(code, "{ addr: \"%s\", socks5: \"127.0.0.1:10808\" }", "{ addr: \\\"%s\\\", socks5: \\\"127.0.0.1:10808\\\" }")
	
	code = strings.ReplaceAll(code, "{ addr: \"%s\" }", "{ addr: \\\"%s\\\" }")

	os.WriteFile("main.go", []byte(code), 0644)
}
