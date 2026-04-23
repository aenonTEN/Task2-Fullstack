package httpserver

import (
	"testing"
	"time"
)

func TestIntegration_CheckWithinDeduplicationWindow(t *testing.T) {
	tests := []struct {
		name      string
		lastCase  time.Time
		expectDup bool
	}{
		{"within 5 min - duplicate", time.Now().Add(-2 * time.Minute), true},
		{"at 5 min boundary - duplicate", time.Now().Add(-5*time.Minute + 30*time.Second), true},
		{"past 5 min - new case allowed", time.Now().Add(-6 * time.Minute), false},
		{"past 10 min - new case allowed", time.Now().Add(-10 * time.Minute), false},
		{"zero time - new case allowed", time.Time{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CheckWithinDeduplicationWindow(tc.lastCase)
			if result != tc.expectDup {
				t.Errorf("expected %v, got %v", tc.expectDup, result)
			}
		})
	}
}

func TestIntegration_ValidateFileSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expectOK bool
	}{
		{"zero - rejected", 0, false},
		{"negative - rejected", -1, false},
		{"under limit - OK", 1024, true},
		{"at limit - OK", MaxFileSize, true},
		{"over limit - rejected", MaxFileSize + 1, false},
		{"too large - rejected", 100 * 1024 * 1024, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateFileSize(tc.size)
			if result != tc.expectOK {
				t.Errorf("expected %v, got %v", tc.expectOK, result)
			}
		})
	}
}

func TestIntegration_ValidateChunkSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		expectOK bool
	}{
		{"zero - rejected", 0, false},
		{"negative - rejected", -1, false},
		{"under limit - OK", 1024, true},
		{"at limit - OK", MaxChunkSize, true},
		{"over limit - rejected", MaxChunkSize + 1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateChunkSize(tc.size)
			if result != tc.expectOK {
				t.Errorf("expected %v, got %v", tc.expectOK, result)
			}
		})
	}
}

func TestIntegration_CheckCaseDuplicate(t *testing.T) {
	now := time.Now()
	withinWindow := now.Add(-2 * time.Minute)
	outsideWindow := now.Add(-10 * time.Minute)

	tests := []struct {
		name      string
		cases     []CaseRecord
		candidate string
		caseType  string
		subject   string
		expectDup bool
	}{
		{
			"duplicate within window",
			[]CaseRecord{
				{CandidateID: "c1", CaseType: "enrollment", Subject: "Test", CreatedAt: withinWindow},
			},
			"c1", "enrollment", "Test", true,
		},
		{
			"no duplicate - outside window",
			[]CaseRecord{
				{CandidateID: "c1", CaseType: "enrollment", Subject: "Test", CreatedAt: outsideWindow},
			},
			"c1", "enrollment", "Test", false,
		},
		{
			"no duplicate - different candidate",
			[]CaseRecord{
				{CandidateID: "c1", CaseType: "enrollment", Subject: "Test", CreatedAt: withinWindow},
			},
			"c2", "enrollment", "Test", false,
		},
		{
			"no duplicate - different case type",
			[]CaseRecord{
				{CandidateID: "c1", CaseType: "enrollment", Subject: "Test", CreatedAt: withinWindow},
			},
			"c1", "complaint", "Test", false,
		},
		{
			"no duplicate - empty cases",
			[]CaseRecord{},
			"c1", "enrollment", "Test", false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CheckCaseDuplicate(tc.cases, tc.candidate, tc.caseType, tc.subject)
			if result != tc.expectDup {
				t.Errorf("expected %v, got %v", tc.expectDup, result)
			}
		})
	}
}

func TestIntegration_IsAllowedMimeType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expectOK bool
	}{
		{"application/pdf - allowed", "application/pdf", true},
		{"image/jpeg - allowed", "image/jpeg", true},
		{"image/png - allowed", "image/png", true},
		{"application/msword - allowed", "application/msword", true},
		{"application/octet-stream - not allowed", "application/octet-stream", false},
		{"text/html - not allowed", "text/html", false},
		{"empty - not allowed", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsAllowedMimeType(tc.mimeType)
			if result != tc.expectOK {
				t.Errorf("expected %v, got %v", tc.expectOK, result)
			}
		})
	}
}

func TestIntegration_IsWithinChunkAssemblyWindow(t *testing.T) {
	tests := []struct {
		name     string
		initTime time.Time
		maxAge   time.Duration
		expectOK bool
	}{
		{"recent init - active", time.Now().Add(-30 * time.Second), 10 * time.Minute, true},
		{"old init - expired", time.Now().Add(-15 * time.Minute), 10 * time.Minute, false},
		{"zero time - expired", time.Time{}, 10 * time.Minute, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsWithinChunkAssemblyWindow(tc.initTime, tc.maxAge)
			if result != tc.expectOK {
				t.Errorf("expected %v, got %v", tc.expectOK, result)
			}
		})
	}
}
