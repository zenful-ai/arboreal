package arboreal

import (
	"testing"
)

func TestAllowedToolCall(t *testing.T) {
	cases := []struct {
		name  string
		tools []string
		call  string
		want  bool
	}{
		{"offered", []string{"alpha", "beta"}, "beta", true},
		{"not offered", []string{"alpha", "beta"}, "gamma", false},
		{"empty list allows everything", nil, "gamma", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allowedToolCall(tc.tools, tc.call); got != tc.want {
				t.Fatalf("allowedToolCall(%v, %q) = %v, want %v", tc.tools, tc.call, got, tc.want)
			}
		})
	}
}
