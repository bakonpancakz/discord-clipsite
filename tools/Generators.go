package tools

import (
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"regexp"
	"strings"
)

// Generate a Base64 Encoded String used for Session Lookup
func GenerateToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

var (
	IDAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	IDMatcher  = regexp.MustCompile(`^([A-z0-9]{11})$`)
	IDLength   = 11
)

// Generate a Youtube-esque Video ID
func GenerateVideoID() string {
	var sb strings.Builder
	for i := 0; i < IDLength; i++ {
		nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(IDAlphabet))))
		sb.WriteByte(IDAlphabet[nBig.Int64()])
	}
	return sb.String()
}
