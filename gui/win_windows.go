//go:build windows

package main

import "syscall"

// Window messages.
const (
	wmCreate         = 0x0001
	wmDestroy        = 0x0002
	wmSize           = 0x0005
	wmPaint          = 0x000F
	wmClose          = 0x0010
	wmEraseBkgnd     = 0x0014
	wmSetCursor      = 0x0020
	wmGetMinMaxInfo  = 0x0024
	wmKeyDown        = 0x0100
	wmChar           = 0x0102
	wmTimer          = 0x0113
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmMouseLeave     = 0x02A3
	wmCtlColorEdit   = 0x0133
	wmCtlColorStatic = 0x0138
	wmNCCalcSize     = 0x0083
	wmNCButtonDown   = 0x00A1
	wmAppFetchDone   = 0x8000 + 1
	wmSetFont        = 0x0030
)

// Window styles and flags.
const (
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabStop          = 0x00010000
	wsPopup            = 0x80000000
	wsThickFrame       = 0x00040000
	wsSysMenu          = 0x00080000
	wsMinimizeBox      = 0x00020000
	wsExAppWindow      = 0x00040000
	wsExLayered        = 0x00080000
	wsExControlParent  = 0x00010000
	esAutoHScroll      = 0x0080
	csHRedraw          = 0x0002
	csVRedraw          = 0x0001
	csDropShadow       = 0x00020000
	lwaAlpha           = 0x00000002
	swpNoSize          = 0x0001
	swpNoMove          = 0x0002
	swpNoActivate      = 0x0010
	swShow             = 5
	swMinimize         = 6
	spiGetWorkArea     = 0x0030
	tmeLeave           = 0x00000002
	htCaption          = 2
	htClient           = 1
	gwlpWndProc        = ^uintptr(3)
	idcArrow           = 32512
	idcHand            = 32649
	idiApplication     = 32512
	emSetSel           = 0x00B1
	emSetCueBanner     = 0x1501
	lbGetText          = 0x0189 // unused, keeps constants grouped
	defaultCharset     = 1
	cleartypeQuality   = 5
	fwNormal           = 400
	fwSemibold         = 600
	transparentBkMode  = 1
	dtLeft             = 0x0000
	dtCenter           = 0x0001
	dtRight            = 0x0002
	dtVCenter          = 0x0004
	dtSingleLine       = 0x0020
	dtEndEllipsis      = 0x8000
	dtNoPrefix         = 0x0800
	psSolid            = 0
	nullBrush          = 5
	nullPen            = 8
	srcCopy            = 0x00CC0020
	dibRGBColors       = 0
	biRGB              = 0
	gradientFillRectV  = 1
	acSrcAlpha         = 1
	swpShowWindow      = 0x0040
	dwmwaRoundedCorner = 33
	dwmwaDarkMode      = 20
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

type size struct {
	cx int32
	cy int32
}

type paintStruct struct {
	hdc       uintptr
	erase     int32
	paint     rect
	restore   int32
	incUpdate int32
	reserved  [32]byte
}

type minMaxInfo struct {
	reserved     point
	maxSize      point
	maxPosition  point
	minTrackSize point
	maxTrackSize point
}

type trackingMouseEvent struct {
	size      uint32
	flags     uint32
	hwnd      uintptr
	hoverTime uint32
}

type triVertex struct {
	x     int32
	y     int32
	red   uint16
	green uint16
	blue  uint16
	alpha uint16
}

type gradientRect struct {
	upperLeft  uint32
	lowerRight uint32
}

type bitmapInfoHeader struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

type bitmapInfo struct {
	header bitmapInfoHeader
	colors [1]uint32
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	msimg32  = syscall.NewLazyDLL("msimg32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	pRegisterClassExW       = user32.NewProc("RegisterClassExW")
	pCreateWindowExW        = user32.NewProc("CreateWindowExW")
	pDefWindowProcW         = user32.NewProc("DefWindowProcW")
	pDestroyWindow          = user32.NewProc("DestroyWindow")
	pPostQuitMessage        = user32.NewProc("PostQuitMessage")
	pGetMessageW            = user32.NewProc("GetMessageW")
	pTranslateMessage       = user32.NewProc("TranslateMessage")
	pDispatchMessageW       = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW       = user32.NewProc("IsDialogMessageW")
	pPostMessageW           = user32.NewProc("PostMessageW")
	pSendMessageW           = user32.NewProc("SendMessageW")
	pSetWindowTextW         = user32.NewProc("SetWindowTextW")
	pGetWindowTextW         = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW   = user32.NewProc("GetWindowTextLengthW")
	pSetFocus               = user32.NewProc("SetFocus")
	pGetFocus               = user32.NewProc("GetFocus")
	pMoveWindow             = user32.NewProc("MoveWindow")
	pGetClientRect          = user32.NewProc("GetClientRect")
	pShowWindow             = user32.NewProc("ShowWindow")
	pUpdateWindow           = user32.NewProc("UpdateWindow")
	pInvalidateRect         = user32.NewProc("InvalidateRect")
	pBeginPaint             = user32.NewProc("BeginPaint")
	pEndPaint               = user32.NewProc("EndPaint")
	pSetTimer               = user32.NewProc("SetTimer")
	pKillTimer              = user32.NewProc("KillTimer")
	pLoadCursorW            = user32.NewProc("LoadCursorW")
	pLoadIconW              = user32.NewProc("LoadIconW")
	pMessageBoxW            = user32.NewProc("MessageBoxW")
	pSetProcessDPIAware     = user32.NewProc("SetProcessDPIAware")
	pSetLayeredWindowAttr   = user32.NewProc("SetLayeredWindowAttributes")
	pSetWindowPos           = user32.NewProc("SetWindowPos")
	pSetWindowRgn           = user32.NewProc("SetWindowRgn")
	pGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	pSystemParametersInfoW  = user32.NewProc("SystemParametersInfoW")
	pSetCapture             = user32.NewProc("SetCapture")
	pReleaseCapture         = user32.NewProc("ReleaseCapture")
	pTrackMouseEvent        = user32.NewProc("TrackMouseEvent")
	pWindowFromDC           = user32.NewProc("WindowFromDC")
	pCallWindowProcW        = user32.NewProc("CallWindowProcW")
	pGetDC                  = user32.NewProc("GetDC")
	pReleaseDC              = user32.NewProc("ReleaseDC")
	pFillRect               = user32.NewProc("FillRect")
	pGetCursorPos           = user32.NewProc("GetCursorPos")
	pGetKeyState            = user32.NewProc("GetKeyState")
	pScreenToClient         = user32.NewProc("ScreenToClient")
	pSetCursor              = user32.NewProc("SetCursor")
	pSetWindowLong          = user32.NewProc("SetWindowLongPtrW")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkColor             = gdi32.NewProc("SetBkColor")
	pDrawTextW              = user32.NewProc("DrawTextW")
	pGetTextExtentPoint32W  = gdi32.NewProc("GetTextExtentPoint32W")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pEllipse                = gdi32.NewProc("Ellipse")
	pArc                    = gdi32.NewProc("Arc")
	pPolygon                = gdi32.NewProc("Polygon")
	pMoveToEx               = gdi32.NewProc("MoveToEx")
	pLineTo                 = gdi32.NewProc("LineTo")
	pCreateRoundRectRgn     = gdi32.NewProc("CreateRoundRectRgn")
	pFillRgn                = gdi32.NewProc("FillRgn")
	pSelectClipRgn          = gdi32.NewProc("SelectClipRgn")
	pSaveDC                 = gdi32.NewProc("SaveDC")
	pRestoreDC              = gdi32.NewProc("RestoreDC")
	pCreateDIBSection       = gdi32.NewProc("CreateDIBSection")
	pGetStockObject         = gdi32.NewProc("GetStockObject")
	pStretchBlt             = gdi32.NewProc("StretchBlt")
	pSetStretchBltMode      = gdi32.NewProc("SetStretchBltMode")
	pGetDpiForSystem        = user32.NewProc("GetDpiForSystem")
	pDwmSetWindowAttribute  = dwmapi.NewProc("DwmSetWindowAttribute")
	pGradientFill           = msimg32.NewProc("GradientFill")
	pAlphaBlend             = msimg32.NewProc("AlphaBlend")
	pGetModuleHandleW       = kernel32.NewProc("GetModuleHandleW")
)

func init() {
	// SetWindowLongPtrW only exists as a real export on 64-bit Windows.
	if pSetWindowLong.Find() != nil {
		pSetWindowLong = user32.NewProc("SetWindowLongW")
	}
}

const (
	htCaptionValue = htCaption
	halftone       = 4
)
