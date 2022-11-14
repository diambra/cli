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

// Mode Enum
type Mode string

const (
	ModeAIvsCOM Mode = "AIvsCOM"

	API = "https://diambra.ai/api/v1alpha1"
)

type submission struct {
	Manifest map[string]interface{} `json:"manifest"`
}

// FIXME: Replace this with oapi generated code
func Submit(logger log.Logger, image string, mode Mode, homedir string, envVars map[string]string) error {
	m := submission{
		Manifest: map[string]interface{}{
			"image": image,
			"mode":  mode,
		},
	}
	for k, v := range envVars {
		m.Manifest[k] = v
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
