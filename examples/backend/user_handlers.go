package backend

import (
	"encoding/json"
	"net/http"
)

type User struct {
	ID        string
	FirstName string
	LastName  string
}

func QueryUserHandler(users map[string]User) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "userId is empty", http.StatusBadRequest)
			return
		}

		user, ok := users[userID]
		if !ok {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		payload, err := json.Marshal(user)
		if err != nil {
			http.Error(w, "can't provide a json. internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})
}

type ShortenedURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func CookieURLsHandler(userURLs map[string][]ShortenedURL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var urls []ShortenedURL
		if cookie, err := r.Cookie("user_id"); err == nil {
			urls = userURLs[cookie.Value]
		}
		if urls == nil {
			urls = []ShortenedURL{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(urls)
	})
}

func EnsureUserCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("user_id"); err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "user_id",
				Value:    "1",
				Path:     "/",
				MaxAge:   360,
				Secure:   true,
				HttpOnly: true,
			})
		}
		next.ServeHTTP(w, r)
	})
}

func UserURLsRoute(userURLs map[string][]ShortenedURL) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/user/urls", EnsureUserCookie(CookieURLsHandler(userURLs)))
	return mux
}
