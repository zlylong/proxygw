#!/bin/bash
awk '{gsub(/proxyg\.\.\.cret/,"proxygw-token-secret"); print}' main.go > main_new.go
mv main_new.go main.go
go build -o proxygw-backend main.go
systemctl restart proxygw
