package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Log      LogConfig
}

type ServerConfig struct {
	Host        string        `mapstructure:"host"`
	Port        int           `mapstructure:"port"`
	BaseURL     string        `mapstructure:"base_url"`
	TLSCertFile string        `mapstructure:"tls_cert"`
	TLSKeyFile  string        `mapstructure:"tls_key"`
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
}

type DatabaseConfig struct {
	// "sqlite" or "postgres"
	Type string `mapstructure:"type"`
	// For sqlite: path to .db file. For postgres: DSN string.
	URL string `mapstructure:"url"`
}

type StorageConfig struct {
	// Root data directory. Subdirs created automatically.
	DataDir string `mapstructure:"data_dir"`
}

type LogConfig struct {
	Level string `mapstructure:"level"` // "debug" | "info" | "warn" | "error"
}

func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8096)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.url", "/config/streamvault.db")
	v.SetDefault("storage.data_dir", "/config")
	v.SetDefault("log.level", "info")

	// Config file
	v.SetConfigName("streamvault")
	v.SetConfigType("yaml")
	v.AddConfigPath("/config")
	v.AddConfigPath(".")
	v.ReadInConfig() // not fatal if missing — defaults apply

	// Environment variables: SV_SERVER_PORT → server.port
	v.SetEnvPrefix("SV")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Database.Type != "sqlite" && cfg.Database.Type != "postgres" {
		return fmt.Errorf("database.type must be 'sqlite' or 'postgres', got %q", cfg.Database.Type)
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port out of range: %d", cfg.Server.Port)
	}
	return nil
}
