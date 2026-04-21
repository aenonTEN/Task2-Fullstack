package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessEndpoint(t *testing.T) {
	router := NewRouterWithDeps(&RouterDeps{DB: nil})
	if router == nil {
		t.Fatal("expected non-nil router")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestMeRequiresAuth(t *testing.T) {
	router := NewRouterWithDeps(&RouterDeps{DB: nil})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestLoginValidation(t *testing.T) {
	router := NewRouterWithDeps(&RouterDeps{DB: nil})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewBufferString(`{"username":"","password":""}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes := httptest.NewRecorder()
	router.ServeHTTP(loginRes, loginReq)
	if loginRes.Code == http.StatusOK {
		t.Fatal("expected non-200 status for empty credentials")
	}
}
