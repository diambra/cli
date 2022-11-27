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
)

// FIXME: Replace this with oapi generated code

// Mode Enum
type Mode string

const (
	ModeAIvsCOM Mode = "AIvsCOM"

	API = "https://diambra.ai/api/v1alpha1"
)

type Manifest struct {
	Image      string            `json:"image"`
	Mode       Mode              `json:"mode"`
	Difficulty string            `json:"difficulty"`
	Command    []string          `json:"command"`
	Env        map[string]string `json:"env"`
	Sources    map[string]string `json:"sources"`
}

type submission struct {
	Manifest Manifest          `json:"manifest"`
	Secrets  map[string]string `json:"secrets"`
}

func Submit(logger log.Logger, image string, mode Mode, homedir string, envVars, sources, secrets map[string]string, manifestPath string) error {
	// Decode manifestPath
	var manifest Manifest
	if manifestPath != "" {
		f, err := os.Open(manifestPath)
		if err != nil {
			return fmt.Errorf("failed to open manifest: %w", err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&manifest); err != nil {
			return fmt.Errorf("failed to decode manifest: %w", err)
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
		return err
	}
	credFile := filepath.Join(homedir, ".diambra", "credentials")
	b, err := os.ReadFile(filepath.Join(credFile))
	if err != nil {
		return err
	}
	level.Debug(logger).Log("msg", "Submitting", "data", string(data))
	req, err := http.NewRequest(
		"POST",
		API+"/submit",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+string(b))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to submit: %s", resp.Status)
	}
	return nil
}
