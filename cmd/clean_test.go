package cmd

import (
	"strings"
	"testing"
)

func TestConfirmAction(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "lowercase yes", in: "y\n", want: true},
		{name: "uppercase yes", in: "Y\n", want: true},
		{name: "no", in: "n\n", want: false},
		{name: "empty", in: "\n", want: false},
		{name: "missing input", in: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			if got := confirmAction("Continue?", strings.NewReader(tt.in), &out); got != tt.want {
				t.Fatalf("confirmAction() = %v, want %v", got, tt.want)
			}
			if !strings.Contains(out.String(), "Continue? [y/N]: ") {
				t.Fatalf("prompt = %q", out.String())
			}
		})
	}
}
