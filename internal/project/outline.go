package project

import (
	"errors"
	"regexp"
	"strings"
)

var markdownHeading = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)

type OutlineItem struct {
	Title    string `json:"title"`
	Level    int    `json:"level"`
	NodeType string `json:"nodeType"`
}

func ParseOutline(content string, projectType Type) ([]OutlineItem, error) {
	items := []OutlineItem{}
	minimum := 7
	for _, raw := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		match := markdownHeading.FindStringSubmatch(strings.TrimSpace(raw))
		if len(match) != 3 {
			continue
		}
		title := strings.TrimSpace(strings.Trim(match[2], "#"))
		if title == "" {
			continue
		}
		level := len(match[1])
		if level < minimum {
			minimum = level
		}
		items = append(items, OutlineItem{Title: title, Level: level})
	}
	if len(items) == 0 {
		return nil, errors.New("outline must contain Markdown headings")
	}
	for index := range items {
		items[index].Level = items[index].Level - minimum + 1
		items[index].NodeType = outlineNodeType(projectType, items[index].Level)
	}
	return items, nil
}

func outlineNodeType(projectType Type, level int) string {
	if projectType == TypeMovieCommentary {
		if level == 1 {
			return "ACT"
		}
		return "COMMENTARY_SEGMENT"
	}
	if projectType == TypeTVCommentary {
		if level == 1 {
			return "SEASON"
		}
		if level == 2 {
			return "EPISODE"
		}
		return "COMMENTARY_SEGMENT"
	}
	if projectType == TypeTechnicalDocument {
		if level == 1 {
			return "MODULE"
		}
		return "SECTION"
	}
	if level == 1 {
		return "VOLUME"
	}
	if level == 2 {
		return "CHAPTER"
	}
	return "SECTION"
}
