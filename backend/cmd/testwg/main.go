package main

import (
	"fmt"
	"proxygw/remote_deploy"
)

func main() {
    res := remote_deploy.GenerateWGInstallScript(1234, "priv", "pub", "10.0.0.1/24")
    fmt.Println(res)
}
