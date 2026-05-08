package redaction

import "strings"

const marker = "[REDACTED]"

func Redact(text string, values []string) string {
	if text == "" {
		return text
	}

	for _, value := range values {
		if len(value) < 3 {
			continue
		}
		text = strings.ReplaceAll(text, value, marker)
	}

	return text
}
