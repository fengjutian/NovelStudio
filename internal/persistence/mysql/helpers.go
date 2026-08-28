package mysql

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func newID(prefix string) string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("generate id: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(raw)
}

func contentHash(content string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(content))) }

func metadataJSON(value map[string]any) []byte {
	if value == nil {
		return nil
	}
	raw, _ := json.Marshal(value)
	return raw
}
