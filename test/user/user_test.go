package user

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserViewHandler(t *testing.T) {
	users := make(map[string]User)
	u1 := User{
		ID:        "u1",
		FirstName: "Misha",
		LastName:  "Popov",
	}
	u2 := User{
		ID:        "u2",
		FirstName: "Sasha",
		LastName:  "Popov",
	}
	users["u1"] = u1
	users["u2"] = u2

	type want struct {
		code        int
		response    User
		contentType string
	}
	tests := []struct {
		name   string
		userID string
		users  map[string]User
		want   want
	}{
		{
			name:   "tests #1",
			userID: "u1",
			users:  users,
			want: want{
				code: 200,
				response: User{
					ID:        "u1",
					FirstName: "Misha",
					LastName:  "Popov",
				},
				contentType: "application/json",
			},
		},
		{
			name:   "tests #2",
			userID: "u2",
			users:  users,
			want: want{
				code: 200,
				response: User{
					ID:        "u2",
					FirstName: "Sasha",
					LastName:  "Popov",
				},
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/users?user_id="+tt.userID, nil)

			w := httptest.NewRecorder()
			h := UserViewHandler(users)
			h(w, request)
			res := w.Result()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			var user User
			err = json.Unmarshal(resBody, &user)
			require.NoError(t, err)
			assert.Equal(t, tt.want.response, user)
		})
	}
}
