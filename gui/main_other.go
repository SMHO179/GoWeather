//go:build !windows

package main

import (
	"fmt"
	"os"
)

// The native GUI uses Win32 directly, so it only exists on Windows. This stub
// keeps "go build ./..." working on Linux and macOS (including CI).
func main() {
	fmt.Fprintln(os.Stderr, "The GoWeather GUI is only available on Windows. Use the CLI or the weather-gui.exe release build.")
	os.Exit(1)
}
