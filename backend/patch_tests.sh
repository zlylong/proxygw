sed -i '48s/want := .*/want := `[{ addr: "1.1.1.1", socks5: "127.0.0.1:10808" }, { addr: "8.8.8.8", socks5: "127.0.0.1:10808" }]`/' /root/proxygw/backend/helpers_test.go
