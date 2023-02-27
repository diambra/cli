package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-kit/log/level"
	"gopkg.in/yaml.v2"
)

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
	Args       []string          `yaml:"args,omitempty" json:"args,omitempty"`
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

func (c *Client) Submit(submission *Submission) (int, error) {
	data, err := json.Marshal(submission)
	if err != nil {
		return 0, err
	}
	level.Debug(c.logger).Log("msg", "Submitting", "data", string(data))
	resp, err := c.Request("POST", "submit", bytes.NewBuffer(data), true)
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

func (c *Client) Submission(id int) (*Submission, error) {
	resp, err := c.Request("GET", fmt.Sprintf("submissions/%d", id), nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errResp, err := io.ReadAll(resp.Body)
		if err != nil {
			errResp = []byte(fmt.Sprintf("failed to read error response: %s", err))
		}
		return nil, fmt.Errorf("failed to get submission: %s: %s", resp.Status, errResp)
	}
	var s Submission
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func ManifestFromPath(path string) (*Manifest, error) {
	manifest := &Manifest{}
	if path == "-" {
		return manifest, yaml.NewDecoder(os.Stdin).Decode(manifest)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer f.Close()
	return manifest, yaml.NewDecoder(f).Decode(manifest)
}
