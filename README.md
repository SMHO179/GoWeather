# 🌦 GoWeather

<div align="center">

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-4CAF50?style=for-the-badge)
![CLI](https://img.shields.io/badge/Interface-CLI%20%2B%20Windows%20GUI-7E57C2?style=for-the-badge)
![License](https://img.shields.io/github/license/SMHO179/GoWeather?style=for-the-badge)
![Stars](https://img.shields.io/github/stars/SMHO179/GoWeather?style=for-the-badge)
![Last Commit](https://img.shields.io/github/last-commit/SMHO179/GoWeather?style=for-the-badge)

</div>

A fast, dependency-free weather tool written in **Go**.

It fetches real-time weather data from **wttr.in** without an API key and ships
in two flavours:

- **`weather`** – a cross-platform command-line app.
- **`weather-gui`** – a native Windows desktop app (Win32, no CGO, no
  third-party modules, no installer required).

---

## 🚀 CLI usage

```bash
go run .                 # uses Tehran
go run . London
go run . "New York"
go run . -timeout 5s Baku
go run . -pause          # waits for Enter before closing (Windows double-click)
```

Build the CLI:

```bash
go build -o weather
./weather Baku
```

On Windows this creates `weather.exe`, and `-o weather` becomes `weather.exe`
automatically. The CLI switches the console to UTF-8 on startup, so `°C`,
weather symbols and non-Latin city names render correctly in `cmd.exe` and
PowerShell.

## 🪟 Windows GUI

```powershell
go build -trimpath -ldflags "-s -w -H windowsgui" -o weather-gui.exe .\gui
.\weather-gui.exe
```

The `-H windowsgui` linker flag removes the console window. The GUI offers:

- a city field pre-filled with Tehran,
- an async **Get Weather** button (the window never freezes),
- current conditions, humidity, wind, pressure, visibility and UV index,
- a three-day forecast,
- automatic DPI scaling and keyboard navigation (Tab / Enter).

The native UI uses `user32.dll` and `gdi32.dll` through the Go standard
library only, so it compiles on a clean Windows machine with just Go installed.

## 🐧 Build on Linux / macOS

```bash
go build -o weather .
```

The GUI package is Windows-only; on other platforms it builds to a stub that
tells the user to use the CLI. This keeps `go build ./...` green everywhere.

## ✨ Features

- 🔑 No API key required
- 📦 Zero third-party dependencies (standard library only)
- ⚡ Fast and lightweight
- 💻 Cross-platform CLI with UTF-8 Windows console support
- 🪟 Native Windows GUI built with Win32
- 🌍 Supports any city worldwide
- ⏱️ Configurable request timeout and graceful error messages

## 📂 Project layout

```text
main.go                      CLI entry point
console_windows.go           UTF-8 / ANSI console setup for Windows
console_other.go             no-op for other platforms
internal/weather/            shared wttr.in client, JSON models and formatting
gui/main_windows.go          native Win32 GUI
gui/main_other.go            non-Windows stub
```

## 📷 Example

```text
Weather in Baku
-------------------------
Baku: ✨ +11°C
```

GUI output includes the location, current conditions and the three-day
forecast rendered in a read-only text panel.

## ❤️ Powered by

- https://wttr.in
