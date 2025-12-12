package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func HashServiceID(domain, pathPrefix string) string {
	input := fmt.Sprintf("%s:%s", domain, pathPrefix)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
