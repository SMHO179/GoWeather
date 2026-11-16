//go:build !windows

package main

// enableWindowsConsole is a no-op outside Windows.
func enableWindowsConsole() {}
