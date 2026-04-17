package remote_deploy

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// GenerateWireGuardKeys returns private and public keys using pure Go
func GenerateWireGuardKeys() (string, string, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}
	return key.String(), key.PublicKey().String(), nil
}

// GenerateXrayRealityKeys returns REALITY private and public keys using pure Go X25519
func GenerateXrayRealityKeys() (string, string, error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	// X25519 clamping
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	priv64 := base64.RawURLEncoding.EncodeToString(priv[:])
	pub64 := base64.RawURLEncoding.EncodeToString(pub[:])

	return priv64, pub64, nil
}

// GenerateUUID generates a random UUID v4
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateShortId generates a random 8-character hex string for REALITY shortId
func GenerateShortId() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
