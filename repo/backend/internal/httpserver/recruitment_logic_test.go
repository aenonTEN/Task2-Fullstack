package httpserver

import (
	"testing"
	"time"
)

func TestCalculateMatchScore(t *testing.T) {
	tests := []struct {
		name          string
		exp           int
		edu           string
		skills        []string
		targetSkill   string
		targetEdu     string
		expScore      int
		expMinReasons int
	}{
		{"Full match", 5, "bachelor", []string{"python", "golang"}, "python", "bachelor", 100, 3},
		{"Skill only", 2, "high school", []string{"python"}, "python", "", 50, 1},
		{"Experience only", 5, "high school", []string{"ruby"}, "", "", 30, 1},
		{"No match", 1, "high school", []string{"javascript"}, "python", "bachelor", 0, 3},
		{"Education only", 1, "bachelor", []string{"ruby"}, "", "bachelor", 20, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, reasons := CalculateMatchScore(tc.exp, tc.edu, tc.skills, tc.targetSkill, tc.targetEdu)
			if score != tc.expScore {
				t.Errorf("expected score %d, got %d", tc.expScore, score)
			}
			if len(reasons) < tc.expMinReasons {
				t.Errorf("expected at least %d reasons, got %d", tc.expMinReasons, len(reasons))
			}
		})
	}
}

func TestIsWithinRestrictionWindow(t *testing.T) {
	tests := []struct {
		name       string
		lastAction time.Time
		windowHrs  int
		expect     bool
	}{
		{"Active - 24h remaining", time.Now().Add(-144 * time.Hour), 168, true},
		{"Active - 100h remaining", time.Now().Add(-68 * time.Hour), 168, true},
		{"Active - exactly 168h left", time.Now(), 168, true},
		{"Expired - past window", time.Now().Add(-200 * time.Hour), 168, false},
		{"Zero time", time.Time{}, 168, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsWithinRestrictionWindow(tc.lastAction, tc.windowHrs)
			if result != tc.expect {
				t.Errorf("expected %v, got %v", tc.expect, result)
			}
		})
	}
}

func TestCalculateMatchScore_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		exp         int
		edu         string
		skills      []string
		targetSkill string
		targetEdu   string
	}{
		{"Empty skills", 5, "bachelor", []string{}, "python", "bachelor"},
		{"Nil skills", 5, "bachelor", nil, "python", "bachelor"},
		{"Empty target", 5, "bachelor", []string{"python"}, "", ""},
		{"Case insensitive skill", 5, "Bachelor", []string{"PYTHON"}, "python", "bachelor"},
		{"Case insensitive education", 5, "BACHELOR", []string{"python"}, "python", "bachelor"},
		{"Zero experience", 0, "bachelor", []string{"python"}, "python", "bachelor"},
		{"Negative experience", -1, "bachelor", []string{"python"}, "python", "bachelor"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, _ := CalculateMatchScore(tc.exp, tc.edu, tc.skills, tc.targetSkill, tc.targetEdu)
			if score < 0 || score > 100 {
				t.Errorf("score %d out of valid range", score)
			}
		})
	}
}

type mockStore map[string]*Candidate

func (m mockStore) GetByPhone(_ interface{}, phone string) (*Candidate, error) {
	if c, ok := m[phone]; ok {
		return c, nil
	}
	return nil, nil
}

func (m mockStore) GetByIDNumber(_ interface{}, idNumber string) (*Candidate, error) {
	for _, c := range m {
		if c.IDNumber == idNumber {
			return c, nil
		}
	}
	return nil, nil
}

func (m mockStore) Create(_ interface{}, c *Candidate) error {
	m[c.Phone] = c
	return nil
}

func TestCandidateStore_InMemory(t *testing.T) {
	store := make(mockStore)
	store["555-123-4567"] = &Candidate{
		ID:       "c1",
		Name:     "John Doe",
		Phone:    "555-123-4567",
		IDNumber: "ABC123",
	}

	t.Run("GetByPhone finds existing", func(t *testing.T) {
		c, err := store.GetByPhone(nil, "555-123-4567")
		if err != nil || c == nil || c.Name != "John Doe" {
			t.Errorf("expected to find John Doe")
		}
	})

	t.Run("GetByPhone returns nil for non-existent", func(t *testing.T) {
		c, err := store.GetByPhone(nil, "555-999-9999")
		if err != nil || c != nil {
			t.Errorf("expected nil for non-existent")
		}
	})

	t.Run("GetByIDNumber finds existing", func(t *testing.T) {
		c, err := store.GetByIDNumber(nil, "ABC123")
		if err != nil || c == nil || c.ID != "c1" {
			t.Errorf("expected to find by ID number")
		}
	})

	t.Run("Create adds new candidate", func(t *testing.T) {
		newC := &Candidate{ID: "c2", Name: "Jane", Phone: "555-999-9999"}
		err := store.Create(nil, newC)
		if err != nil {
			t.Errorf("create failed: %v", err)
		}
		c, _ := store.GetByPhone(nil, "555-999-9999")
		if c == nil || c.Name != "Jane" {
			t.Errorf("create did not work")
		}
	})
}
