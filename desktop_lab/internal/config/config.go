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

type ServerConfig struct {
	Host           string        `mapstructure:"host" validate:"required"`
	Port           string        `mapstructure:"port" validate:"required"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout" validate:"required"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout" validate:"required"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes" validate:"required"`
}

// DBConfig теперь содержит только путь к файлу базы данных
type DBConfig struct {
	Path           string `mapstructure:"path" validate:"required"` // Путь к файлу, например: "./data/app.db"
	AllowSelection bool
}

type LoggerConfig struct {
	Level             string   `mapstructure:"level"`
	Development       bool     `mapstructure:"development"`
	DisableCaller     bool     `mapstructure:"disable_caller"`
	DisableStacktrace bool     `mapstructure:"disable_stacktrace"`
	Encoding          string   `mapstructure:"encoding"`
	OutputPaths       []string `mapstructure:"output_paths"`
	ErrorOutputPaths  []string `mapstructure:"error_output_paths"`
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"db"`
	Logger LoggerConfig `mapstructure:"logger"`
}

// internal/config/config.go

func NewConfig() (*Config, error) {
	// Получаем рабочую директорию (где запущен go run / wails dev)
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Путь к БД в корне проекта (для dev)
	// Для продакшена можно оставить execDir
	dbPath := filepath.Join(wd, "lab_data.db")

	// Загружаем .env файл, если он существует
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	v := viper.New()
	v.AutomaticEnv()
	v.AddConfigPath("./configs")

	name := v.GetString("CONFIG_NAME")
	if name == "" {
		name = "default"
	}
	v.SetConfigName(name)

	if err := v.ReadInConfig(); err != nil {
		// Игнорируем, если файла нет — используем ENV
	}

	cfg := Config{}
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// ✅ Переопределяем путь к БД на фиксированный
	cfg.DB.Path = dbPath

	if err := valid.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}
