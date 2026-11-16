//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetStdHandle       = kernel32.NewProc("GetStdHandle")
	procGetConsoleMode     = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode     = kernel32.NewProc("SetConsoleMode")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP       = kernel32.NewProc("SetConsoleCP")
)

// enableWindowsConsole switches the console to UTF-8 and enables ANSI escape
// sequences so the degree sign and weather symbols render correctly instead of
// showing up as garbage in cmd.exe / PowerShell.
func enableWindowsConsole() {
	const (
		utf8CodePage       = 65001
		stdOutputHandle    = ^uintptr(0) - 10 // -11 (STD_OUTPUT_HANDLE)
		enableProcessed    = 0x0001
		enableVTProcessing = 0x0004
	)

	procSetConsoleOutputCP.Call(utf8CodePage)
	procSetConsoleCP.Call(utf8CodePage)

	handle, _, _ := procGetStdHandle.Call(stdOutputHandle)
	if handle == 0 || handle == ^uintptr(0) {
		return
	}

	var mode uint32
	if ok, _, _ := procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode))); ok == 0 {
		return
	}

	procSetConsoleMode.Call(handle, uintptr(mode|enableProcessed|enableVTProcessing))
}
