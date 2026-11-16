# GoWeather

Minimal, dependency-free weather for the terminal and Windows 11 — powered by [wttr.in](https://wttr.in), no API key required.

## Features

- **CLI**: fast, cross-platform weather lookup
- **Windows 11 GUI widget**: native frameless Win32 app with smooth animations, live conditions, and 3-day forecast
- **Zero dependencies**: Go standard library only
- **No setup**: just build and run

## CLI

```bash
go run .                # default: Tehran
go run . London
go run . "New York"
go build -o weather .   # build binary
```

## Windows GUI

```powershell
go build -trimpath -ldflags "-s -w -H windowsgui" -o weather-gui.exe .\gui
.\weather-gui.exe
```

- Search any city (Enter to refresh, F5 / Ctrl+R to reload)
- Pin on top, minimize, or drag the widget anywhere
- Current conditions, feels-like, humidity, wind, pressure, visibility, UV, sunrise/sunset
- Animated icons and 3-day forecast with temperature range

## Project layout

```text
main.go                CLI entry point
internal/weather/      wttr.in client and models
gui/                   Native Windows 11 widget (Win32, no CGO)
```

## License

[BSD 3-Clause](LICENSE)
