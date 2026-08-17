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

func TestSubmissionState(t *testing.T) {
	tests := []struct {
		name       string
		revisions  []revision
		wantStatus string
		wantID     int64
	}{
		{name: "none", wantStatus: "not_submitted"},
		{name: "draft", revisions: []revision{{RevisionID: 1, Draft: 1}}, wantStatus: "draft"},
		{name: "latest submitted", revisions: []revision{{RevisionID: 1, Created: 10}, {RevisionID: 2, Created: 20}, {RevisionID: 3, Created: 30, Draft: 1}}, wantStatus: "submitted", wantID: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, latest := submissionState(tt.revisions)
			if status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", status, tt.wantStatus)
			}
			if tt.wantID == 0 && latest != nil {
				t.Fatalf("latest = %#v, want nil", latest)
			}
			if tt.wantID != 0 && (latest == nil || latest.RevisionID != tt.wantID) {
				t.Fatalf("latest = %#v, want revision %d", latest, tt.wantID)
			}
		})
	}
}

func TestSectionMatches(t *testing.T) {
	s := section{CourseTitle: "Existentialism and the Absurd", CourseCode: "ABS", SectionTitle: "Period 2"}
	for _, query := range []string{"", "existentialism", "ABS", "period 2"} {
		if !sectionMatches(s, query) {
			t.Errorf("sectionMatches(%q) = false", query)
		}
	}
	if sectionMatches(s, "physics") {
		t.Error("unexpected match")
	}
}

func TestIsExternalSubmission(t *testing.T) {
	if !isExternalSubmission(assignment{Description: "Upload through Turnitin.com"}) {
		t.Error("Turnitin assignment not detected")
	}
	if isExternalSubmission(assignment{Description: "Upload to the Schoology dropbox"}) {
		t.Error("Schoology assignment marked external")
	}
}
