package remote_deploy

import (
	"fmt"
	"net/url"
)

func GenerateWireGuardShareLink(clientPrivKey, serverIp string, serverPort int, serverPubKey, clientIp, reserved, remarks string, mtu int) string {
	base := fmt.Sprintf("wireguard://%s@%s:%d", clientPrivKey, serverIp, serverPort)
	v := url.Values{}
	v.Add("publickey", serverPubKey)
	if clientIp != "" {
		v.Add("address", clientIp)
	}
	if mtu > 0 {
		v.Add("mtu", fmt.Sprintf("%d", mtu))
	}
	if reserved != "" {
		v.Add("reserved", reserved)
	}
	return fmt.Sprintf("%s?%s#%s", base, v.Encode(), url.QueryEscape(remarks))
}
