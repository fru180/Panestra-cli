package launcher

import "testing"

func TestInteractive(t *testing.T) {
	tests := []struct {
		agent         string
		args          []string
		in, out, want bool
	}{
		{"codex", nil, true, true, true}, {"codex", []string{"--model", "x"}, true, true, true},
		{"codex", []string{"exec", "test"}, true, true, false}, {"codex", []string{"--help"}, true, true, false},
		{"claude", nil, true, true, true}, {"claude", []string{"--model", "x"}, true, true, true},
		{"claude", []string{"-p", "test"}, true, true, false}, {"claude", nil, false, true, false},
	}
	for _, tt := range tests {
		if got := Interactive(tt.agent, tt.args, tt.in, tt.out); got != tt.want {
			t.Errorf("Interactive(%s,%v)=%v", tt.agent, tt.args, got)
		}
	}
}
