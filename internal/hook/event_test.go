package hook

import "testing"

func TestParseBothAgentShapes(t *testing.T) {
	for _, data := range []string{
		`{"hook_event_name":"UserPromptSubmit","prompt":"codex prompt","turn_id":"1"}`,
		`{"hook_event_name":"UserPromptSubmit","prompt":"claude prompt","session_id":"1","transcript_path":"/tmp/x"}`,
	} {
		e, err := Parse([]byte(data))
		if err != nil || e.Prompt == "" {
			t.Fatalf("Parse(%s) = %+v, %v", data, e, err)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse([]byte("{")); err == nil {
		t.Fatal("expected error")
	}
}
