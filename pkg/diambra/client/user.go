package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func (c *Client) User() (*UserResponse, error) {
	resp, err := c.Request("GET", "user", nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user: %s", resp.Status)
	}
	var u UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}
