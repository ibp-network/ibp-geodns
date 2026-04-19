package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDnsApiRouterRejectsUnknownMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/dns", strings.NewReader(`{"method":"bogus","parameters":{}}`))
	rec := httptest.NewRecorder()

	dnsApiRouter(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got := resp["result"]; got != "Invalid Request" {
		t.Fatalf("expected invalid request result, got %#v", got)
	}
}

func TestDnsApiRouterRejectsInvalidMemberEventsTime(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/dns", strings.NewReader(`{"method":"getMemberEvents","parameters":{"memberName":"member-a","startTime":"bad-time"}}`))
	rec := httptest.NewRecorder()

	dnsApiRouter(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got := resp["result"]; got != "invalid startTime" {
		t.Fatalf("expected invalid startTime result, got %#v", got)
	}
}
