package resty

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/http"
	"sort"
	"time"
)

type (
	User struct {
		ID       int     `json:"id"`
		Name     string  `json:"name"`
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Address  Address `json:"address"`
		Phone    string  `json:"phone"`
		Website  string  `json:"website"`
		Company  Company `json:"company"`
	}
	Address struct {
		Street  string `json:"street"`
		Suite   string `json:"suite"`
		City    string `json:"city"`
		Zipcode string `json:"zipcode"`
		Geo     struct {
			Lat string `json:"lat"`
			Lng string `json:"lng"`
		} `json:"geo"`
	}
	Company struct {
		Name        string `json:"name"`
		CatchPhrase string `json:"catchPhrase"`
		Bs          string `json:"bs"`
	}
)

func main() {
	client := resty.New().
		SetBaseURL("https://jsonplaceholder.typicode.com/").
		SetRetryCount(5).
		SetRetryWaitTime(10 * time.Second)

	users, err := getUsers(client)
	if err != nil {
		panic(err)
	}

	sort.SliceStable(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})

	for _, u := range users {
		fmt.Println(u.Name)
	}
}

func getUsers(client *resty.Client) ([]User, error) {
	var users []User

	resp, err := client.R().SetResult(&users).Get("/users")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.New("can't get users. Status code <> 200")
	}

	return users, nil
}
