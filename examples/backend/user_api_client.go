package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

type APIUser struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	Username string     `json:"username"`
	Email    string     `json:"email"`
	Address  APIAddress `json:"address"`
	Phone    string     `json:"phone"`
	Website  string     `json:"website"`
	Company  APICompany `json:"company"`
}

type APIAddress struct {
	Street  string `json:"street"`
	Suite   string `json:"suite"`
	City    string `json:"city"`
	Zipcode string `json:"zipcode"`
	Geo     APIGeo `json:"geo"`
}

type APIGeo struct {
	Lat string `json:"lat"`
	Lng string `json:"lng"`
}

type APICompany struct {
	Name        string `json:"name"`
	CatchPhrase string `json:"catchPhrase"`
	Bs          string `json:"bs"`
}

func FetchUsers(client *http.Client, endpoint string) ([]APIUser, error) {
	if client == nil {
		return nil, fmt.Errorf("fetch users: HTTP client is nil")
	}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch users: create request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch users: send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch users: unexpected status %s", response.Status)
	}

	var users []APIUser
	if err := json.NewDecoder(response.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("fetch users: decode response: %w", err)
	}
	return users, nil
}

func SortUsersByName(users []APIUser) {
	sort.SliceStable(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})
}
