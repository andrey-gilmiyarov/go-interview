package checks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type userView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func Run(t *testing.T, handler func(map[string]string, string) http.Handler) {
	t.Helper()

	t.Run("authentication precedes method and path handling", func(t *testing.T) {
		h := handler(map[string]string{"u1": "Ann"}, "secret")
		for _, request := range []struct {
			method string
			path   string
			auth   string
		}{
			{http.MethodGet, "/users/u1", ""},
			{http.MethodPost, "/not-a-route", ""},
			{http.MethodGet, "/users/u1", "Bearer wrong"},
			{http.MethodGet, "/users/u1", "Basic secret"},
		} {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(request.method, request.path, nil)
			if request.auth != "" {
				req.Header.Set("Authorization", request.auth)
			}
			h.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s auth %q status = %d, want 401", request.method, request.path, request.auth, recorder.Code)
			}
		}
	})

	t.Run("route status and JSON behavior", func(t *testing.T) {
		h := handler(map[string]string{"u1": "Ann"}, "secret")
		tests := []struct {
			name       string
			method     string
			path       string
			wantStatus int
			wantAllow  string
		}{
			{"wrong method", http.MethodPost, "/users/u1", http.StatusMethodNotAllowed, http.MethodGet},
			{"missing route", http.MethodGet, "/users", http.StatusNotFound, ""},
			{"missing user", http.MethodGet, "/users/absent", http.StatusNotFound, ""},
			{"extra path segment", http.MethodGet, "/users/u1/extra", http.StatusNotFound, ""},
			{"known user", http.MethodGet, "/users/u1", http.StatusOK, ""},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				request := httptest.NewRequest(test.method, test.path, nil)
				request.Header.Set("Authorization", "Bearer secret")
				recorder := httptest.NewRecorder()
				h.ServeHTTP(recorder, request)
				if recorder.Code != test.wantStatus {
					t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
				}
				if got := recorder.Header().Get("Allow"); got != test.wantAllow {
					t.Fatalf("Allow = %q, want %q", got, test.wantAllow)
				}
				if test.wantStatus == http.StatusOK {
					if got := recorder.Header().Get("Content-Type"); got != "application/json" {
						t.Fatalf("Content-Type = %q, want application/json", got)
					}
					var got userView
					if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
						t.Fatalf("decode response: %v", err)
					}
					if got != (userView{ID: "u1", Name: "Ann"}) {
						t.Fatalf("response = %#v", got)
					}
				}
			})
		}
	})

	t.Run("constructor snapshots users and rejects empty secret", func(t *testing.T) {
		users := map[string]string{"u1": "before"}
		h := handler(users, "secret")
		users["u1"] = "after"
		request := httptest.NewRequest(http.MethodGet, "/users/u1", nil)
		request.Header.Set("Authorization", "Bearer secret")
		recorder := httptest.NewRecorder()
		h.ServeHTTP(recorder, request)
		var got userView
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Name != "before" {
			t.Fatalf("handler observed later map mutation: %#v", got)
		}

		defer func() {
			if recover() == nil {
				t.Fatal("Handler accepted an empty token")
			}
		}()
		handler(nil, "")
	})
}
