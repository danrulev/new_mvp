package main

import (
	"desktop_lab/internal/app"
	// Если нужен сид
	// Если нужен логгер здесь
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed templates/protocols/*.html
var protocolTemplates embed.FS

//go:embed fonts/dejavu-fonts-ttf-2.37/ttf/*.ttf
var fontFiles embed.FS

func main() {
	// 1. Инициализация бэкенда через app.Init
	// Передаем внедренные файлы шрифтов и шаблонов
	backendApp, err := app.Init(fontFiles, protocolTemplates)
	if err != nil {
		fmt.Println("Fatal error during initialization:", err)
		os.Exit(1)
	}

	// 2. (Опционально) Сиды можно вызвать здесь, если они нужны до старта UI
	// Но лучше делать это внутри app.Init или по запросу из UI
	// if err := data.SeedData(backendApp.Services(), logger); err != nil { ... }

	// 3. Запуск Wails
	err = wails.Run(&options.App{
		Title:     "Лабораторная Система",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},

		// Регистрируем методы для вызова из JS
		Bind: []interface{}{
			backendApp,
		},

		OnStartup:  backendApp.Startup,
		OnShutdown: backendApp.Shutdown,
	})

	if err != nil {
		fmt.Println("Error running application:", err)
		os.Exit(1)
	}
}
