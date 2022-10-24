package diambra

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/log"
)

// Mode Enum
type Mode string

const (
	ModeAIvsCOM Mode = "AIvsCOM"

	API = "https://diambra.ai/api/v1alpha1"
)

// FIXME: Replace this with oapi generated code
func Submit(_ log.Logger, image string, mode Mode, homedir string) error {
	credFile := filepath.Join(homedir, ".diambra", "credentials")
	b, err := os.ReadFile(filepath.Join(credFile))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		"POST",
		API+"/submit",
		strings.NewReader(fmt.Sprintf(`{ "manifest": {"image": "%s", "mode": "%s"} }`, image, mode)),
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
