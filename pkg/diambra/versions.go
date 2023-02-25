package diambra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/diambra/cli/pkg/pyarena"
)

const DiambraArenaPyPiJSONURL = "https://pypi.org/pypi/diambra-arena/json"

// Use pyarena script find package versions. Use for finding env image and generating
// requirements.txt
func GetInstalledPackageVersion(packageName string) ([]string, error) {
	cmd := exec.Command(pyarena.FindPython(), "-c", pyarena.GetDiambraEngineVersion, packageName)
	stdout := &bytes.Buffer{}

	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(stdout.String()), "."), nil
}

type pypiJSON struct {
	Info struct {
		Version string `json:"version"`
	} `json:"info"`
}

func GetLatestDiambraArenaVersion() ([]string, error) {
	resp, err := http.Get(DiambraArenaPyPiJSONURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest diambra-arena version: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get latest diambra-arena version: %s", resp.Status)
	}
	var pypi pypiJSON
	if err := json.NewDecoder(resp.Body).Decode(&pypi); err != nil {
		return nil, fmt.Errorf("failed to get latest diambra-arena version: %w", err)
	}
	return strings.Split(pypi.Info.Version, "."), nil
}
