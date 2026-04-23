package httpserver

import (
	"time"
)

const (
	DedupeWindowMinutes = 5
	MaxChunkSize        = 5 * 1024 * 1024
	MaxFileSize         = 50 * 1024 * 1024
)

func CheckWithinDeduplicationWindow(lastCaseTime time.Time) bool {
	if lastCaseTime.IsZero() {
		return false
	}
	windowEnd := lastCaseTime.Add(DedupeWindowMinutes * time.Minute)
	return time.Now().UTC().Before(windowEnd) || time.Now().UTC().Equal(windowEnd)
}

func ValidateFileSize(size int64) bool {
	return size > 0 && size <= MaxFileSize
}

func ValidateChunkSize(size int) bool {
	return size > 0 && size <= MaxChunkSize
}

func IsWithinChunkAssemblyWindow(initTime time.Time, maxAge time.Duration) bool {
	if initTime.IsZero() {
		return false
	}
	return time.Now().UTC().Before(initTime.Add(maxAge))
}

type CaseRecord struct {
	ID            string
	CandidateID   string
	CaseType      string
	Subject       string
	CreatedAt     time.Time
	InstitutionID string
}

func CheckCaseDuplicate(cases []CaseRecord, candidateID, caseType, subject string) bool {
	windowStart := time.Now().UTC().Add(-DedupeWindowMinutes * time.Minute)
	for _, c := range cases {
		if c.CandidateID == candidateID &&
			c.CaseType == caseType &&
			c.Subject == subject &&
			c.CreatedAt.After(windowStart) {
			return true
		}
	}
	return false
}

func AllowedMimeTypes() map[string]bool {
	return map[string]bool{
		"application/pdf":    true,
		"image/jpeg":         true,
		"image/png":          true,
		"image/gif":          true,
		"application/msword": true,
	}
}

func IsAllowedMimeType(mimeType string) bool {
	return AllowedMimeTypes()[mimeType]
}
