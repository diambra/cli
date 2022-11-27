package diambra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"gopkg.in/yaml.v2"
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
	Difficulty string            `yaml:"difficulty" json:"difficulty"`
	Command    []string          `yaml:"command" json:"command"`
	Env        map[string]string `yaml:"env" json:"env"`
	Sources    map[string]string `yaml:"sources" json:"sources"`
}

type submission struct {
	Manifest Manifest          `json:"manifest"`
	Secrets  map[string]string `json:"secrets"`
}

type submitResponse struct {
	submission
	ID int `json:"id"`
}

func Submit(logger log.Logger, image string, mode Mode, homedir string, envVars, sources, secrets map[string]string, manifestPath string) (int, error) {
	// Decode manifestPath
	var manifest Manifest
	if manifestPath != "" {
		f, err := os.Open(manifestPath)
		if err != nil {
			return 0, fmt.Errorf("failed to open manifest: %w", err)
		}
		defer f.Close()
		if err := yaml.NewDecoder(f).Decode(&manifest); err != nil {
			return 0, fmt.Errorf("failed to decode manifest: %w", err)
		}
	}
	if image != "" {
		manifest.Image = image
	}

	for k, v := range envVars {
		manifest.Env[k] = v
	}
	for k, v := range sources {
		manifest.Sources[k] = v
	}

	m := submission{
		Manifest: manifest,
		Secrets:  secrets,
	}

	data, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	credFile := filepath.Join(homedir, ".diambra", "credentials")
	b, err := os.ReadFile(filepath.Join(credFile))
	if err != nil {
		return 0, err
	}
	level.Debug(logger).Log("msg", "Submitting", "data", string(data))
	req, err := http.NewRequest(
		"POST",
		API+"/submit",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+string(b))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("failed to submit: %s", resp.Status)
	}
	var s submitResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return 0, err
	}
	return s.ID, nil
}
