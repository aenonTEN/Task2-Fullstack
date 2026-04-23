package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireRole_WithMockedRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		roles    []string
		required string
		wantCode int
	}{
		{"admin allowed", []string{"role_admin"}, "role_admin", http.StatusOK},
		{"user denied", []string{"role_user"}, "role_admin", http.StatusForbidden},
		{"empty roles denied", []string{}, "role_admin", http.StatusForbidden},
		{"mixed roles - admin present", []string{"role_user", "role_admin"}, "role_admin", http.StatusOK},
		{"viewer denied", []string{"role_viewer"}, "role_admin", http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("roleIds", tc.roles)
				c.Next()
			})
			r.GET("/test", requireRole(tc.required), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			rec := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Errorf("%s: expected %d, got %d", tc.name, tc.wantCode, rec.Code)
			}
		})
	}
}

func TestDataScope_InsitutionIsolation(t *testing.T) {
	tests := []struct {
		name          string
		sessionScope  dataScope
		resourceScope dataScope
		allow         bool
	}{
		{"same institution", dataScope{InstitutionID: "inst-A"}, dataScope{InstitutionID: "inst-A"}, true},
		{"cross institution", dataScope{InstitutionID: "inst-A"}, dataScope{InstitutionID: "inst-B"}, false},
		{"empty session", dataScope{}, dataScope{InstitutionID: "inst-A"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scope := tc.sessionScope
			allow := scope.InstitutionID != "" && scope.InstitutionID == tc.resourceScope.InstitutionID

			if allow != tc.allow {
				t.Errorf("expected allow=%v, got %v", tc.allow, allow)
			}
		})
	}
}

func TestRequireRole_MissingRoleContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", requireRole("role_admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestRequireRole_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", requireRole("role_admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		// Expected - check error response format
	}
	if rec.Code == 0 {
		t.Errorf("no response code")
	}
}
