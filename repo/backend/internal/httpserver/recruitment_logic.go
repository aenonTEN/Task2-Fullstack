package httpserver

import (
	"strings"
	"time"
)

const (
	ScoreSkillMax     = 50
	ScoreExpThreshold = 3
	ScoreExpMax       = 30
	ScoreEduMax       = 20
)

func CalculateMatchScore(experienceYears int, education string, skills []string, targetSkill, targetEducation string) (int, []string) {
	score := 0
	reasons := make([]string, 0, 4)

	if targetSkill != "" {
		found := false
		for _, s := range skills {
			if strings.ToLower(strings.TrimSpace(s)) == strings.ToLower(targetSkill) {
				found = true
				break
			}
		}
		if found {
			score += ScoreSkillMax
			reasons = append(reasons, "Requested skill matched (+50)")
		} else {
			reasons = append(reasons, "Requested skill not matched (+0)")
		}
	}

	if experienceYears >= ScoreExpThreshold {
		score += ScoreExpMax
		reasons = append(reasons, "Experience threshold met (+30)")
	} else {
		reasons = append(reasons, "Experience below threshold (+0)")
	}

	if targetEducation != "" && strings.Contains(strings.ToLower(education), strings.ToLower(targetEducation)) {
		score += ScoreEduMax
		reasons = append(reasons, "Education requirement matched (+20)")
	} else if targetEducation != "" {
		reasons = append(reasons, "Education requirement not matched (+0)")
	}

	return score, reasons
}

func IsWithinRestrictionWindow(actionTime time.Time, windowHours int) bool {
	if actionTime.IsZero() {
		return false
	}
	windowEnd := actionTime.Add(time.Duration(windowHours) * time.Hour)
	return time.Now().UTC().Before(windowEnd)
}

type CandidateStore interface {
	GetByPhone(ctx interface{}, phone string) (*Candidate, error)
	GetByIDNumber(ctx interface{}, idNumber string) (*Candidate, error)
	Create(ctx interface{}, c *Candidate) error
}

type Candidate struct {
	ID              string
	Name            string
	Phone           string
	PhoneMasked     string
	IDNumber        string
	Education       string
	ExperienceYears int
	Skills          []string
	InstitutionID   string
}

var _ CandidateStore = nil

type inMemoryStore map[string]*Candidate

func (m inMemoryStore) GetByPhone(_ interface{}, phone string) (*Candidate, error) {
	if c, ok := m[phone]; ok {
		return c, nil
	}
	return nil, nil
}

func (m inMemoryStore) GetByIDNumber(_ interface{}, idNumber string) (*Candidate, error) {
	for _, c := range m {
		if c.IDNumber == idNumber {
			return c, nil
		}
	}
	return nil, nil
}

func (m inMemoryStore) Create(_ interface{}, c *Candidate) error {
	m[c.Phone] = c
	return nil
}
