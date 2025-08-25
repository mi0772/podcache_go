package hashpath

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type PathParts [4]string

func Generate(key string) PathParts {
	h := sha256.New()
	h.Write([]byte(key))
	sha256Hash := hex.EncodeToString(h.Sum(nil)) // 64 caratteri hex

	return PathParts{
		sha256Hash[0:16],
		sha256Hash[16:32],
		sha256Hash[32:48],
		sha256Hash[48:64],
	}
}

func (p PathParts) String() string {
	return strings.Join(p[:], "/")
}

func PathFromKey(key string) string {
	return Generate(key).String()
}
