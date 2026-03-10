package config

import (
	"desktop_lab/pkg/valid"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// DBConfig теперь содержит только путь к файлу базы данных
type DBConfig struct {
	Path string `mapstructure:"path" validate:"required"` // Путь к файлу, например: "./data/app.db"
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
	DB     DBConfig     `mapstructure:"db"`
	Logger LoggerConfig `mapstructure:"logger"`
}

func NewConfig() (*Config, error) {
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)

	// Формируем абсолютный путь к БД в той же папке, где лежит exe
	dbPath := filepath.Join(execDir, "lab_data.db")
	// Загружаем .env файл, если он существует
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
		// Если файла нет, продолжаем работу (переменные могут быть заданы в OS env)
	}

	v := viper.New()
	v.AutomaticEnv()

	// Путь к директории с конфигами (опционально, можно убрать для десктопа)
	v.AddConfigPath("./configs")

	name := v.GetString("CONFIG_NAME")
	if name == "" {
		name = "default"
	}

	v.SetConfigName(name)

	// Читаем конфиг файл (yaml/json/toml), если он есть
	// Ошибка игнорируется, если мы полагаемся только на ENV переменные для десктопа
	if err := v.ReadInConfig(); err != nil {
		// Для десктопного приложения конфиг может отсутствовать, если все задано в ENV
		// Но если вы хотите жестко требовать файл, раскомментируйте return:
		return nil, fmt.Errorf("failed to read config file: %w", err)

		// Если файла нет, проверяем, заданы ли обязательные переменные окружения вручную
		// Это позволяет работать без файлов конфигов вообще
	}

	cfg := Config{}
	err := v.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	cfg.DB.Path = dbPath

	// Валидация структуры
	// Убедитесь, что в pkg/valid учтено, что DB теперь требует только Path
	if err := valid.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}
