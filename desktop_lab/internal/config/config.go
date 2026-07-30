package config

import (
	"desktop_lab/pkg/valid"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type AuthCfg struct {
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl" validate:"required"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl" validate:"required"`
	JwtSecret       string        `mapstructure:"jwt_secret" validate:"required"`
}

type ServerConfig struct {
	Host           string        `mapstructure:"host" validate:"required"`
	Port           string        `mapstructure:"port" validate:"required"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout" validate:"required"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout" validate:"required"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes" validate:"required"`
}

type DBConfig struct {
	Path           string `mapstructure:"path" validate:"required"`
	AllowSelection bool   `mapstructure:"allow_selection"`
}

type LoggerConfig struct {
	Level             string   `mapstructure:"level" validate:"required"`
	Development       bool     `mapstructure:"development"`
	DisableCaller     bool     `mapstructure:"disable_caller"`
	DisableStacktrace bool     `mapstructure:"disable_stacktrace"`
	Encoding          string   `mapstructure:"encoding" validate:"required"`
	OutputPaths       []string `mapstructure:"output_paths"`
	ErrorOutputPaths  []string `mapstructure:"error_output_paths"`
}

type Config struct {
	Auth   AuthCfg      `mapstructure:"auth"`
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"db"`
	Logger LoggerConfig `mapstructure:"logger"`
}

func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	v := viper.New()
	v.AutomaticEnv()
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")
	v.SetConfigType("yaml")

	configName := v.GetString("CONFIG_NAME")
	if configName == "" {
		configName = "default"
	}
	v.SetConfigName(configName)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	cfg := Config{}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.DB.Path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		cfg.DB.Path = filepath.Join(wd, "lab_data.db")
	}

	if err := valid.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}
