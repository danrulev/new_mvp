// main.go
package main

import (
	"desktop_lab/internal/app"
	"embed"
	"fmt"
	"os"
)

//go:embed bin/windows/wkhtmltopdf.exe
var wkhtmltopdfWindows []byte

//go:embed fonts/dejavu-fonts-ttf-2.37/ttf/*.ttf
var fontFS embed.FS

//go:embed templates/protocols/*.html
//go:embed internal/db/migration_sqlite/*.sql
//go:embed internal/db/migration_postgresql/*.sql
//go:embed frontend/*
var templateFS embed.FS

func main() {
	if err := app.NewApp(wkhtmltopdfWindows, fontFS, templateFS); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
