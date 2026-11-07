package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfig_validate_Valid(t *testing.T) {
	// Arrange.
	isolateEnv(t)
	cfg := validConfig()

	// Act.
	err := cfg.validate()

	// Assert.
	require.NoError(t, err)
}

func TestConfig_validate_Invalid(t *testing.T) {
	isolateEnv(t)

	cases := []struct {
		name    string
		change  func(*Config)
		wantErr error
	}{
		{
			name:    "Missing required value",
			change:  func(c *Config) { c.DB.Postgres.Password = "" },
			wantErr: ErrEmptyValue,
		},
		{
			name:    "Port below range",
			change:  func(c *Config) { c.HTTP.Port = 0 },
			wantErr: ErrInvalidPort,
		},
		{
			name:    "Port above range",
			change:  func(c *Config) { c.DB.Postgres.Port = 65536 },
			wantErr: ErrInvalidPort,
		},
		{
			name:    "Zero timeout",
			change:  func(c *Config) { c.App.ShutdownTimeout = 0 },
			wantErr: ErrNonPositiveValue,
		},
		{
			name:    "Negative timeout",
			change:  func(c *Config) { c.HTTP.WriteTimeout = -time.Second },
			wantErr: ErrNonPositiveValue,
		},
		{
			name:    "Zero header limit",
			change:  func(c *Config) { c.HTTP.MaxHeaderBytes = 0 },
			wantErr: ErrNonPositiveValue,
		},
		{
			name:    "Negative body limit",
			change:  func(c *Config) { c.HTTP.MaxBodyBytes = -1 },
			wantErr: ErrNonPositiveValue,
		},
		{
			name:    "Unsupported log level",
			change:  func(c *Config) { c.Observability.Log.Level = "unknown" },
			wantErr: ErrLogLevel,
		},
		{
			name:    "Unsupported log format",
			change:  func(c *Config) { c.Observability.Log.Format = "xml" },
			wantErr: ErrLogFormat,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			cfg := validConfig()
			tc.change(cfg)

			// Act.
			err := cfg.validate()

			// Assert.
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func validConfig() *Config {
	return &Config{
		App: App{
			ShutdownTimeout: 10 * time.Second,
		},
		Observability: Observability{
			Log: Log{
				Level:  "info",
				Format: "json",
			},
		},
		HTTP: HTTP{
			Host:           "localhost",
			Port:           8080,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    time.Minute,
			MaxHeaderBytes: 16384,
			MaxBodyBytes:   65536,
		},
		DB: DB{
			Postgres: Postgres{
				Host:     "localhost",
				Port:     5432,
				Database: "threats",
				User:     "threats_app",
				Password: "test-value",
				SSLMode:  "disable",
			},
		},
	}
}
