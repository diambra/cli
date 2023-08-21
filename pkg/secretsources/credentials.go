package secretsources

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

type CredentialProvider interface {
	Credentials(url string) (map[string]string, error)
}

type GitCredentials struct {
	Helper string
}

func (c *GitCredentials) Credentials(url string) (map[string]string, error) {
	args := []string{}
	if c.Helper != "" {
		args = append(args, "-c", fmt.Sprintf("credential.helper=%s", c.Helper))
	}
	args = append(args, "credential", "fill")
	cmd := exec.Command("git", args...)
	cmd.Stdin = strings.NewReader("url=" + url + "\n")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run %v: %w", cmd, err)
	}

	credentials := make(map[string]string)
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			credentials[parts[0]] = parts[1]
		}
	}

	return credentials, nil
}

// CredentialsFill calls the CredentialsProvider for each source and returns
// a new source map with templating as well as a map of credentials for the templated values.
func CredentialsFill(provider CredentialProvider, sources map[string]string) (map[string]string, error) {
	secrets := make(map[string]string)
	i := 0
	for k, v := range sources {
		i++
		u, err := url.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url %s: %w", v, err)
		}
		credentials, err := provider.Credentials(v)
		if err != nil {
			return nil, err
		}
		if credentials["password"] == "" {
			continue
		}

		if credentials["host"] != u.Host {
			return nil, fmt.Errorf("host %s does not match %s (this should never happend)", credentials["host"], u.Host)
		}

		var (
			uservar = fmt.Sprintf("git_username_%d", i)
			passvar = fmt.Sprintf("git_password_%d", i)
		)

		u.User = url.UserPassword(fmt.Sprintf("{{ %s }}", uservar), fmt.Sprintf("{{ %s }}", passvar))
		secrets[uservar] = credentials["username"]
		secrets[passvar] = credentials["password"]
		sources[k] = fmt.Sprintf("%s://{{ .Secrets.%s }}:{{ .Secrets.%s }}@%s%s", u.Scheme, uservar, passvar, u.Host, u.Path)
	}
	return secrets, nil
}
