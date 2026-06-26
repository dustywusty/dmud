package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func vitals(hp, maxHP int) chunkMsg {
	return chunkMsg{raw: `EVENT|{"type":"char.vitals","hp":` + itoa(hp) + `,"max_hp":` + itoa(maxHP) +
		`,"level":1,"xp":0,"req_xp":100,"area":"X"}`}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func TestLowHealthAlarm(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m.alarmPct = 20
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Healthy: no alarm.
	m = drive(m, vitals(80, 100))
	if m.hpAlarmed {
		t.Error("should not alarm at 80% HP")
	}

	// Drops below threshold: alarm latches and warns.
	m = drive(m, vitals(15, 100))
	if !m.hpAlarmed {
		t.Error("should alarm at 15% HP")
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "LOW HEALTH") {
		t.Error("output should show the LOW HEALTH warning")
	}

	// Recovers above threshold: alarm rearms.
	m = drive(m, vitals(90, 100))
	if m.hpAlarmed {
		t.Error("alarm should rearm after HP recovers")
	}
}

func TestAlarmDisabled(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m.alarmPct = -1 // off
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, vitals(1, 100))
	if m.hpAlarmed {
		t.Error("a disabled alarm must never latch")
	}
}
