package agents

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

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
	Secret    bool
}

func NewConfig() (*Config, error) {
	return &Config{
		Arena: ArenaConfig{},
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
		diffs := differ.DiffMain(string(old), new.String(), true)
		if len(diffs) < 2 {
			level.Info(logger).Log("msg", "Skipping "+name+", content identical", "file", name)
			return nil
		}

		level.Info(logger).Log("msg", name+" has local changes:", "name", name)
		fmt.Println(differ.DiffPrettyText(diffs))
		level.Info(logger).Log("msg", "Overwrite "+name+"? [y/N]", "name", name)

		var answer string
		// FIXME: There must be a better way to do this
		if _, err := fmt.Scanln(&answer); err != nil && err.Error() != "unexpected newline" {
			return fmt.Errorf("couldn't read answer: %w", err)
		}
		if answer != "y" {
			level.Info(logger).Log("msg", "Skipping "+name, "name", name)
			return nil
		}
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
	if config.Secret {
		return generateWithSecrets(logger, path, config)
	}
	return generateWithoutSecrets(logger, path, config)
}

func generateWithoutSecrets(logger log.Logger, path string, config *Config) error {
	for name, tmpl := range map[string]string{
		"Dockerfile":       DockerfileTemplate,
		"requirements.txt": RequirementsTxt,
		"agent.py":         AgentPyTemplate,
		"README.md":        ReadmeTemplate,
	} {
		if err := WriteFile(logger, path, name, tmpl, config); err != nil {
			return err
		}
	}
	return nil
}

func generateWithSecrets(logger log.Logger, path string, config *Config) error {
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
