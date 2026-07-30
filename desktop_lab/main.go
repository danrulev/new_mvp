// main.go
package main

import (
	"context"
	"desktop_lab/internal/app"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	// Создаем канал для получения сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Канал для получения результата работы приложения
	appDone := make(chan error, 1)

	// Запускаем приложение в горутине
	go func() {
		appDone <- app.NewApp(wkhtmltopdfWindows, fontFS, templateFS)
	}()

	// Ждем либо сигнала завершения, либо ошибки от приложения
	select {
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "Received signal %v, initiating graceful shutdown...\n", sig)

		// Здесь можно добавить логику уведомления приложения о необходимости завершения
		// Например, через контекст или канал

	case err := <-appDone:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
			os.Exit(1)
		}
		// Приложение завершилось успешно
		return
	}

	// Graceful shutdown - даем приложению время на завершение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ждем завершения приложения с таймаутом
	select {
	case err := <-appDone:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Application shutdown error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Application shut down gracefully")
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "Shutdown timeout exceeded, forcing exit\n")
		os.Exit(1)
	}
}
