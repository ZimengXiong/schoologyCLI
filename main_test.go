package main

import "testing"

func TestHasSubmittedRevision(t *testing.T) {
	tests := []struct {
		name      string
		revisions []revision
		want      bool
	}{
		{name: "none", want: false},
		{name: "draft only", revisions: []revision{{Draft: 1}}, want: false},
		{name: "submitted", revisions: []revision{{Draft: 0}}, want: true},
		{name: "draft and submitted", revisions: []revision{{Draft: 1}, {Draft: 0}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasSubmittedRevision(tt.revisions); got != tt.want {
				t.Fatalf("hasSubmittedRevision() = %v, want %v", got, tt.want)
			}
		})
	}
}
