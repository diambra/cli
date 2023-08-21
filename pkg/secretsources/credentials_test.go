package secretsources

import (
	"reflect"
	"testing"
)

type MockCredentialProvider struct {
	creds map[string]string
}

func (m *MockCredentialProvider) Credentials(url string) (map[string]string, error) {
	return m.creds, nil
}

func TestCredentialsFill(t *testing.T) {
	for _, tc := range []struct {
		name            string
		source          map[string]string
		credentials     map[string]string
		expectedSource  map[string]string
		expectedSecrets map[string]string
	}{
		{
			name: "no credentials",
			source: map[string]string{
				"foo": "git+https://example.com/foo",
			},
			credentials: map[string]string{},
			expectedSource: map[string]string{
				"foo": "git+https://example.com/foo",
			},
			expectedSecrets: map[string]string{},
		},
		{
			name: "single credential",
			source: map[string]string{
				"foo": "git+https://example.com/foo",
			},
			credentials: map[string]string{
				"username": "foo",
				"password": "bar",
				"host":     "example.com",
			},
			expectedSource: map[string]string{
				"foo": "git+https://{{ .Secrets.git_username_1 }}:{{ .Secrets.git_password_1 }}@example.com/foo",
			},
			expectedSecrets: map[string]string{
				"git_username_1": "foo",
				"git_password_1": "bar",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			secrets, err := CredentialsFill(&MockCredentialProvider{tc.credentials}, tc.source)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !reflect.DeepEqual(tc.expectedSource, tc.source) {
				t.Fatalf("expected source %v, got %v", tc.expectedSource, tc.source)
			}
			if !reflect.DeepEqual(tc.expectedSecrets, secrets) {
				t.Fatalf("expected secrets %v, got %v", tc.expectedSecrets, secrets)
			}
		})
	}
}
