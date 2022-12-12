package agents

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/diambra/cli/pkg/diambra"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/sergi/go-diff/diffmatchpatch"
)

//go:embed dockerfile.tmpl
var DockerfileTemplate string

//go:embed requirements.txt.tmpl
var RequirementsTxt string

//go:embed agent.py.tmpl
var AgentPyTemplate string

//go:embed submission.yaml.tmpl
var SubmissionTemplate string

//go:embed README.md.tmpl
var ReadmeTemplate string

var differ = diffmatchpatch.New()

type PythonConfig struct {
	Version string
}

type BaseImageConfig struct {
	Registry string
	Image    string
}

type ArenaConfig struct {
	Version string
}

type Config struct {
	Python    PythonConfig
	BaseImage BaseImageConfig
	Arena     ArenaConfig
}

func NewConfig(logger log.Logger) (*Config, error) {
	parts, err := diambra.GetInstalledDiambraArenaVersion()
	if err != nil || len(parts) != 3 || (parts[0] == "0" && parts[1] == "0" && parts[2] == "0") {
		level.Info(logger).Log("msg", "can't find local diambra-arena version, using latest", "err", err)
		parts, err = diambra.GetLatestDiambraArenaVersion()
		if err != nil {
			return nil, err
		}
	}
	level.Debug(logger).Log("msg", "using diambra-arena version", "version", strings.Join(parts, "."))
	return &Config{
		Arena: ArenaConfig{
			Version: strings.Join(parts, "."),
		},
		Python: PythonConfig{
			Version: "3.7", // FIXME: Detect version
		},
		BaseImage: BaseImageConfig{
			Registry: "docker.io",
			Image:    "python",
		},
	}, nil
}

func WriteFile(logger log.Logger, path, name, tmpl string, config *Config) error {
	exists := true
	if _, err := os.Stat(filepath.Join(path, name)); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error checking if %s exists: %w", name, err)
		}
		exists = false
	}

	if exists {
		fh, err := os.Open(filepath.Join(path, name))
		if err != nil {
			return err
		}
		defer fh.Close()

		old, err := ioutil.ReadAll(fh)
		if err != nil {
			return fmt.Errorf("couldn't read existing file %s: %w", name, err)
		}
		new := bytes.Buffer{}
		if err := template.Must(template.New(name).Parse(tmpl)).Execute(&new, config); err != nil {
			return err
		}
		diffs := differ.DiffMain(new.String(), string(old), true)
		if len(diffs) > 1 {
			level.Info(logger).Log("msg", name+" has local changes, skipping:", "name", name)
			fmt.Println(differ.DiffPrettyText(diffs))
			return nil
		}
		level.Info(logger).Log("msg", "Skipping "+name+", content identical", "file", name)
		return nil
	}
	fh, err := os.Create(filepath.Join(path, name))
	if err != nil {
		return err
	}
	level.Info(logger).Log("msg", "Creating "+name, "file", name)
	return template.Must(template.New(name).Parse(tmpl)).Execute(fh, config)
}

func Generate(logger log.Logger, path string, config *Config) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	for name, tmpl := range map[string]string{
		"Dockerfile":       DockerfileTemplate,
		"requirements.txt": RequirementsTxt,
		"agent.py":         AgentPyTemplate,
		"submission.yaml":  SubmissionTemplate,
		"README.md":        ReadmeTemplate,
	} {
		if err := WriteFile(logger, path, name, tmpl, config); err != nil {
			return err
		}
	}
	return nil
}
