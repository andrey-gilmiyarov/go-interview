package backend

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestValidateLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantErrors []error
	}{
		{"valid Unicode letters and exact spaces", "Код это Go", nil},
		{"punctuation is not a digit", "a! b. c?", nil},
		{"19 Unicode runes is allowed", strings.Repeat("я", 17) + "  ", nil},
		{"20 Unicode runes is too long", strings.Repeat("я", 18) + "  ", []error{ErrLineTooLong}},
		{"Unicode digit and wrong space count", "Тест ٣", []error{ErrContainsDigit, ErrRequiresTwoSpaces}},
		{"all independent rules join", strings.Repeat("x", 20) + "1", []error{ErrContainsDigit, ErrLineTooLong, ErrRequiresTwoSpaces}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateLine(test.line)
			if len(test.wantErrors) == 0 {
				if err != nil {
					t.Fatalf("ValidateLine(%q) = %#v, want nil", test.line, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateLine(%q) = nil, want validation errors", test.line)
			}
			for _, want := range test.wantErrors {
				if !errors.Is(err, want) {
					t.Errorf("errors.Is(error, %v) = false; error = %v", want, err)
				}
			}
			joined, ok := err.(interface{ Unwrap() []error })
			if !ok || len(joined.Unwrap()) != len(test.wantErrors) {
				t.Errorf("joined causes = %v, want %d", err, len(test.wantErrors))
			}
		})
	}
}

func TestLoggingLevelEmbeddingAndSlog(t *testing.T) {
	var legacyOutput bytes.Buffer
	legacy := NewEmbeddedLevelLogger(&legacyOutput, LogLevelError)
	legacy.Infoln("filtered")
	legacy.SetLogLevel(LogLevelWarning)
	legacy.Warnln("warning visible")
	legacy.Errorln("visible")
	legacy.Println("promoted method bypasses filter")
	if got := legacyOutput.String(); strings.Contains(got, "filtered") || !strings.Contains(got, "warning visible") || !strings.Contains(got, "visible") || !strings.Contains(got, "promoted method bypasses filter") {
		t.Fatalf("embedded logger output = %q", got)
	}

	var slogOutput bytes.Buffer
	modern := NewSlogLogger(&slogOutput, slog.LevelError)
	modern.Info("filtered")
	modern.Error("visible")
	if got := slogOutput.String(); strings.Contains(got, "filtered") || !strings.Contains(got, "level=ERROR") || !strings.Contains(got, "visible") {
		t.Fatalf("slog output = %q", got)
	}
}

type oneMethodSender struct {
	email   string
	message string
}

func (s *oneMethodSender) SendMessage(email, message string) error {
	s.email, s.message = email, message
	return nil
}

func TestSendWelcomeUsesNarrowClientInterface(t *testing.T) {
	var _ MessageSender = (*BigAPIClient)(nil)
	sender := &oneMethodSender{}
	if err := SendWelcome(sender, "ana@example.test"); err != nil {
		t.Fatalf("SendWelcome returned error: %v", err)
	}
	if sender.email != "ana@example.test" || sender.message != "Welcome" {
		t.Fatalf("SendMessage arguments = (%q, %q)", sender.email, sender.message)
	}
}

func TestQueryUserHandler(t *testing.T) {
	users := map[string]User{
		"u1": {ID: "u1", FirstName: "Misha", LastName: "Popov"},
	}
	handler := QueryUserHandler(users)
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"missing query value", "", http.StatusBadRequest},
		{"unknown user", "unknown", http.StatusNotFound},
		{"known user", "u1", http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/users?user_id="+test.query, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body %q", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if test.wantStatus == http.StatusOK {
				if got := recorder.Header().Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %q", got)
				}
				var got User
				if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
					t.Fatalf("decode user: %v", err)
				}
				if got != users["u1"] {
					t.Fatalf("user = %#v, want %#v", got, users["u1"])
				}
			}
		})
	}
}

func TestCookieHandlerAndAuthenticationOrdering(t *testing.T) {
	urls := map[string][]ShortenedURL{
		"1": {{ShortURL: "https://short.example/a", OriginalURL: "https://long.example/a"}},
	}
	handler := CookieURLsHandler(urls)

	t.Run("missing cookie is safe and returns an empty array", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/user/urls", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", recorder.Code)
		}
		if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
			t.Fatalf("body = %q, want []", got)
		}
	})

	t.Run("middleware is registered before the route and cookie applies next request", func(t *testing.T) {
		wrapped := UserURLsRoute(urls)
		first := httptest.NewRecorder()
		wrapped.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/user/urls", nil))
		cookies := first.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != "user_id" || cookies[0].Value != "1" || !cookies[0].HttpOnly || !cookies[0].Secure {
			t.Fatalf("Set-Cookie = %#v", cookies)
		}
		if got := strings.TrimSpace(first.Body.String()); got != "[]" {
			t.Fatalf("first response body = %q, want empty list before browser returns the cookie", got)
		}

		nextRequest := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		nextRequest.AddCookie(cookies[0])
		next := httptest.NewRecorder()
		wrapped.ServeHTTP(next, nextRequest)
		var got []ShortenedURL
		if err := json.Unmarshal(next.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode cookie response: %v", err)
		}
		if !reflect.DeepEqual(got, urls["1"]) {
			t.Fatalf("urls = %#v, want %#v", got, urls["1"])
		}
	})
}

func TestFetchUsersWithLocalHTTPServer(t *testing.T) {
	wantUsers := []APIUser{
		{
			ID:       2,
			Name:     "Zoë",
			Username: "zoe",
			Email:    "zoe@example.test",
			Address: APIAddress{
				Street:  "Main Street",
				Suite:   "Apt. 2",
				City:    "Tbilisi",
				Zipcode: "0100",
				Geo:     APIGeo{Lat: "41.7", Lng: "44.8"},
			},
			Phone:   "555-0102",
			Website: "zoe.example.test",
			Company: APICompany{
				Name:        "Example Co",
				CatchPhrase: "Learn together",
				Bs:          "build practice",
			},
		},
		{ID: 1, Name: "Аня"},
	}
	tests := []struct {
		name      string
		status    int
		body      string
		wantUsers []APIUser
		wantErr   bool
	}{
		{"status and complete JSON body", http.StatusOK, `[{"id":2,"name":"Zoë","username":"zoe","email":"zoe@example.test","address":{"street":"Main Street","suite":"Apt. 2","city":"Tbilisi","zipcode":"0100","geo":{"lat":"41.7","lng":"44.8"}},"phone":"555-0102","website":"zoe.example.test","company":{"name":"Example Co","catchPhrase":"Learn together","bs":"build practice"}},{"id":1,"name":"Аня"}]`, wantUsers, false},
		{"non-200 status", http.StatusServiceUnavailable, "temporarily unavailable", nil, true},
		{"invalid JSON body", http.StatusOK, "not-json", nil, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/users" {
					t.Errorf("request = %s %s, want GET /users", r.Method, r.URL.Path)
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()

			users, err := FetchUsers(server.Client(), server.URL+"/users")
			if (err != nil) != test.wantErr {
				t.Fatalf("FetchUsers error = %v, wantErr %v", err, test.wantErr)
			}
			if !reflect.DeepEqual(users, test.wantUsers) {
				t.Fatalf("users = %#v, want %#v", users, test.wantUsers)
			}
			if !test.wantErr {
				SortUsersByName(users)
				if users[0].Name != "Zoë" || users[1].Name != "Аня" {
					t.Fatalf("sorted user names = %#v", users)
				}
			}
		})
	}
}
