package validation

import "testing"

func TestResultSchemaUsesBooleanAdditionalProperties(t *testing.T) {
	var walk func(any)
	walk = func(value any) {
		switch current := value.(type) {
		case map[string]any:
			if additional, ok := current["additionalProperties"]; ok {
				if _, valid := additional.(bool); !valid {
					t.Fatalf("additionalProperties must be boolean for OpenAI-compatible providers, got %T", additional)
				}
			}
			for _, child := range current {
				walk(child)
			}
		case []any:
			for _, child := range current {
				walk(child)
			}
		}
	}
	walk(resultSchema())
}
