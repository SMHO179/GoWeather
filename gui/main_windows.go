//go:build windows

// Command weather-gui is a small native Win32 window for GoWeather.
//
// It only uses the Go standard library (syscall + unsafe), so it builds
// without CGO, without a C compiler and without third-party modules.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/SMHO179/GoWeather/internal/weather"
)

const (
	appName     = "GoWeather"
	windowClass = "GoWeatherMainWindow"

	idLocationLabel = 1001
	idLocationEdit  = 1002
	idFetchButton   = 1003
	idOutputEdit    = 1004
	idStatusLabel   = 1005
)

const (
	wmCreate         = 0x0001
	wmDestroy        = 0x0002
	wmSize           = 0x0005
	wmClose          = 0x0010
	wmCommand        = 0x0111
	wmCtlColorStatic = 0x0138
	wmFetchDone      = 0x8000 + 1

	bnClicked       = 0
	bsDefPushButton = 0x0001

	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabStop          = 0x00010000
	wsVScroll          = 0x00200000
	wsBorder           = 0x00800000
	wsOverlappedWindow = 0x00CF0000
	wsThickFrame       = 0x00040000
	wsMaximizeBox      = 0x00010000

	wsExClientEdge    = 0x00000200
	wsExControlParent = 0x00010000

	esMultiline   = 0x0004
	esAutoVScroll = 0x0040
	esAutoHScroll = 0x0080
	esReadOnly    = 0x0800
	esWantReturn  = 0x1000

	swShow = 5

	smCXScreen = 0
	smCYScreen = 1

	idiApplication = 32512
	idcArrow       = 32512

	colorWindow  = 5
	colorBtnFace = 15

	fwNormal         = 400
	defaultCharset   = 1
	clearTypeQuality = 5

	transparent = 1
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procIsDialogMessageW     = user32.NewProc("IsDialogMessageW")
	procPostMessageW         = user32.NewProc("PostMessageW")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procEnableWindow         = user32.NewProc("EnableWindow")
	procSetFocus             = user32.NewProc("SetFocus")
	procMoveWindow           = user32.NewProc("MoveWindow")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procLoadIconW            = user32.NewProc("LoadIconW")
	procAdjustWindowRectEx   = user32.NewProc("AdjustWindowRectEx")
	procGetSysColorBrush     = user32.NewProc("GetSysColorBrush")
	procWindowFromDC         = user32.NewProc("WindowFromDC")
	procMessageBoxW          = user32.NewProc("MessageBoxW")
	procSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	procSetBkMode            = gdi32.NewProc("SetBkMode")
	procCreateFontW          = gdi32.NewProc("CreateFontW")
	procDeleteObject         = gdi32.NewProc("DeleteObject")
	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
)

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSmall  uintptr
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	point   point
	private uint32
}

type point struct {
	x int32
	y int32
}

type rect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type fetchResult struct {
	location string
	text     string
	err      error
}

type app struct {
	hwnd     uintptr
	instance uintptr
	scale    float64

	fontUI     uintptr
	fontOutput uintptr

	label  uintptr
	input  uintptr
	button uintptr
	output uintptr
	status uintptr

	client *weather.Client

	fetching bool

	mu     sync.Mutex
	result fetchResult
}

var (
	appPtr          *app
	wndProcCallback uintptr
)

func main() {
	procSetProcessDPIAware.Call()

	instance, _, _ := procGetModuleHandleW.Call(0)
	if instance == 0 {
		os.Exit(1)
	}
	appPtr = &app{
		instance: instance,
		scale:    systemScale(),
		client:   weather.NewClient(),
	}
	a := appPtr

	wndProcCallback = syscall.NewCallback(wndProc)
	className, _ := syscall.UTF16PtrFromString(windowClass)
	windowTitle, _ := syscall.UTF16PtrFromString(appName)

	class := wndClassEx{
		style:      0x0001 | 0x0002, // CS_VREDRAW | CS_HREDRAW
		wndProc:    wndProcCallback,
		instance:   a.instance,
		icon:       loadIcon(),
		cursor:     loadCursor(),
		background: uintptr(colorBtnFace + 1),
		className:  className,
	}
	class.size = uint32(unsafe.Sizeof(class))

	if atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		fatal("Could not register the window class.\n\n" + callErr.Error())
	}

	style := uintptr(wsOverlappedWindow &^ wsThickFrame &^ wsMaximizeBox)
	clientRect := rect{right: int32(a.px(660)), bottom: int32(a.px(540))}
	procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&clientRect)), style, 0, uintptr(wsExControlParent))

	width := clientRect.right - clientRect.left
	height := clientRect.bottom - clientRect.top
	screenWidth, _, _ := procGetSystemMetrics.Call(smCXScreen)
	screenHeight, _, _ := procGetSystemMetrics.Call(smCYScreen)
	x := (int(screenWidth) - int(width)) / 2
	y := (int(screenHeight) - int(height)) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	hwnd, _, callErr := procCreateWindowExW.Call(
		uintptr(wsExControlParent),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		style,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		0, 0, a.instance, 0,
	)
	if hwnd == 0 {
		fatal("Could not create the main window.\n\n" + callErr.Error())
	}

	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	procSetFocus.Call(a.input)
	a.startFetch()

	os.Exit(a.messageLoop())
}

// wndProc is the single window procedure shared by the main window.
func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	if appPtr != nil {
		if result, handled := appPtr.handleMessage(hwnd, message, wParam, lParam); handled {
			return result
		}
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return result
}

func (a *app) handleMessage(hwnd, message, wParam, lParam uintptr) (uintptr, bool) {
	switch message {
	case wmCreate:
		a.hwnd = hwnd
		a.onCreate()
		return 0, true
	case wmSize:
		a.layout()
		return 0, true
	case wmCommand:
		if wParam&0xffff == idFetchButton && wParam>>16 == bnClicked {
			a.startFetch()
			return 0, true
		}
	case wmFetchDone:
		a.finishFetch()
		return 0, true
	case wmCtlColorStatic:
		// Read-only edits send WM_CTLCOLORSTATIC (not ...EDIT), so keep the
		// output box white and only make the labels transparent.
		if control, _, _ := procWindowFromDC.Call(lParam); a.output != 0 && control == a.output {
			return brush(colorWindow), true
		}
		procSetBkMode.Call(lParam, transparent)
		return brush(colorBtnFace), true
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0, true
	case wmDestroy:
		if a.fontUI != 0 {
			procDeleteObject.Call(a.fontUI)
		}
		if a.fontOutput != 0 {
			procDeleteObject.Call(a.fontOutput)
		}
		procPostQuitMessage.Call(0)
		return 0, true
	}
	return 0, false
}

func (a *app) onCreate() {
	a.fontUI = a.createFont(-a.px(12), "Segoe UI")
	a.fontOutput = a.createFont(-a.px(12), "Consolas")

	a.label = a.createControl("STATIC", "City:", wsChild|wsVisible, 0, idLocationLabel)
	a.input = a.createControl("EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, wsExClientEdge, idLocationEdit)
	a.button = a.createControl("BUTTON", "Get Weather", wsChild|wsVisible|wsTabStop|bsDefPushButton, 0, idFetchButton)
	a.output = a.createControl("EDIT", "", wsChild|wsVisible|wsVScroll|esMultiline|esAutoVScroll|esWantReturn|esReadOnly|wsBorder, wsExClientEdge, idOutputEdit)
	a.status = a.createControl("STATIC", "Ready", wsChild|wsVisible, 0, idStatusLabel)

	a.setFont(a.label, a.fontUI)
	a.setFont(a.input, a.fontUI)
	a.setFont(a.button, a.fontUI)
	a.setFont(a.output, a.fontOutput)
	a.setFont(a.status, a.fontUI)

	setText(a.input, "Tehran")
	a.layout()
}

func (a *app) layout() {
	if a.hwnd == 0 || a.output == 0 {
		return
	}

	var client rect
	procGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client)))

	width := int(client.right - client.left)
	height := int(client.bottom - client.top)

	margin := a.px(16)
	labelWidth := a.px(32)
	buttonWidth := a.px(120)
	rowHeight := a.px(28)
	gap := a.px(10)
	statusHeight := a.px(20)

	a.move(a.label, margin, margin+a.px(6), labelWidth, rowHeight)

	inputX := margin + labelWidth + a.px(6)
	buttonX := width - margin - buttonWidth
	inputWidth := buttonX - a.px(8) - inputX
	if inputWidth < a.px(80) {
		inputWidth = a.px(80)
	}
	a.move(a.input, inputX, margin, inputWidth, rowHeight)
	a.move(a.button, buttonX, margin, buttonWidth, rowHeight)

	outputY := margin + rowHeight + gap
	outputHeight := height - outputY - statusHeight - margin - a.px(4)
	if outputHeight < a.px(80) {
		outputHeight = a.px(80)
	}
	a.move(a.output, margin, outputY, width-2*margin, outputHeight)
	a.move(a.status, margin, height-statusHeight, width-2*margin, statusHeight)
}

func (a *app) startFetch() {
	if a.fetching {
		return
	}

	location := strings.TrimSpace(getText(a.input))
	if location == "" {
		location = weather.DefaultLocation
		setText(a.input, location)
	}

	a.fetching = true
	procEnableWindow.Call(a.button, 0)
	setText(a.status, "Fetching weather for "+location+"…")

	go func() {
		result := fetchResult{location: location}
		forecast, err := a.client.FetchForecast(context.Background(), location)
		if err != nil {
			result.err = err
		} else {
			result.text = forecast.Summary()
		}

		a.mu.Lock()
		a.result = result
		a.mu.Unlock()

		procPostMessageW.Call(a.hwnd, wmFetchDone, 0, 0)
	}()
}

func (a *app) finishFetch() {
	a.mu.Lock()
	result := a.result
	a.mu.Unlock()

	a.fetching = false
	procEnableWindow.Call(a.button, 1)

	if result.err != nil {
		setText(a.output, toCRLF(fmt.Sprintf(
			"Could not fetch weather for %q.\n\n%v\n\nCheck your internet connection and try again. The service used is https://wttr.in.",
			result.location, result.err,
		)))
		setText(a.status, "Request failed")
		return
	}

	setText(a.output, toCRLF(result.text))
	setText(a.hwnd, appName+" - "+result.location)
	setText(a.status, "Updated at "+time.Now().Format("15:04:05"))
}

func (a *app) messageLoop() int {
	var message msg
	for {
		// Blocks until a message arrives; keep the struct alive for the whole
		// iteration because TranslateMessage/DispatchMessage read from it.
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		runtime.KeepAlive(&message)

		if int32(ret) == -1 {
			return 1
		}
		if ret == 0 {
			return int(message.wParam)
		}

		if handled, _, _ := procIsDialogMessageW.Call(a.hwnd, uintptr(unsafe.Pointer(&message))); handled != 0 {
			continue
		}

		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func (a *app) createControl(className, text string, style, exStyle uintptr, id int) uintptr {
	classPtr, _ := syscall.UTF16PtrFromString(className)
	textPtr, _ := syscall.UTF16PtrFromString(text)

	hwnd, _, _ := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(textPtr)),
		style,
		0, 0, 0, 0,
		a.hwnd,
		uintptr(id),
		a.instance,
		0,
	)
	return hwnd
}

func (a *app) setFont(hwnd, font uintptr) {
	if hwnd == 0 || font == 0 {
		return
	}
	const wmSetFont = 0x0030
	procSendMessageW.Call(hwnd, wmSetFont, font, 1)
}

func (a *app) createFont(height int, face string) uintptr {
	facePtr, _ := syscall.UTF16PtrFromString(face)
	font, _, _ := procCreateFontW.Call(
		uintptr(int32(height)), 0, 0, 0,
		fwNormal, 0, 0, 0,
		defaultCharset, 0, 0,
		clearTypeQuality, 0,
		uintptr(unsafe.Pointer(facePtr)),
	)
	return font
}

func (a *app) move(hwnd uintptr, x, y, width, height int) {
	if hwnd == 0 {
		return
	}
	procMoveWindow.Call(hwnd, uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(width)), uintptr(int32(height)), 1)
}

func (a *app) px(value int) int {
	if a.scale <= 0 {
		return value
	}
	return int(float64(value)*a.scale + 0.5)
}

func setText(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(textPtr)))
	runtime.KeepAlive(textPtr)
}

func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return ""
	}
	buffer := make([]uint16, int(length)+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer)
}

func toCRLF(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\n", "\r\n")
}

func brush(color int) uintptr {
	handle, _, _ := procGetSysColorBrush.Call(uintptr(color))
	return handle
}

func loadIcon() uintptr {
	icon, _, _ := procLoadIconW.Call(0, uintptr(idiApplication))
	return icon
}

func loadCursor() uintptr {
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(idcArrow))
	return cursor
}

func systemScale() float64 {
	const defaultDPI = 96.0

	if procGetDpiForSystem := user32.NewProc("GetDpiForSystem"); procGetDpiForSystem.Find() == nil {
		if dpi, _, _ := procGetDpiForSystem.Call(); dpi >= 96 {
			return float64(dpi) / defaultDPI
		}
	}
	return 1
}

func fatal(message string) {
	title, _ := syscall.UTF16PtrFromString(appName)
	text, _ := syscall.UTF16PtrFromString(message)
	const mbIconError = 0x00000010
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbIconError)
	os.Exit(1)
}
