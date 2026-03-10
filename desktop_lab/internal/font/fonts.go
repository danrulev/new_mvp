package font

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// ExtractFonts извлекает внедренные шрифты во временную папку
// и возвращает путь к этой папке.
func ExtractFonts(fontFS embed.FS) (string, error) {
	// Создаем уникальную временную папку для шрифтов
	tempDir, err := os.MkdirTemp("", "lab-desktop-fonts-*")
	if err != nil {
		return "", fmt.Errorf("ошибка создания временной папки для шрифтов: %w", err)
	}

	// Проходим по всем файлам в embed FS
	entries, err := fontFS.ReadDir("fonts/dejavu-fonts-ttf-2.37/ttf")
	if err != nil {
		return "", fmt.Errorf("ошибка чтения директории шрифтов: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Читаем файл из embed
		data, err := fontFS.ReadFile(filepath.Join("fonts/dejavu-fonts-ttf-2.37/ttf", entry.Name()))
		if err != nil {
			return "", fmt.Errorf("ошибка чтения файла шрифта %s: %w", entry.Name(), err)
		}

		// Пишем файл во временную папку
		destPath := filepath.Join(tempDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return "", fmt.Errorf("ошибка записи шрифта %s: %w", entry.Name(), err)
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
