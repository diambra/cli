package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/log/level"
)

type TokenResponse struct {
	Token string `json:"token"`
}

type TokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Client) Token(username, password string) (string, error) {
	data, err := json.Marshal(&TokenRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", err
	}
	level.Debug(c.logger).Log("msg", "Submitting", "data", string(data))
	resp, err := c.Request("POST", "token", bytes.NewBuffer(data), false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %s", resp.Status)
	}
	var t TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", err
	}
	return t.Token, nil
}
