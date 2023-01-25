package diambra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// FIXME: Replace this with oapi generated code

// Mode Enum
type Mode string

const (
	ModeAIvsCOM Mode = "AIvsCOM"

	API = "https://diambra.ai/api/v1alpha1"
)

type Manifest struct {
	Image      string            `yaml:"image" json:"image"`
	Mode       Mode              `yaml:"mode" json:"mode"`
	Difficulty string            `yaml:"difficulty,omitempty" json:"difficulty,omitempty"`
	Command    []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Env        map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Sources    map[string]string `yaml:"sources,omitempty" json:"sources,omitempty"`
}

type Submission struct {
	Manifest Manifest          `yaml:"manifest" json:"manifest"`
	Secrets  map[string]string `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

type submitResponse struct {
	Submission
	ID int `json:"id"`
}

func readCredentials(homedir string) (string, error) {
	creds := os.Getenv("DIAMBRA_TOKEN")
	if creds != "" {
		return creds, nil
	}
	credFile := filepath.Join(homedir, ".diambra", "credentials")
	b, err := os.ReadFile(filepath.Join(credFile))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Submit(logger log.Logger, homedir string, submission *Submission) (int, error) {
	apiURL := os.Getenv("DIAMBRA_API_URL")
	if apiURL == "" {
		apiURL = API
	}
	logger = log.With(logger, "api", apiURL)

	data, err := json.Marshal(submission)
	if err != nil {
		return 0, err
	}
	creds, err := readCredentials(homedir)
	if err != nil {
		return 0, err
	}
	level.Debug(logger).Log("msg", "Submitting", "data", string(data))
	req, err := http.NewRequest(
		"POST",
		apiURL+"/submit",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+creds)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		errResp, err := io.ReadAll(resp.Body)
		if err != nil {
			errResp = []byte(fmt.Sprintf("failed to read error response: %s", err))
		}
		return 0, fmt.Errorf("failed to submit: %s: %s", resp.Status, errResp)
	}
	var s submitResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return 0, err
	}
	return s.ID, nil
}
