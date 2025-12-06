package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
)

func HashServiceID(domain, pathPrefix string) string {
	input := fmt.Sprintf("%s:%s", domain, pathPrefix)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func ParseServices(ports []string, scheme string) ([]*url.URL, error) {
	out := make([]*url.URL, 0, len(ports))
	for _, p := range ports {
		u, err := url.Parse(scheme + "://localhost:" + p)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}
