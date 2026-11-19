package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rantauplus/backend/internal/modules/assistant"
	"rantauplus/backend/internal/modules/auth"
	"rantauplus/backend/internal/modules/finance"
)

type demoChatClient struct{}

func (demoChatClient) Chat(_ context.Context, input assistant.Request) (assistant.Answer, error) {
	if input.Skill != "finance" {
		return assistant.Answer{}, assistant.ErrUnknownSkill
	}
	return assistant.Answer{Skill: input.Skill, Mode: "retrieval", Answer: input.Message}, nil
}

func TestPersonalRoutesRequireSession(t *testing.T) {
	router := New(false, nil, nil, &auth.Service{}, &finance.SetupService{})
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/finance/setup"},
		{http.MethodPost, "/finance/setup/preview"},
		{http.MethodPut, "/finance/setup"},
		{http.MethodGet, "/finance/summary"},
		{http.MethodPost, "/assistant/chat"},
	} {
		req := httptest.NewRequest(route.method, "/api/v1"+route.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s %s: status %d, cache %q", route.method, route.path, rec.Code, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestDemoRoutes(t *testing.T) {
	const financePath = "/api/v1/demo/finance/summary"
	const assistantPath = "/api/v1/demo/assistant/chat"

	service := assistant.NewService(demoChatClient{})
	disabled := New(false, service, nil, nil, nil)
	financeOnly := New(true, nil, nil, nil, nil)
	fullDemo := New(true, service, nil, nil, nil)

	cases := []struct {
		name    string
		handler http.Handler
		method  string
		path    string
		body    string
		status  int
		match   string
	}{
		{"finance hidden without demo mode", disabled, http.MethodGet, financePath, "", http.StatusNotFound, ""},
		{"assistant hidden without demo mode", disabled, http.MethodPost, assistantPath, "", http.StatusNotFound, ""},
		{"assistant hidden without client", financeOnly, http.MethodPost, assistantPath, "", http.StatusNotFound, ""},
		{"finance response", financeOnly, http.MethodGet, financePath, "", http.StatusOK, `"safe_to_spend_daily":65000`},
		{"assistant invalid JSON", fullDemo, http.MethodPost, assistantPath, "{", http.StatusBadRequest, "INVALID_JSON"},
		{"assistant blank message", fullDemo, http.MethodPost, assistantPath, `{"message":"   "}`, http.StatusBadRequest, "INVALID_MESSAGE"},
		{"assistant unknown skill", fullDemo, http.MethodPost, assistantPath, `{"skill":"other","message":"halo"}`, http.StatusBadRequest, "UNKNOWN_SKILL"},
		{"assistant trimmed message", fullDemo, http.MethodPost, assistantPath, `{"skill":"finance","message":"  halo  "}`, http.StatusOK, `"answer":"halo"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			tc.handler.ServeHTTP(rec, req)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.match) {
				t.Fatalf("got status %d and body %q, want status %d and body containing %q",
					rec.Code, rec.Body.String(), tc.status, tc.match)
			}
		})
	}
}
