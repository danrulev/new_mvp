// main.go
package main

import (
	"desktop_lab/internal/app"
	"embed"
)

//go:embed bin/windows/wkhtmltopdf.exe
var wkhtmltopdfWindows []byte

//go:embed fonts/dejavu-fonts-ttf-2.37/ttf/*.ttf
var fontFS embed.FS

//go:embed templates/protocols/*.html
//go:embed internal/db/migration/*.sql
//go:embed frontend/*
var templateFS embed.FS

func main() {
	// 1. Инициализация бэкенда
	err := app.NewApp(wkhtmltopdfWindows, fontFS, templateFS)
	if err != nil {
		panic(err)
	}
}
