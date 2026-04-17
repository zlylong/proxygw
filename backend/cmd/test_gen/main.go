package main

import (
	"fmt"
	"proxygw/remote_deploy"
)

func main() {
	wp, wu, err := remote_deploy.GenerateWireGuardKeys()
	fmt.Println("WG:", wp, wu, err)
	rp, ru, err := remote_deploy.GenerateXrayRealityKeys()
	fmt.Println("Xray:", rp, ru, err)
}
