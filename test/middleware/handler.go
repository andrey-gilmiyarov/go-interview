package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type  ShortenedURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var UserShortenedURLs = map[string][]ShortenedURL{
	"1": {
		{ShortURL: "http://short1.com", OriginalURL: "http://long1.com"},
		{ShortURL: "http://short2.com", OriginalURL: "http://long2.com"},
		},
		"2": {
		{ShortURL: "http://short3.com", OriginalURL: "http://long3.com"},
		{ShortURL: "http://short4.com", OriginalURL: "http://long4.com"},
		},
}

func GetUserURLsHandler(c *gin.Context) {
	cookie, err := c.Request.Cookie("user_id")
	if err != nil {
		fmt.Println("not found user_id")
	}

	urls, ok := UserShortenedURLs[cookie.Value]
	if !ok {
		urls = []ShortenedURL{}
	}

	c.JSON(http.StatusOK, urls)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for a cookie with the user's ID
		_, err := c.Request.Cookie("user_id")
		if err != nil {
			// No cookie or empty value, so generate a new one
			c.SetCookie("user_id", "1", 360, "", "", true, true)
		}

		c.Next()
	}
}