package game

import (
	"fmt"
	"regexp"
	"strings"
)

func buildItemMatcher(raw string) (func(string) bool, string, bool, error) {
	pattern := normalizePattern(raw)
	if pattern == "" {
		return nil, "", false, fmt.Errorf("empty pattern")
	}

	lower := strings.ToLower(pattern)
	if strings.HasPrefix(lower, "regex:") || strings.HasPrefix(lower, "re:") {
		idx := strings.Index(pattern, ":")
		body := strings.TrimSpace(pattern[idx+1:])
		re, err := regexp.Compile("(?i)" + body)
		if err != nil {
			return nil, pattern, true, err
		}
		return func(name string) bool { return re.MatchString(name) }, pattern, true, nil
	}

	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2 {
		body := pattern[1 : len(pattern)-1]
		re, err := regexp.Compile("(?i)" + body)
		if err != nil {
			return nil, pattern, true, err
		}
		return func(name string) bool { return re.MatchString(name) }, pattern, true, nil
	}

	hasWildcard := strings.ContainsAny(pattern, "*?")

	// Prepare the search key by removing wildcards then normalizing
	clean := strings.ReplaceAll(pattern, "*", "")
	clean = strings.ReplaceAll(clean, "?", "")
	searchKey := normalizeMatchKey(clean)

	return func(name string) bool {
		return strings.Contains(normalizeMatchKey(name), searchKey)
	}, pattern, hasWildcard, nil
}

func normalizePattern(raw string) string {
	pattern := strings.TrimSpace(raw)
	runes := []rune(pattern)
	if len(runes) < 2 {
		return pattern
	}
	first := runes[0]
	last := runes[len(runes)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') ||
		(first == '“' && last == '”') || (first == '‘' && last == '’') {
		return strings.TrimSpace(string(runes[1 : len(runes)-1]))
	}
	return pattern
}

func normalizeMatchKey(value string) string {
	if value == "" {
		return ""
	}
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\'', '“', '”', '‘', '’':
			return -1
		default:
			return r
		}
	}, trimmed)
	trimmed = strings.ToLower(trimmed)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}
