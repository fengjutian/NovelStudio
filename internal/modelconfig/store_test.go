package modelconfig_test

import (
	"encoding/json"
	"novelstudio/internal/modelconfig"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreKeepsAPIKeyLocalAndMasksPublicConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model-config.json")
	store := modelconfig.Store{Path: path}
	public, err := store.Save(modelconfig.UpdateInput{ActiveProvider: "deepseek", DeepSeek: modelconfig.UpdateProvider{BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash", APIKey: "secret-key", Enabled: true}, MiniMax: modelconfig.UpdateProvider{BaseURL: "https://api.minimaxi.com/v1", Model: "MiniMax-M2.7"}})
	if err != nil {
		t.Fatal(err)
	}
	if !public.DeepSeek.HasAPIKey {
		t.Fatal("expected key marker")
	}
	encoded, _ := json.Marshal(public)
	if strings.Contains(string(encoded), "secret-key") {
		t.Fatal("public config leaked API key")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "secret-key") {
		t.Fatal("local file did not retain API key")
	}
}

func TestBlankAPIKeyPreservesExistingValue(t *testing.T) {
	store := modelconfig.Store{Path: filepath.Join(t.TempDir(), "model-config.json")}
	input := modelconfig.UpdateInput{ActiveProvider: "minimax", DeepSeek: modelconfig.UpdateProvider{BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash"}, MiniMax: modelconfig.UpdateProvider{BaseURL: "https://api.minimaxi.com/v1", Model: "MiniMax-M2.7", APIKey: "first", Enabled: true}}
	if _, err := store.Save(input); err != nil {
		t.Fatal(err)
	}
	input.MiniMax.APIKey = ""
	if _, err := store.Save(input); err != nil {
		t.Fatal(err)
	}
	_, provider, err := store.Active()
	if err != nil || provider.APIKey != "first" {
		t.Fatalf("key not preserved: %#v %v", provider, err)
	}
}
