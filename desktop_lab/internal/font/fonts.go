package font

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExtractFonts извлекает внедренные шрифты во временную папку
// и возвращает путь к этой папке.
func ExtractFonts(fontFS embed.FS) (string, error) {
	// Создаём временную папку для шрифтов
	tempDir, err := os.MkdirTemp("", "lab_fonts_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Путь внутри embed — ВАЖНО: используйте прямые слеши / даже на Windows
	// И звёздочку в go:embed, чтобы захватить все .ttf
	embedPath := "fonts/dejavu-fonts-ttf-2.37/ttf"

	// Читаем список файлов из embed
	entries, err := fontFS.ReadDir(embedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read font directory from embed: %w", err)
	}

	// Копируем каждый .ttf файл во временную папку
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".ttf") {
			continue
		}

		// 🔥 Читаем файл из embed — путь с ПРЯМЫМИ слешами!
		// filepath.ToSlash гарантирует корректный путь для embed
		embedFilePath := filepath.ToSlash(filepath.Join(embedPath, entry.Name()))

		data, err := fontFS.ReadFile(embedFilePath)
		if err != nil {
			return "", fmt.Errorf("ошибка чтения файла шрифта %s из embed: %w", entry.Name(), err)
		}

		// 🔥 Записываем во временную папку — здесь уже нативные пути ОС
		destPath := filepath.Join(tempDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return "", fmt.Errorf("failed to write font file %s: %w", entry.Name(), err)
		}
	}

	return tempDir, nil
}

// CleanupFonts удаляет временную папку со шрифтами
// Вызывайте эту функцию при закрытии приложения (OnShutdown)
func CleanupFonts(fontDir string) {
	if fontDir != "" {
		os.RemoveAll(fontDir)
	}
}
