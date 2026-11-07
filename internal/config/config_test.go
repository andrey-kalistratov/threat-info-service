package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSources(t *testing.T) {
	cases := []struct {
		name string
		path string
		env  map[string]string
	}{
		{
			name: "YAML supplemented by env",
			path: "testdata/base.yaml",
			env: map[string]string{
				"DB_POSTGRES_DATABASE": "threats",
				"DB_POSTGRES_USER":     "threats_app",
				"DB_POSTGRES_PASSWORD": "test-value",
				"DB_POSTGRES_SSLMODE":  "disable",
			},
		},
		{
			name: "Env only",
			env:  envValues(),
		},
		{
			name: "Empty YAML supplemented by env",
			path: "testdata/empty.yaml",
			env:  envValues(),
		},
		{
			name: "Env overrides partial YAML",
			path: "testdata/partial.yaml",
			env:  envValues(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			isolateEnv(t)
			setEnv(t, tc.env)

			expected := validConfig()

			// Act.
			cfg, err := Load(tc.path)

			// Assert.
			require.NoError(t, err)

			require.Equal(t, expected, cfg)
		})
	}
}

func TestLoadErrors(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		env      map[string]string
		expected error
	}{
		{
			name:     "Missing file",
			path:     filepath.Join(t.TempDir(), "missing.yaml"),
			expected: ErrReadConfig,
		},
		{
			name:     "Unsupported format",
			path:     "testdata/unsupported.txt",
			expected: ErrUnsupportedConfigFormat,
		},
		{
			name:     "Invalid YAML value",
			path:     "testdata/invalid.yaml",
			expected: ErrParseYAML,
		},
		{
			name: "Invalid env value",
			env: map[string]string{
				"HTTP_PORT": "sensitive-test-value",
			},
			expected: ErrParseEnv,
		},
		{
			name: "Empty env overrides YAML",
			path: "testdata/base.yaml",
			env: map[string]string{
				"DB_POSTGRES_HOST": "",
			},
			expected: ErrEmptyValue,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			isolateEnv(t)
			setEnv(t, envValues())
			setEnv(t, tc.env)

			// Act.
			_, err := Load(tc.path)

			// Assert.
			require.ErrorIs(t, err, tc.expected)

			require.NotContains(t, err.Error(), "sensitive-test-value")
		})
	}
}

func TestLoadPreservesSensitiveValues(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name:     "YAML",
			expected: `  $word:"/\ test  `,
		},
		{
			name: "Env",
			//nolint:gosec // Искусственные данные проверяют сохранение спецсимволов в env.
			env: map[string]string{
				"DB_POSTGRES_PASSWORD": `  $env:"/\ test  `,
			},
			expected: `  $env:"/\ test  `,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			isolateEnv(t)
			setEnv(t, tc.env)

			// Act.
			cfg, err := Load("testdata/sensitive.yaml")

			// Assert.
			require.NoError(t, err)

			require.Equal(t, tc.expected, cfg.DB.Postgres.Password)
		})
	}
}

func TestHTTP_Address(t *testing.T) {
	// Arrange.
	cfg := HTTP{
		Host: "::1",
		Port: 9090,
	}

	// Act.
	address := cfg.Address()

	// Assert.
	require.Equal(t, "[::1]:9090", address)
}

func isolateEnv(t *testing.T) {
	t.Helper()

	for name := range envValues() {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func setEnv(t *testing.T, values map[string]string) {
	t.Helper()

	for name, value := range values {
		t.Setenv(name, value)
	}
}

func envValues() map[string]string {
	return map[string]string{
		"APP_SHUTDOWN_TIMEOUT":     "10s",
		"OBSERVABILITY_LOG_LEVEL":  "info",
		"OBSERVABILITY_LOG_FORMAT": "json",
		"HTTP_HOST":                "localhost",
		"HTTP_PORT":                "8080",
		"HTTP_READ_TIMEOUT":        "10s",
		"HTTP_WRITE_TIMEOUT":       "10s",
		"HTTP_IDLE_TIMEOUT":        "60s",
		"HTTP_MAX_HEADER_BYTES":    "16384",
		"HTTP_MAX_BODY_BYTES":      "65536",
		"DB_POSTGRES_HOST":         "localhost",
		"DB_POSTGRES_PORT":         "5432",
		"DB_POSTGRES_DATABASE":     "threats",
		"DB_POSTGRES_USER":         "threats_app",
		"DB_POSTGRES_PASSWORD":     "test-value",
		"DB_POSTGRES_SSLMODE":      "disable",
	}
}
