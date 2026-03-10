#!/bin/bash

# Экспортируем флаги для GTK и WebKit 4.1
export CGO_CFLAGS=$(pkg-config --cflags gtk+-3.0 webkit2gtk-4.1)
export CGO_LDFLAGS=$(pkg-config --libs gtk+-3.0 webkit2gtk-4.1)

# Выводим для отладки (можно убрать потом)
echo "Starting Wails with CGO flags..."
echo "CFLAGS: $CGO_CFLAGS"
echo "LDFLAGS: $CGO_LDFLAGS"

# Запускаем wails
wails build -debug

