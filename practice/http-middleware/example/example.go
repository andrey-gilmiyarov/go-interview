package example

import (
	"net/http"

	"github.com/andreygilmiyarov/go-interview/practice/http-middleware/starter"
)

func UserHandler() http.Handler {
	return starter.Handler(map[string]string{
		"u1": "Анна",
		"u2": "Михаил",
	}, "example-token")
}
