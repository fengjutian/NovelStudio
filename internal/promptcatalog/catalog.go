package promptcatalog

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/*.md
var files embed.FS

type Catalog struct{ Dir string }

func (c Catalog) Load(name string) (content, version string, err error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if strings.TrimSpace(c.Dir) != "" {
		raw, readErr := os.ReadFile(filepath.Join(c.Dir, name+".v1.md"))
		if readErr == nil {
			return strings.TrimSpace(string(raw)), "v1", nil
		}
		if !os.IsNotExist(readErr) {
			return "", "", fmt.Errorf("load external prompt %s: %w", name, readErr)
		}
	}
	raw, err := files.ReadFile("templates/" + name + ".v1.md")
	if err != nil {
		return "", "", fmt.Errorf("load prompt %s: %w", name, err)
	}
	return strings.TrimSpace(string(raw)), "v1", nil
}
