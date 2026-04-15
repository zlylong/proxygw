package main

import (
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("main.go")
	s := string(b)
	s = strings.ReplaceAll(s, "fmt.Sprintf(\"{ addr: \"%s\", socks5: \"127.0.0.1:10808\" }\", p)", "fmt.Sprintf(\"{ addr: \\\"%s\\\", socks5: \\\"127.0.0.1:10808\\\" }\", p)")
	s = strings.ReplaceAll(s, "fmt.Sprintf(\"{ addr: \"%s\" }\", p)", "fmt.Sprintf(\"{ addr: \\\"%s\\\" }\", p)")
	s = strings.ReplaceAll(s, "return \"[{ addr: \"114.114.114.114\" }]\"", "return \"[{ addr: \\\"114.114.114.114\\\" }]\"")
	os.WriteFile("main.go", []byte(s), 0644)
}
