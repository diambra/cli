package secretsources

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-kit/log"

	"github.com/diambra/cli/pkg/pyarena"
	"github.com/go-kit/log/level"
)

const HFTokenPath = ".cache/huggingface/token"

//go:embed get_token.py
var GetHuggingfaceToken string

type HuggingfaceCredentials struct {
	logger log.Logger
	Home   string
}

func NewHuggingfaceCredentials(logger log.Logger, home string) *HuggingfaceCredentials {
	return &HuggingfaceCredentials{
		logger: logger,
		Home:   home,
	}
}

func (c *HuggingfaceCredentials) Credentials(url string) (map[string]string, error) {
	cmd := exec.Command(pyarena.FindPython(), "-c", GetHuggingfaceToken)
	stdout := &bytes.Buffer{}

	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		level.Debug(c.logger).Log("msg", "couldn't get huggingface token programtically, trying open token file directly", "err", err)
		token, err := os.ReadFile(filepath.Join(c.Home, HFTokenPath))
		if err != nil {
			return nil, fmt.Errorf("couldn't get huggingface token: %w", err)
		}
		return map[string]string{"HF_TOKEN": strings.TrimSpace(string(token))}, nil
	}
	return map[string]string{"HF_TOKEN": strings.TrimSpace(stdout.String())}, nil
}
