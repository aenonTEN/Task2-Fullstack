package httpserver

import (
	"testing"
)

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5551234567", "***4567"},
		{"123", "***"},
		{"", "***"},
		{"  ", "***"},
		{"+12345678901", "***8901"},
		{"ABC", "***"},
	}

	for _, tt := range tests {
		result := maskPhone(tt.input)
		if result != tt.expected {
			t.Errorf("maskPhone(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCheckQualificationExpiry(t *testing.T) {
	tests := []struct {
		expiryDate string
		expected   bool
	}{
		{"", true},
		{"2099-01-01", false},
		{"2020-01-01", true},
	}

	for _, tt := range tests {
		result := checkQualificationExpiry(tt.expiryDate)
		if result != tt.expected {
			t.Errorf("checkQualificationExpiry(%q) = %v, want %v", tt.expiryDate, result, tt.expected)
		}
	}
}

func TestErrorResponse(t *testing.T) {
	resp := errorResponse{
		Code:    "test_code",
		Message: "Test message",
		TraceID: "test-trace",
	}

	if resp.Code != "test_code" {
		t.Errorf("expected code test_code, got %s", resp.Code)
	}
	if resp.Message != "Test message" {
		t.Errorf("expected message Test message, got %s", resp.Message)
	}
}

func TestDataScope(t *testing.T) {
	scope := dataScope{
		InstitutionID: "inst_test",
		DepartmentID:  "dept_test",
		TeamID:        "team_test",
	}

	if scope.InstitutionID != "inst_test" {
		t.Errorf("expected InstitutionID inst_test, got %s", scope.InstitutionID)
	}
	if scope.DepartmentID != "dept_test" {
		t.Errorf("expected DepartmentID dept_test, got %s", scope.DepartmentID)
	}
	if scope.TeamID != "team_test" {
		t.Errorf("expected TeamID team_test, got %s", scope.TeamID)
	}
}

func TestAuditRecord(t *testing.T) {
	record := auditRecord{
		ID:        "test-id",
		TraceID:   "trace-id",
		ActorID:   "actor-id",
		Action:    "CREATE",
		Entity:    "Candidate",
		EntityID:  "entity-id",
		Before:    nil,
		After:     map[string]string{"name": "test"},
		CreatedAt: "2026-04-21T00:00:00Z",
	}

	if record.Action != "CREATE" {
		t.Errorf("expected action CREATE, got %s", record.Action)
	}
	if record.Entity != "Candidate" {
		t.Errorf("expected entity Candidate, got %s", record.Entity)
	}
}

func TestCandidateMergeResult(t *testing.T) {
	result := candidateMergeResult{
		IsNew:        true,
		CandidateID:  "cand-id-123",
		MergedFromID: "",
		Conflicts:    []string{},
	}

	if !result.IsNew {
		t.Errorf("expected IsNew to be true")
	}
	if result.CandidateID != "cand-id-123" {
		t.Errorf("expected CandidateID cand-id-123, got %s", result.CandidateID)
	}
}

func TestMatchResult(t *testing.T) {
	result := matchResult{
		CandidateID: "cand-123",
		Score:       100,
		Reasons:     []string{"Skill match", "Experience match"},
	}

	if result.Score != 100 {
		t.Errorf("expected score 100, got %d", result.Score)
	}
	if len(result.Reasons) != 2 {
		t.Errorf("expected 2 reasons, got %d", len(result.Reasons))
	}
}

func TestRestrictionCheckResponse(t *testing.T) {
	resp := restrictionCheckResponse{
		Allowed:        false,
		Reason:         "Restriction active",
		NextEligibleAt: "2026-04-28T16:00:00Z",
	}

	if resp.Allowed {
		t.Errorf("expected Allowed to be false")
	}
	if resp.Reason != "Restriction active" {
		t.Errorf("expected reason 'Restriction active', got %s", resp.Reason)
	}
	if resp.NextEligibleAt == "" {
		t.Errorf("expected NextEligibleAt to be set")
	}
}

func TestPositionItem(t *testing.T) {
	item := positionItem{
		ID:          "pos-123",
		Title:       "Software Engineer",
		Department:  "Engineering",
		Description: "Backend developer",
		Status:      "open",
		CreatedBy:   "user-123",
		CreatedAt:   "2026-04-21T00:00:00Z",
	}

	if item.Status != "open" {
		t.Errorf("expected status open, got %s", item.Status)
	}
	if item.Department != "Engineering" {
		t.Errorf("expected department Engineering, got %s", item.Department)
	}
}

func TestCaseItem(t *testing.T) {
	item := caseItem{
		ID:         "case-123",
		CaseNumber: "CASE-20260421-0001",
		CaseType:   "enrollment",
		Status:     "pending",
		Subject:    "Test case",
		CreatedBy:  "user-123",
		CreatedAt:  "2026-04-21T00:00:00Z",
	}

	if item.Status != "pending" {
		t.Errorf("expected status pending, got %s", item.Status)
	}
	if item.CaseNumber != "CASE-20260421-0001" {
		t.Errorf("expected case number CASE-20260421-0001, got %s", item.CaseNumber)
	}
}

func TestQualificationItem(t *testing.T) {
	item := qualificationItem{
		ID:          "qual-123",
		CandidateID: "cand-123",
		Name:        "Driver License",
		IssuedDate:  "2020-01-01",
		ExpiryDate:  "2030-01-01",
		Status:      "active",
	}

	if item.Status != "active" {
		t.Errorf("expected status active, got %s", item.Status)
	}
	if item.Name != "Driver License" {
		t.Errorf("expected name Driver License, got %s", item.Name)
	}
}

func TestAttachmentItem(t *testing.T) {
	item := attachmentItem{
		ID:         "att-123",
		CaseID:     "case-123",
		FileName:   "document.pdf",
		FileSize:   1024000,
		MimeType:   "application/pdf",
		SHA256:     "abc123",
		UploadedBy: "user-123",
		CreatedAt:  "2026-04-21T00:00:00Z",
	}

	if item.MimeType != "application/pdf" {
		t.Errorf("expected mime type application/pdf, got %s", item.MimeType)
	}
	if item.FileSize != 1024000 {
		t.Errorf("expected file size 1024000, got %d", item.FileSize)
	}
}
