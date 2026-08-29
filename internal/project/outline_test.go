package project_test

import (
	"novelstudio/internal/project"
	"testing"
)

func TestParseOutlineNormalizesHeadingLevels(t *testing.T) {
	items, err := project.ParseOutline("## 第一卷\n### 第一章\n#### 第一节", project.TypeNovel)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].NodeType != "VOLUME" || items[1].NodeType != "CHAPTER" || items[2].Level != 3 {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestParseOutlineSupportsTechnicalDocuments(t *testing.T) {
	items, err := project.ParseOutline("# API\n## Authentication", project.TypeTechnicalDocument)
	if err != nil || items[0].NodeType != "MODULE" || items[1].NodeType != "SECTION" {
		t.Fatalf("unexpected items: %#v, %v", items, err)
	}
}
