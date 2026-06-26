package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// A rule is a compiled highlight (Value is a lipgloss color) or trigger (Value
// is a command to send when the pattern matches a line).
type rule struct {
	Pattern string
	Value   string
	re      *regexp.Regexp
}

func compileRules(cfgs []ruleConfig) []rule {
	var rules []rule
	for _, c := range cfgs {
		re, err := regexp.Compile(c.Pattern)
		if err != nil {
			continue // skip invalid patterns rather than failing startup
		}
		rules = append(rules, rule{Pattern: c.Pattern, Value: c.Value, re: re})
	}
	return rules
}

func rulesToConfig(rules []rule) []ruleConfig {
	if len(rules) == 0 {
		return nil
	}
	cfgs := make([]ruleConfig, len(rules))
	for i, r := range rules {
		cfgs[i] = ruleConfig{Pattern: r.Pattern, Value: r.Value}
	}
	return cfgs
}

// applyHighlights colorizes substrings matching any highlight rule. It is a
// no-op on text that already contains ANSI styling, to avoid nesting escapes.
func applyHighlights(s string, highlights []rule) string {
	if len(highlights) == 0 || strings.Contains(s, "\x1b") {
		return s
	}
	for _, h := range highlights {
		color := lipgloss.NewStyle().Foreground(lipgloss.Color(h.Value)).Bold(true)
		s = h.re.ReplaceAllStringFunc(s, func(m string) string {
			return color.Render(m)
		})
	}
	return s
}

// matchTriggers returns the commands to fire for a line (already ANSI-stripped).
func matchTriggers(s string, triggers []rule) []string {
	var out []string
	for _, t := range triggers {
		if t.re.MatchString(s) {
			out = append(out, t.Value)
		}
	}
	return out
}
