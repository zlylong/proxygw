package main

import (
	"bytes"
	"os"
)

func main() {
	b, _ := os.ReadFile("main.go")
	b = bytes.ReplaceAll(b, []byte("proxyg...cret"), []byte("proxygw-token-secret"))
	os.WriteFile("main.go", b, 0644)
}
