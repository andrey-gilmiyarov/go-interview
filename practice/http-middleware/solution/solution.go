package solution

import (
	"encoding/json"
	"net/http"
	"strings"
)

func Handler(users map[string]string, token string) http.Handler {
	if token == "" {
		panic("http-middleware: token must not be empty")
	}

	snapshot := make(map[string]string, len(users))
	for id, name := range users {
		snapshot[id] = name
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		const prefix = "/users/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, prefix)
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		name, ok := snapshot[id]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{ID: id, Name: name})
	})
}
