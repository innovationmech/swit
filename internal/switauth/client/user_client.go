package client

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/innovationmech/swit/internal/switserve/model"
)

type UserClient interface {
	GetUserByUsername(username string) (*model.User, error)
}

type userClient struct {
	baseUrl string
}

func NewUserClient(baseUrl string) UserClient {
	return &userClient{
		baseUrl: baseUrl,
	}
}

func (c *userClient) GetUserByUsername(username string) (*model.User, error) {
	resp, err := http.Get(c.baseUrl + "/v1/users/username/" + username)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get user by username")
	}

	var user model.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
