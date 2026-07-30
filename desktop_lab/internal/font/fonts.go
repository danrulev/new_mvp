package font

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ExtractFonts(fontFS embed.FS) (string, error) {
	tempDir, err := os.MkdirTemp("", "lab_fonts_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	embedPath := "fonts/dejavu-fonts-ttf-2.37/ttf"

	entries, err := fontFS.ReadDir(embedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read font directory from embed: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".ttf") {
			continue
		}

		embedFilePath := filepath.ToSlash(filepath.Join(embedPath, entry.Name()))

		data, err := fontFS.ReadFile(embedFilePath)
		if err != nil {
			return "", fmt.Errorf("ошибка чтения файла шрифта %s из embed: %w", entry.Name(), err)
		}

		destPath := filepath.Join(tempDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return "", fmt.Errorf("failed to write font file %s: %w", entry.Name(), err)
		}
	}

	return tempDir, nil
}

// CleanupFonts удаляет временную папку со шрифтами
func CleanupFonts(fontDir string) {
	if fontDir != "" {
		os.RemoveAll(fontDir)
	}
}
