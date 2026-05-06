// main.go
package main

import (
	"desktop_lab/internal/app"
	"embed"
)

// ✅ Встраиваем шрифты — обратите внимание на звёздочку рекурсивно
//go:embed fonts/dejavu-fonts-ttf-2.37/ttf/*.ttf
var fontFS embed.FS

// ✅ Если нужно встроить ещё и шаблоны/миграции:
//go:embed templates/protocols/*.html
//go:embed internal/db/migration/*.sql
//go:embed frontend/*
var templateFS embed.FS

func main() {
	// 1. Инициализация бэкенда
	err := app.NewApp(fontFS, templateFS)
	if err != nil {
		panic(err)
	}
}
