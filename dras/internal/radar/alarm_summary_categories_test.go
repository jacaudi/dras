package radar

import (
	"strings"
	"testing"
)

// This file gathers the remaining test categories for the alarmSummary
// feature (issue #128) that don't have a natural home in the existing
// per-function test files: Security, Retry, and Frame. The Unit,
// Functional, Performance, and Integration categories live alongside their
// subjects (radar_test.go, compare_test.go, benchmark_test.go,
// config_test.go / integration_test.go respectively).

// --- Security -------------------------------------------------------------
//
// The parser preserves NWS-supplied values verbatim rather than squashing
// them, so it must not panic, truncate, or otherwise transform adversarial
// input. alarmSummary is a text-only field that flows into a notification
// message; there is no interpolation into a query/template beyond fmt, so
// the security contract here is "faithful, non-crashing pass-through".
func TestParseAlarmSummary_Security_VerbatimPassthrough(t *testing.T) {
	adversarial := []string{
		"Communication\n\rInjected: line", // CR/LF injection attempt
		strings.Repeat("A", 8192),         // oversized value
		"'; DROP TABLE alarms;--",         // SQL-ish payload (no DB, but prove pass-through)
		"{{.Secret}}",                     // Go-template-looking payload
		"No Alarms\x00Communication",      // embedded NUL
		"⚠️ 警報 alarme",                    // multibyte / emoji
	}

	for _, in := range adversarial {
		got := ParseAlarmSummary(in)
		if string(got) != in {
			t.Errorf("ParseAlarmSummary(%q) = %q, want verbatim pass-through", in, string(got))
		}
	}
}

// --- Retry ----------------------------------------------------------------
//
// N/A placeholder. This feature adds no new external call: alarmSummary is
// read from the *already-fetched* radarResponse inside Service.FetchData, so
// it rides on the single existing nws.RadarStation() request. Retry/backoff
// for that upstream fetch lives in the nws client and internal/httpretry and
// is unchanged by this feature. ParseAlarmSummary and CompareData are pure,
// in-memory, and deterministic — there is nothing to retry. This test
// documents that contract by asserting the parser is side-effect-free and
// idempotent across repeated calls.
func TestParseAlarmSummary_Retry_NoExternalCall(t *testing.T) {
	const in = "Communication"
	first := ParseAlarmSummary(in)
	for i := 0; i < 5; i++ {
		if got := ParseAlarmSummary(in); got != first {
			t.Fatalf("ParseAlarmSummary is not idempotent: call %d = %q, want %q", i, got, first)
		}
	}
}

// --- Frame ----------------------------------------------------------------
//
// Builds & runs: exercise the whole alarmSummary path end-to-end in one
// frame — parse two raw NWS strings into Data, run them through the real
// CompareData comparator with the toggle on, and assert the exact
// user-facing message. If the wiring (type, Data field, comparator block)
// is broken this fails to compile or fails the assertion.
func TestAlarmSummary_Frame_EndToEnd(t *testing.T) {
	oldData := &Data{
		Name:         "KATX",
		AlarmSummary: ParseAlarmSummary("No Alarms"),
	}
	newData := &Data{
		Name:         "KATX",
		AlarmSummary: ParseAlarmSummary("Communication"),
	}

	changed, message := CompareData(oldData, newData, AlertConfig{AlarmSummary: true})
	if !changed {
		t.Fatal("Frame: expected alarm-summary change to be detected end-to-end")
	}
	const want = "Alarm summary changed from No Alarms to Communication"
	if message != want {
		t.Errorf("Frame: message = %q, want %q", message, want)
	}
}
