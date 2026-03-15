#!/bin/bash

# Экспортируем флаги для GTK и WebKit 4.1
export CGO_CFLAGS=$(pkg-config --cflags gtk+-3.0 webkit2gtk-4.1)
export CGO_LDFLAGS=$(pkg-config --libs gtk+-3.0 webkit2gtk-4.1)

# Запускаем wails
wails build -tags webkit2_41