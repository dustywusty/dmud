package util

import "strings"

const (
	WelcomeBanner = `                                      
          _/                                  _/   
     _/_/_/  _/_/_/  _/_/    _/    _/    _/_/_/    
  _/    _/  _/    _/    _/  _/    _/  _/    _/     
 _/    _/  _/    _/    _/  _/    _/  _/    _/      
  _/_/_/  _/    _/    _/    _/_/_/    _/_/_/` + "\n\n"
)

// TagMessage prefixes a message with a protocol tag (e.g., "DMG|").
func TagMessage(tag, msg string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return msg
	}
	if strings.HasPrefix(msg, tag+"|") {
		return msg
	}
	return tag + "|" + msg
}

// TagMessageWithStatus prefixes a message with tag and status (e.g., "DMG|DEATH|").
func TagMessageWithStatus(tag, status, msg string) string {
	tag = strings.TrimSpace(tag)
	status = strings.TrimSpace(status)
	if tag == "" {
		return msg
	}
	prefix := tag + "|"
	if status != "" {
		prefix = tag + "|" + status + "|"
	}
	if strings.HasPrefix(msg, prefix) {
		return msg
	}
	return prefix + msg
}

// StripTag removes known message tags for legacy clients.
func StripTag(msg string) string {
	parts := strings.SplitN(msg, "|", 3)
	if len(parts) < 2 {
		return msg
	}
	if !isTag(parts[0]) {
		return msg
	}
	if len(parts) == 2 {
		return parts[1]
	}
	if isStatus(parts[1]) {
		return parts[2]
	}
	return parts[1] + "|" + parts[2]
}

// IsStateMessage returns true for STATE| protocol frames.
func IsStateMessage(msg string) bool {
	return strings.HasPrefix(msg, "STATE|")
}

func isTag(tag string) bool {
	switch tag {
	case "DMG", "SYS", "CHAT":
		return true
	default:
		return false
	}
}

func isStatus(status string) bool {
	switch status {
	case "DEATH":
		return true
	default:
		return false
	}
}
