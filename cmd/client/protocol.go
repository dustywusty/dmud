package main

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"dmud/internal/util"
)

// fxEffect is one active status effect with its countdown, for the FX bar.
type fxEffect struct {
	Name      string
	Kind      string // buff | dot | heal | control
	Remaining int    // seconds; 0 = permanent
	Magnitude int
}

// statusInfo is the parsed form of a STATE| frame.
type statusInfo struct {
	HP, MaxHP int
	EP, MaxEP int // endurance
	Level     int
	XP, ReqXP int
	Area      string
	Gold      int
	Effects   []string
	Fx        []fxEffect // structured effects with timers (preferred over Effects)
	Stats     map[string]int // STR/DEX/CON/INT/WIS
	Race      string
	HasHP     bool
	HasEP     bool
	HasXP     bool
}

// roomInfo is the parsed form of a room-description chunk.
type roomInfo struct {
	Title     string
	Exits     []string
	Occupants []string // players, NPCs, corpses, and items present in the room
}

// roomContents is the decoded EVENT|{room.contents} push: a live "who/what is
// here" list that updates without anyone looking.
type roomContents struct {
	Players []string `json:"players"`
	NPCs    []string `json:"npcs"`
	Items   []string `json:"items"`
	Corpses []string `json:"corpses"`
}

// commsMsg is a decoded EVENT|{comms} chat push.
type commsMsg struct {
	Channel string `json:"channel"` // say | shout | tell
	From    string `json:"from"`
	To      string `json:"to"`
	Self    bool   `json:"self"`
	Text    string `json:"text"`
}

// macroBinding is one suggested hotkey from an EVENT|{macros} push (e.g. when
// picking a class), applied to the macro bar.
type macroBinding struct {
	Slot  int    `json:"slot"`
	Label string `json:"label"`
	Cmd   string `json:"cmd"`
}

// combatMsg is a decoded EVENT|{combat} event with the target's health.
type combatMsg struct {
	Target    string `json:"target"`
	Damage    int    `json:"damage"`
	TargetHP  int    `json:"target_hp"`
	TargetMax int    `json:"target_max"`
	Killed    bool   `json:"killed"`
}

// routed is the result of classifying one server chunk: where each part of it
// should be displayed in the UI.
type routed struct {
	status   *statusInfo    // non-nil: update the status bar (not echoed elsewhere)
	room     *roomInfo      // non-nil: update the room panel (body still goes to main)
	info     *roomInfo      // non-nil: authoritative room name/exits (room.info, silent)
	contents *roomContents  // non-nil: replace the live HERE list (silent)
	comms    *commsMsg      // non-nil: structured chat for a COMMS tab
	combat   *combatMsg     // non-nil: combat event with target health
	macros   []macroBinding // non-empty: server-pushed hotkey loadout to apply
	identity string         // non-empty: server-assigned login id to remember (silent)
	toChat   string         // non-empty: append to the chat panel
	toMain   string         // non-empty: append to the main view
}

// chat / communication lines from the current command set: say, shout, and the
// echoes of your own speech. Kept deliberately swappable: if the server starts
// emitting real CHAT| tags this whole heuristic can be replaced by a tag check.
var chatLine = regexp.MustCompile(`(?i)^(you (say|shout|tell|whisper)|.+ (says|shouts|tells you|whispers))`)

// exitsLine matches the trailing "Exits: [north, east]" line of a room block.
var exitsLine = regexp.MustCompile(`(?m)^Exits:\s*\[([^\]]*)\]`)

// hereLine matches a room-occupant line, e.g. "Gandalf is here." or
// "a cookie x3 is here." capturing the entity's display name.
var hereLine = regexp.MustCompile(`^(.+?)(?: x\d+)? is here\.$`)

// joinRe / leaveRe match the server's player join/leave notices (optionally
// STATUS|-tagged) and capture the player name.
var (
	joinRe  = regexp.MustCompile(`^(?:STATUS\|)?(.+?) has joined the game\.$`)
	leaveRe = regexp.MustCompile(`^(?:STATUS\|)?(.+?) has left the game\.$`)
)

// parseWhoNames extracts player names from a rendered `who` table. It returns
// nil for any other chunk, so it is safe to call on every message.
func parseWhoNames(raw string) []string {
	if !strings.Contains(raw, "Player") || !strings.Contains(raw, "Online Since") {
		return nil
	}
	var names []string
	for _, line := range strings.Split(raw, "\n") {
		cells := strings.FieldsFunc(line, func(r rune) bool { return r == '│' || r == '|' })
		if len(cells) == 0 {
			continue
		}
		name := strings.TrimSpace(cells[0])
		if name == "" || name == "Player" || !hasAlnum(name) {
			continue // header, border, or separator row
		}
		names = append(names, name)
	}
	return names
}

func hasAlnum(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return true
		}
	}
	return false
}

// classify routes a single server chunk into UI panels.
func classify(raw string) routed {
	if util.IsStateMessage(raw) {
		if s, ok := parseStatus(raw); ok {
			return routed{status: &s}
		}
		return routed{}
	}

	// Tag-aware routing for the prefixes the server emits in practice: STATUS|
	// (notifications), DMG| and DMG|DEATH| (combat), CHAT|, SYS|. Any other
	// uppercase TAG| is stripped to avoid leaking a raw prefix into the output.
	if tag, rest, ok := cutTag(raw); ok {
		switch tag {
		case "EVENT":
			return parseEvent(rest)
		case "IDENTITY":
			return routed{identity: rest}
		case "CHAT":
			return routed{toChat: rest}
		case "STATUS":
			return routed{toMain: noticeStyle.Render(rest)}
		case "DMG":
			return routed{toMain: combatStyle.Render(stripLeadingTag(rest))}
		default: // SYS and anything else: just show the text
			return routed{toMain: rest}
		}
	}

	text := util.StripTag(raw)

	if m := exitsLine.FindStringSubmatch(text); m != nil {
		return routed{room: parseRoom(text, m[1]), toMain: text}
	}

	if chatLine.MatchString(firstLine(text)) {
		return routed{toChat: text}
	}

	return routed{toMain: text}
}

// tagPrefix matches a leading uppercase protocol tag like "DMG|" or "STATUS|".
var tagPrefix = regexp.MustCompile(`^([A-Z][A-Z]+)\|`)

// cutTag splits a leading uppercase protocol tag from the rest of the message.
// Room titles (no "|") and prose never match.
func cutTag(raw string) (tag, rest string, ok bool) {
	m := tagPrefix.FindStringSubmatch(raw)
	if m == nil {
		return "", "", false
	}
	return m[1], raw[len(m[0]):], true
}

// parseEvent decodes an EVENT|{json} structured push. Unknown event types are
// ignored (silent), so the server can add packages without breaking the client.
func parseEvent(payload string) routed {
	var env struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(payload), &env) != nil {
		return routed{}
	}
	switch env.Type {
	case "room.contents":
		var c roomContents
		if json.Unmarshal([]byte(payload), &c) == nil {
			return routed{contents: &c}
		}
	case "room.info":
		var v struct {
			Name   string   `json:"name"`
			Exits  []string `json:"exits"`
			Region string   `json:"region"`
		}
		if json.Unmarshal([]byte(payload), &v) == nil {
			return routed{info: &roomInfo{Title: v.Name, Exits: v.Exits}}
		}
	case "comms":
		var v commsMsg
		if json.Unmarshal([]byte(payload), &v) == nil {
			return routed{comms: &v}
		}
	case "combat":
		var v combatMsg
		if json.Unmarshal([]byte(payload), &v) == nil {
			return routed{combat: &v}
		}
	case "macros":
		var v struct {
			Set []macroBinding `json:"set"`
		}
		if json.Unmarshal([]byte(payload), &v) == nil && len(v.Set) > 0 {
			return routed{macros: v.Set}
		}
	case "char.vitals":
		var v struct {
			HP      int            `json:"hp"`
			MaxHP   int            `json:"max_hp"`
			EP      int            `json:"ep"`
			MaxEP   int            `json:"max_ep"`
			Level   int            `json:"level"`
			XP      int            `json:"xp"`
			ReqXP   int            `json:"req_xp"`
			Area    string         `json:"area"`
			Gold    int            `json:"gold"`
			Effects []string       `json:"effects"`
			Fx      []struct {
				Name      string `json:"name"`
				Kind      string `json:"kind"`
				Remaining int    `json:"remaining"`
				Magnitude int    `json:"magnitude"`
			} `json:"fx"`
			Stats map[string]int `json:"stats"`
			Race  string         `json:"race"`
		}
		if json.Unmarshal([]byte(payload), &v) == nil {
			var fx []fxEffect
			for _, f := range v.Fx {
				fx = append(fx, fxEffect{Name: f.Name, Kind: f.Kind, Remaining: f.Remaining, Magnitude: f.Magnitude})
			}
			s := statusInfo{
				HP: v.HP, MaxHP: v.MaxHP, EP: v.EP, MaxEP: v.MaxEP,
				Level: v.Level, XP: v.XP, ReqXP: v.ReqXP,
				Area: v.Area, Gold: v.Gold, Effects: v.Effects, Fx: fx, Stats: v.Stats, Race: v.Race,
				HasHP: true, HasXP: true, HasEP: v.MaxEP > 0,
			}
			return routed{status: &s}
		}
	}
	return routed{}
}

// stripLeadingTag removes one more "WORD|" status segment, e.g. the DEATH in
// "DMG|DEATH|...".
func stripLeadingTag(rest string) string {
	if m := tagPrefix.FindString(rest); m != "" {
		return rest[len(m):]
	}
	return rest
}

// parseStatus decodes "STATE|HP:cur/max|LEVEL:n|XP:cur/req|AREA:..|EFFECTS:..".
func parseStatus(raw string) (statusInfo, bool) {
	if !strings.HasPrefix(raw, "STATE|") {
		return statusInfo{}, false
	}
	var s statusInfo
	for _, seg := range strings.Split(raw, "|")[1:] {
		key, val, ok := strings.Cut(seg, ":")
		if !ok {
			continue
		}
		switch key {
		case "HP":
			if cur, max, ok := cutInts(val, "/"); ok {
				s.HP, s.MaxHP, s.HasHP = cur, max, true
			}
		case "LEVEL":
			s.Level, _ = strconv.Atoi(val)
		case "XP":
			if cur, req, ok := cutInts(val, "/"); ok {
				s.XP, s.ReqXP, s.HasXP = cur, req, true
			}
		case "AREA":
			// The server packs a truncated room description (title + prose,
			// newlines and all) into AREA. Keep only the first line so the
			// single-line status bar stays intact.
			s.Area = firstLine(val)
		case "EFFECTS":
			for _, e := range strings.Split(val, ",") {
				if name, _, ok := strings.Cut(e, ":"); ok && name != "" {
					s.Effects = append(s.Effects, name)
				}
			}
		}
	}
	return s, true
}

// parseRoom extracts the room title (first non-empty line), exit list, and the
// list of entities present ("X is here." lines).
func parseRoom(text, exitsCSV string) *roomInfo {
	r := &roomInfo{Title: firstLine(text)}
	for _, e := range strings.Split(exitsCSV, ",") {
		if e = strings.TrimSpace(e); e != "" {
			r.Exits = append(r.Exits, e)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if m := hereLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			r.Occupants = append(r.Occupants, strings.TrimSpace(m[1]))
		}
	}
	return r
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

func cutInts(s, sep string) (a, b int, ok bool) {
	x, y, found := strings.Cut(s, sep)
	if !found {
		return 0, 0, false
	}
	a, err1 := strconv.Atoi(strings.TrimSpace(x))
	b, err2 := strconv.Atoi(strings.TrimSpace(y))
	return a, b, err1 == nil && err2 == nil
}
