// Package config загружает и проверяет настройки приложения из YAML и окружения.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/goccy/go-yaml"
)

var (
	// ErrUnsupportedConfigFormat означает неподдерживаемое расширение конфигурационного файла.
	ErrUnsupportedConfigFormat = errors.New("expected a .yaml or .yml file")

	// ErrReadConfig означает невозможность прочитать явно указанный YAML.
	ErrReadConfig = errors.New("cannot read YAML file")

	// ErrParseYAML означает ошибку синтаксиса или типа поля YAML.
	ErrParseYAML = errors.New("invalid YAML syntax or field type")

	// ErrParseEnv означает ошибку разбора значения переменной окружения.
	ErrParseEnv = errors.New("cannot parse environment value types")
)

// Config содержит итоговые настройки приложения.
type Config struct {
	App           App           `yaml:"app" envPrefix:"APP_"`
	Observability Observability `yaml:"observability" envPrefix:"OBSERVABILITY_"`
	HTTP          HTTP          `yaml:"http" envPrefix:"HTTP_"`
	DB            DB            `yaml:"db" envPrefix:"DB_"`
}

// App содержит общие настройки жизненного цикла приложения.
type App struct {
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT"`
}

// Observability содержит настройки наблюдаемости.
type Observability struct {
	Log Log `yaml:"log" envPrefix:"LOG_"`
}

// Log содержит настройки логирования.
type Log struct {
	Level  string `yaml:"level" env:"LEVEL"`
	Format string `yaml:"format" env:"FORMAT"`
}

// HTTP содержит настройки HTTP-сервера.
type HTTP struct {
	Host           string        `yaml:"host" env:"HOST"`
	Port           int           `yaml:"port" env:"PORT"`
	ReadTimeout    time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT"`
	WriteTimeout   time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" env:"IDLE_TIMEOUT"`
	MaxHeaderBytes int           `yaml:"max_header_bytes" env:"MAX_HEADER_BYTES"`
	MaxBodyBytes   int64         `yaml:"max_body_bytes" env:"MAX_BODY_BYTES"`
}

// DB объединяет настройки хранилищ.
type DB struct {
	Postgres Postgres `yaml:"postgres" envPrefix:"POSTGRES_"`
}

// Postgres содержит настройки подключения к PostgreSQL.
type Postgres struct {
	Host     string `yaml:"host" env:"HOST"`
	Port     int    `yaml:"port" env:"PORT"`
	Database string `yaml:"database" env:"DATABASE"`
	User     string `yaml:"user" env:"USER"`
	Password string `yaml:"password" env:"PASSWORD"`
	SSLMode  string `yaml:"sslmode" env:"SSLMODE"`
}

// Load читает явно указанный YAML и перекрывает его окружением. Пустой путь означает только env.
func Load(path string) (*Config, error) {
	var cfg Config

	if path != "" {
		if err := readYAML(path, &cfg); err != nil {
			return nil, err
		}
	}

	if err := env.Parse(&cfg); err != nil {
		// Ошибка разбора env может содержать чувствительные данные. Не включаем её текст.
		return nil, ErrParseEnv
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Address возвращает адрес для HTTP-сервера, включая корректную запись IPv6.
func (h *HTTP) Address() string {
	return net.JoinHostPort(h.Host, strconv.Itoa(h.Port))
}

func readYAML(path string, cfg *Config) error {
	if ext := filepath.Ext(path); ext != ".yaml" && ext != ".yml" {
		return ErrUnsupportedConfigFormat
	}

	//nolint:gosec // Чтение файла по произвольному пути — назначение этой функции.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrReadConfig, err.Error())
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		// Ошибка разбора может содержать фрагмент YAML с чувствительными данными. Не включаем её текст.
		return ErrParseYAML
	}

	return nil
}
