package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Error struct {
	error
}

type ErrForbidden Error

// FIXME: Replace this with oapi generated code

type Client struct {
	logger   log.Logger
	credPath string
}

func readCredentials(credPath string) (string, error) {
	creds := os.Getenv("DIAMBRA_TOKEN")
	if creds != "" {
		return creds, nil
	}
	b, err := os.ReadFile(credPath)
	if err != nil {
		return "", fmt.Errorf("can't read credentials file %s: %w", credPath, err)
	}
	return strings.TrimSpace(string(b)), nil
}

func NewClient(logger log.Logger, credPath string) (*Client, error) {
	return &Client{
		logger:   logger,
		credPath: credPath,
	}, nil
}

func (c *Client) token() (string, error) {
	return readCredentials(c.credPath)
}
func (c *Client) Request(method string, path string, body io.Reader, authenticated bool) (*http.Response, error) {
	apiURL := os.Getenv("DIAMBRA_API_URL")
	if apiURL == "" {
		apiURL = API
	}
	surl, err := url.JoinPath(apiURL, path)
	if err != nil {
		return nil, err
	}
	level.Debug(c.logger).Log("msg", "Request", "method", method, "url", surl, "authenticated", authenticated)

	req, err := http.NewRequest(
		method,
		surl,
		body,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authenticated {
		token, err := c.token()
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Token "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// read the body for error

		rb, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, &ErrForbidden{fmt.Errorf("unauthorized; couldn't read body: %w", err)}
		}
		_ = resp.Body.Close()
		return nil, ErrForbidden{errors.New(string(rb))}
	}
	return resp, nil
}
