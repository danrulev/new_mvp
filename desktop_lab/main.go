// main.go
package main

import (
	"desktop_lab/internal/app"
	"embed"
	"fmt"
	"log"
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
	// 1. Инициализация бэкенда
	app := app.New()
	if err := app.Init(fontFiles, protocolTemplates); err != nil {
		log.Fatal(err)
	}

	// 2. Запуск Wails приложения
	err := wails.Run(&options.App{
		Title:     "Лаборатория материалов",
		Width:     1400,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 248, G: 250, B: 252, A: 1},
		OnStartup:        app.Startup,
		OnDomReady:       app.DomReady,
		OnBeforeClose:    app.BeforeClose,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app, // Экспортируем методы *App в JS
		},
	})

	if err != nil {
		fmt.Println("Error running application:", err)
		os.Exit(1)
	}
}
