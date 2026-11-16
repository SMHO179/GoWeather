//go:build windows

// Command weather-gui is a frameless, animated Win32 weather widget.
//
// It uses only the Go standard library (syscall + unsafe): no CGO, no C
// compiler and no third-party modules.
package main

import (
	"context"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/SMHO179/GoWeather/internal/weather"
)

const (
	appName     = "GoWeather"
	windowClass = "GoWeatherWidgetWindow"

	widgetWidth  = 880
	widgetHeight = 524

	timerID  = 1
	tickMS   = 30
	idEdit   = 1001
	roundDIP = 18
)

// Interactive regions. Values are indices into app.hover.
const (
	btnNone = iota
	btnRefresh
	btnPin
	btnMinimize
	btnClose
	btnField
	btnCount
)

const cardBase = 20 // cardBase+i identifies hovered forecast card i

type snapshot struct {
	place    string
	has      bool
	tempC    int
	feelsC   int
	desc     string
	observed string
	humidity string
	wind     string
	windDir  string
	pressure string
	vis      string
	uv       string
	sunrise  string
	sunset   string
	highC    int
	lowC     int
	days     []daySnapshot
	icon     iconKind
	when     time.Time
}

type daySnapshot struct {
	label string
	desc  string
	minC  int
	maxC  int
	icon  iconKind
}

type fetchOutcome struct {
	snap snapshot
	err  error
}

type widgetLayout struct {
	search  rect
	refresh rect
	pin     rect
	min     rect
	close   rect
	hero    rect
	cards   [3]rect
	footer  rect
}

type fontSet struct {
	title    uintptr
	small    uintptr
	place    uintptr
	desc     uintptr
	temp     uintptr
	label    uintptr
	stat     uintptr
	statLbl  uintptr
	cardDay  uintptr
	cardDesc uintptr
	cardMax  uintptr
	cardMin  uintptr
	button   uintptr
}

type app struct {
	hwnd     uintptr
	instance uintptr
	scale    float64

	edit       uintptr
	fieldBrush uintptr
	fonts      fontSet

	memDC  uintptr
	memBmp uintptr
	memW   int32
	memH   int32

	staticDC    uintptr
	staticBmp   uintptr
	staticW     int32
	staticH     int32
	staticShift int32
	staticDirty bool

	glows map[uint32]*glowTex

	lay widgetLayout

	client *weather.Client

	mu      sync.Mutex
	outcome fetchOutcome

	loading      bool
	pendingPlace string
	errText      string
	snap         snapshot
	hasData      bool

	tempTarget   float64
	tempDisplay  float64
	contentShift float64
	focus        float64
	hover        [btnCount]float64
	hoverCard    [3]float64
	active       int
	pressed      int
	pinned       bool
	tracked      bool

	started time.Time
	alpha   uint8
}

var (
	appPtr           *app
	wndProcCallback  uintptr
	editProcCallback uintptr
	editOldProc      uintptr
)

func main() {
	pSetProcessDPIAware.Call()

	instance, _, _ := pGetModuleHandleW.Call(0)
	if instance == 0 {
		os.Exit(1)
	}
	appPtr = &app{
		instance:     instance,
		scale:        systemScale(),
		client:       weather.NewClient(),
		glows:        map[uint32]*glowTex{},
		pendingPlace: "Tehran",
		started:      time.Now(),
		staticDirty:  true,
	}
	a := appPtr

	wndProcCallback = syscall.NewCallback(wndProc)
	className, _ := syscall.UTF16PtrFromString(windowClass)
	windowTitle, _ := syscall.UTF16PtrFromString(appName)

	class := wndClassEx{
		style:      csVRedraw | csHRedraw | csDropShadow,
		wndProc:    wndProcCallback,
		instance:   a.instance,
		icon:       loadIcon(),
		cursor:     loadCursor(),
		background: 0,
		className:  className,
	}
	class.size = uint32(unsafe.Sizeof(class))

	if atom, _, callErr := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		fatal("Could not register the window class.\n\n" + callErr.Error())
	}

	width := a.px(widgetWidth)
	height := a.px(widgetHeight)
	x, y := placeWindow(width, height)

	style := uintptr(wsPopup | wsThickFrame | wsSysMenu | wsMinimizeBox)
	exStyle := uintptr(wsExAppWindow | wsExLayered | wsExControlParent)
	hwnd, _, callErr := pCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		style,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		0, 0, a.instance, 0,
	)
	if hwnd == 0 {
		fatal("Could not create the main window.\n\n" + callErr.Error())
	}

	radius := a.px(roundDIP)
	if region, _, _ := pCreateRoundRectRgn.Call(0, 0, uintptr(width+1), uintptr(height+1), uintptr(radius*2), uintptr(radius*2)); region != 0 {
		pSetWindowRgn.Call(hwnd, region, 1)
	}
	a.applyDWM()

	pSetLayeredWindowAttr.Call(hwnd, 0, 0, lwaAlpha)
	pShowWindow.Call(hwnd, swShow)
	pUpdateWindow.Call(hwnd)
	pSetFocus.Call(a.edit)
	a.startFetch()

	os.Exit(a.messageLoop())
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	if appPtr != nil {
		if result, handled := appPtr.handleMessage(hwnd, message, wParam, lParam); handled {
			return result
		}
	}
	result, _, _ := pDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return result
}

func (a *app) handleMessage(hwnd, message, wParam, lParam uintptr) (uintptr, bool) {
	switch message {
	case wmCreate:
		a.hwnd = hwnd
		a.onCreate()
		return 0, true
	case wmSize:
		if wParam == 1 { // SIZE_MINIMIZED
			pKillTimer.Call(hwnd, timerID)
		} else {
			pSetTimer.Call(hwnd, timerID, tickMS, 0)
			a.layout()
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0, true
	case wmTimer:
		a.tick()
		return 0, true
	case wmPaint:
		a.onPaint()
		return 0, true
	case wmEraseBkgnd:
		return 1, true
	case wmCtlColorEdit:
		if a.fieldBrush != 0 {
			pSetBkColor.Call(wParam, uintptr(colField))
			pSetTextColor.Call(wParam, uintptr(colText))
			return a.fieldBrush, true
		}
	case wmSetCursor:
		if uintptr(uint16(lParam)) == htClient {
			if id := a.cursorTarget(); id == btnRefresh || id == btnPin || id == btnMinimize || id == btnClose || id == btnField {
				if hand := loadHandCursor(); hand != 0 {
					pSetCursor.Call(hand)
					return 1, true
				}
			}
		}
	case wmMouseMove:
		x, y := mousePoint(lParam)
		a.setActive(a.hitTest(x, y))
		if !a.tracked {
			track := trackingMouseEvent{size: uint32(unsafe.Sizeof(trackingMouseEvent{})), flags: tmeLeave, hwnd: hwnd}
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&track)))
			a.tracked = true
		}
		return 0, true
	case wmLButtonDown:
		x, y := mousePoint(lParam)
		id := a.hitTest(x, y)
		a.setActive(id)
		switch id {
		case btnRefresh, btnPin, btnMinimize, btnClose:
			a.pressed = id
			pSetCapture.Call(hwnd)
			return 0, true
		case btnField:
			pSetFocus.Call(a.edit)
			pSendMessageW.Call(a.edit, emSetSel, 0, ^uintptr(0))
			return 0, true
		}
		a.beginDrag()
		return 0, true
	case wmLButtonUp:
		x, y := mousePoint(lParam)
		id := a.hitTest(x, y)
		pressed := a.pressed
		a.pressed = 0
		pReleaseCapture.Call()
		if pressed != 0 && pressed == id {
			a.activate(pressed)
		}
		return 0, true
	case wmMouseLeave:
		a.tracked = false
		a.setActive(btnNone)
		return 0, true
	case wmKeyDown:
		if wParam == 0x74 || (wParam == 'R' && isKeyDown(0x11)) { // F5 / Ctrl+R
			a.startFetch()
			return 0, true
		}
	case wmNCCalcSize:
		if wParam != 0 {
			return 0, true // client area covers the whole window
		}
	case wmClose:
		pDestroyWindow.Call(hwnd)
		return 0, true
	case wmAppFetchDone:
		a.finishFetch()
		return 0, true
	case wmDestroy:
		a.cleanup()
		pPostQuitMessage.Call(0)
		return 0, true
	}
	return 0, false
}

func (a *app) onCreate() {
	a.fonts = fontSet{
		title:    a.createFont(13, "Segoe UI Semibold", fwSemibold),
		small:    a.createFont(10, "Segoe UI", fwNormal),
		place:    a.createFont(19, "Segoe UI Semibold", fwSemibold),
		desc:     a.createFont(13, "Segoe UI", fwNormal),
		temp:     a.createFont(66, "Segoe UI Light", fwNormal),
		label:    a.createFont(11, "Segoe UI", fwNormal),
		stat:     a.createFont(13, "Segoe UI Semibold", fwSemibold),
		statLbl:  a.createFont(9, "Segoe UI", fwNormal),
		cardDay:  a.createFont(12, "Segoe UI Semibold", fwSemibold),
		cardDesc: a.createFont(10, "Segoe UI", fwNormal),
		cardMax:  a.createFont(16, "Segoe UI Semibold", fwSemibold),
		cardMin:  a.createFont(10, "Segoe UI", fwNormal),
		button:   a.createFont(12, "Segoe UI Semibold", fwSemibold),
	}

	a.edit = a.createEdit()
	a.fieldBrush, _, _ = pCreateSolidBrush.Call(uintptr(colField))

	editProcCallback = syscall.NewCallback(editProc)
	editOldProc, _, _ = pSetWindowLong.Call(a.edit, gwlpWndProc, editProcCallback)

	setText(a.edit, "Tehran")
	if cue, err := syscall.UTF16PtrFromString("Search city…"); err == nil {
		pSendMessageW.Call(a.edit, emSetCueBanner, 1, uintptr(unsafe.Pointer(cue)))
		runtime.KeepAlive(cue)
	}

	pSetTimer.Call(a.hwnd, timerID, tickMS, 0)
	a.layout()
}

func editProc(hwnd, message, wParam, lParam uintptr) uintptr {
	if appPtr != nil {
		switch message {
		case wmKeyDown:
			switch {
			case wParam == 0x0D: // Enter
				appPtr.startFetch()
				return 0
			case wParam == 0x1B: // Escape
				pSetFocus.Call(appPtr.hwnd)
				return 0
			case wParam == 0x74 || (wParam == 'R' && isKeyDown(0x11)):
				appPtr.startFetch()
				return 0
			}
		case wmChar:
			if wParam == 0x0D {
				return 0
			}
		}
	}
	result, _, _ := pCallWindowProcW.Call(editOldProc, hwnd, message, wParam, lParam)
	return result
}

func (a *app) createEdit() uintptr {
	classPtr, _ := syscall.UTF16PtrFromString("EDIT")
	root, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(classPtr)),
		0,
		uintptr(wsChild|wsVisible|wsTabStop|esAutoHScroll),
		0, 0, 0, 0,
		a.hwnd, idEdit, a.instance, 0,
	)
	if root != 0 {
		pSendMessageW.Call(root, wmSetFont, a.fonts.desc, 1)
	}
	return root
}

func (a *app) layout() {
	if a.hwnd == 0 {
		return
	}
	a.staticDirty = true
	var client rect
	pGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client)))
	w, h := client.right, client.bottom
	pad := int32(a.px(24))

	// Window controls sit in the header, right aligned.
	btn := int32(a.px(30))
	gap := int32(a.px(8))
	top := int32(a.px(20))
	a.lay.close = rect{right: w - pad, top: top, bottom: top + btn}
	a.lay.close.left = a.lay.close.right - btn
	a.lay.min = rect{left: a.lay.close.left - gap - btn, right: a.lay.close.left - gap, top: top, bottom: top + btn}
	a.lay.pin = rect{left: a.lay.min.left - gap - btn, right: a.lay.min.left - gap, top: top, bottom: top + btn}

	// Search field and update button.
	sy := int32(a.px(76))
	sh := int32(a.px(44))
	refreshW := int32(a.px(126))
	a.lay.refresh = rect{left: w - pad - refreshW, right: w - pad, top: sy, bottom: sy + sh}
	a.lay.search = rect{left: pad, right: a.lay.refresh.left - int32(a.px(12)), top: sy, bottom: sy + sh}

	// Hero card and three forecast cards.
	cy := int32(a.px(136))
	contentH := int32(a.px(316))
	heroW := int32(a.px(404))
	a.lay.hero = rect{left: pad, top: cy, right: pad + heroW, bottom: cy + contentH}

	rightX := a.lay.hero.right + int32(a.px(16))
	rightW := w - pad - rightX
	cardGap := int32(a.px(12))
	cardH := (contentH - 2*cardGap) / 3
	for i := 0; i < 3; i++ {
		y := cy + int32(i)*(cardH+cardGap)
		a.lay.cards[i] = rect{left: rightX, right: rightX + rightW, top: y, bottom: y + cardH}
	}

	fy := h - int32(a.px(50))
	a.lay.footer = rect{left: pad, right: w - pad, top: fy, bottom: fy + int32(a.px(22))}

	if a.edit != 0 {
		eh := int32(a.px(24))
		ex := a.lay.search.left + int32(a.px(44))
		ew := a.lay.search.right - ex - int32(a.px(16))
		ey := a.lay.search.top + (sh-eh)/2
		pMoveWindow.Call(a.edit, uintptr(ex), uintptr(ey), uintptr(ew), uintptr(eh), 1)
	}
}

func (a *app) startFetch() {
	if a.loading {
		return
	}

	location := strings.TrimSpace(getText(a.edit))
	if location == "" {
		location = "Tehran"
		setText(a.edit, location)
	}

	a.loading = true
	a.errText = ""
	a.contentShift = 0
	a.staticDirty = true
	if !a.hasData {
		a.pendingPlace = location
	}
	pInvalidateRect.Call(a.hwnd, 0, 0)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		outcome := fetchOutcome{}
		forecast, err := a.client.FetchForecast(ctx, location)
		if err != nil {
			outcome.err = err
		} else {
			outcome.snap = snapshotFrom(forecast)
		}

		a.mu.Lock()
		a.outcome = outcome
		a.mu.Unlock()
		pPostMessageW.Call(a.hwnd, wmAppFetchDone, 0, 0)
	}()
}

func (a *app) finishFetch() {
	a.mu.Lock()
	outcome := a.outcome
	a.mu.Unlock()

	a.loading = false
	if outcome.err != nil {
		a.errText = friendlyError(outcome.err)
		a.staticDirty = true
		pInvalidateRect.Call(a.hwnd, 0, 0)
		return
	}

	first := !a.hasData
	a.snap = outcome.snap
	a.hasData = true
	a.errText = ""
	a.staticDirty = true
	a.tempTarget = float64(a.snap.tempC)
	if first {
		a.tempDisplay = 0
	}
	a.contentShift = 14
	setText(a.hwnd, appName+" · "+a.snap.place)
	pInvalidateRect.Call(a.hwnd, 0, 0)
}

func (a *app) activate(id int) {
	switch id {
	case btnRefresh:
		a.startFetch()
	case btnPin:
		a.pinned = !a.pinned
		target := ^uintptr(1) // HWND_NOTOPMOST (-2)
		if a.pinned {
			target = ^uintptr(0) // HWND_TOPMOST (-1)
		}
		pSetWindowPos.Call(a.hwnd, target, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
	case btnMinimize:
		pShowWindow.Call(a.hwnd, swMinimize)
	case btnClose:
		pPostMessageW.Call(a.hwnd, wmClose, 0, 0)
	}
}

func (a *app) beginDrag() {
	pReleaseCapture.Call()
	pSendMessageW.Call(a.hwnd, wmNCButtonDown, htCaption, 0)
}

func (a *app) setActive(id int) {
	if a.active != id {
		a.active = id
	}
}

func (a *app) hitTest(x, y int32) int {
	l := a.lay
	in := func(r rect) bool { return x >= r.left && x < r.right && y >= r.top && y < r.bottom }

	switch {
	case in(l.close):
		return btnClose
	case in(l.min):
		return btnMinimize
	case in(l.pin):
		return btnPin
	case in(l.refresh):
		return btnRefresh
	case in(l.search):
		return btnField
	}
	for i := 0; i < 3; i++ {
		if in(l.cards[i]) {
			return cardBase + i
		}
	}
	return btnNone
}

func (a *app) cursorTarget() int {
	var p point
	if ok, _, _ := pGetCursorPos.Call(uintptr(unsafe.Pointer(&p))); ok == 0 {
		return btnNone
	}
	pScreenToClient.Call(a.hwnd, uintptr(unsafe.Pointer(&p)))
	return a.hitTest(p.x, p.y)
}

func (a *app) tick() {
	if a.alpha < 255 {
		next := int(a.alpha) + 18
		if next > 255 {
			next = 255
		}
		a.alpha = uint8(next)
		pSetLayeredWindowAttr.Call(a.hwnd, 0, uintptr(a.alpha), lwaAlpha)
	}

	for i := 0; i < btnCount; i++ {
		target := 0.0
		if a.active == i {
			target = 1
		}
		a.hover[i] += (target - a.hover[i]) * 0.22
	}
	for i := 0; i < 3; i++ {
		target := 0.0
		if a.active == cardBase+i {
			target = 1
		}
		a.hoverCard[i] += (target - a.hoverCard[i]) * 0.18
	}

	focusTarget := 0.0
	if f, _, _ := pGetFocus.Call(); f == a.edit {
		focusTarget = 1
	}
	a.focus += (focusTarget - a.focus) * 0.2

	a.tempDisplay += (a.tempTarget - a.tempDisplay) * 0.12
	if math.Abs(a.tempTarget-a.tempDisplay) < 0.06 {
		a.tempDisplay = a.tempTarget
	}
	if a.contentShift > 0.2 {
		a.contentShift *= 0.8
		a.staticDirty = true
	} else {
		a.contentShift = 0
	}

	pInvalidateRect.Call(a.hwnd, 0, 0)
}

func (a *app) messageLoop() int {
	var message msg
	for {
		ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		runtime.KeepAlive(&message)

		if int32(ret) == -1 {
			return 1
		}
		if ret == 0 {
			return int(message.wParam)
		}

		if handled, _, _ := pIsDialogMessageW.Call(a.hwnd, uintptr(unsafe.Pointer(&message))); handled != 0 {
			continue
		}

		pTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func (a *app) cleanup() {
	pKillTimer.Call(a.hwnd, timerID)
	a.releaseStatic()
	if a.memDC != 0 {
		pDeleteDC.Call(a.memDC)
		a.memDC = 0
	}
	if a.memBmp != 0 {
		pDeleteObject.Call(a.memBmp)
		a.memBmp = 0
	}
	for _, tex := range a.glows {
		tex.release()
	}
	a.glows = map[uint32]*glowTex{}

	fonts := []uintptr{
		a.fonts.title, a.fonts.small, a.fonts.place, a.fonts.desc, a.fonts.temp,
		a.fonts.label, a.fonts.stat, a.fonts.statLbl, a.fonts.cardDay,
		a.fonts.cardDesc, a.fonts.cardMax, a.fonts.cardMin, a.fonts.button,
	}
	for _, font := range fonts {
		if font != 0 {
			pDeleteObject.Call(font)
		}
	}
	if a.fieldBrush != 0 {
		pDeleteObject.Call(a.fieldBrush)
	}
}

func (a *app) createFont(height int, face string, weight int) uintptr {
	facePtr, _ := syscall.UTF16PtrFromString(face)
	font, _, _ := pCreateFontW.Call(
		uintptr(int32(-a.px(height))), 0, 0, 0,
		uintptr(weight), 0, 0, 0,
		defaultCharset, 0, 0,
		cleartypeQuality, 0,
		uintptr(unsafe.Pointer(facePtr)),
	)
	return font
}

func (a *app) px(value int) int {
	if a.scale <= 0 {
		return value
	}
	return int(float64(value)*a.scale + 0.5)
}

func (a *app) applyDWM() {
	dark := int32(1)
	pDwmSetWindowAttribute.Call(a.hwnd, dwmwaDarkMode, uintptr(unsafe.Pointer(&dark)), unsafe.Sizeof(dark))
	round := int32(2)
	pDwmSetWindowAttribute.Call(a.hwnd, dwmwaRoundedCorner, uintptr(unsafe.Pointer(&round)), unsafe.Sizeof(round))
}

func (a *app) onPaint() {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}

	var client rect
	pGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client)))
	w, h := client.right, client.bottom

	if a.memDC == 0 || a.memW != w || a.memH != h {
		if a.memBmp != 0 {
			pDeleteObject.Call(a.memBmp)
			a.memBmp = 0
		}
		if a.memDC != 0 {
			pDeleteDC.Call(a.memDC)
			a.memDC = 0
		}
		a.memDC, _, _ = pCreateCompatibleDC.Call(hdc)
		a.memBmp, _, _ = pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
		if a.memDC != 0 && a.memBmp != 0 {
			pSelectObject.Call(a.memDC, a.memBmp)
		}
		a.memW, a.memH = w, h
	}

	if a.memDC != 0 && a.memBmp != 0 {
		a.render(a.memDC, w, h)
		pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), a.memDC, 0, 0, srcCopy)
	}

	pEndPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))
}

func snapshotFrom(f *weather.Forecast) snapshot {
	s := snapshot{
		place:    f.Place(),
		has:      true,
		humidity: "—",
		wind:     "—",
		windDir:  "",
		pressure: "—",
		vis:      "—",
		uv:       "—",
		observed: "",
		when:     time.Now(),
	}

	if len(f.CurrentCondition) > 0 {
		c := f.CurrentCondition[0]
		s.tempC = atoiOr(c.TempC, 0)
		s.feelsC = atoiOr(c.FeelsLikeC, 0)
		s.desc = c.Description()
		s.humidity = textOr(c.Humidity)
		s.wind = textOr(c.WindKmph)
		s.windDir = strings.TrimSpace(c.WindDir)
		s.pressure = textOr(c.Pressure)
		s.vis = textOr(c.Visibility)
		s.uv = textOr(c.UVIndex)
		s.observed = strings.TrimSpace(c.LocalObsDateTime)
		if s.observed == "" {
			s.observed = strings.TrimSpace(c.ObservationTime)
		}
		s.icon = iconFromDescription(s.desc)
	}

	for i, day := range f.Weather {
		if i >= 3 {
			break
		}
		s.days = append(s.days, daySnapshot{
			label: dayLabel(day.Date, i),
			desc:  day.Description(),
			minC:  atoiOr(day.MinTempC, 0),
			maxC:  atoiOr(day.MaxTempC, 0),
			icon:  iconFromDescription(day.Description()),
		})
	}
	if len(s.days) > 0 {
		s.highC = s.days[0].maxC
		s.lowC = s.days[0].minC
	}
	if len(f.Weather) > 0 && len(f.Weather[0].Astronomy) > 0 {
		a := f.Weather[0].Astronomy[0]
		s.sunrise = strings.TrimSpace(a.Sunrise)
		s.sunset = strings.TrimSpace(a.Sunset)
	}
	return s
}

func dayLabel(date string, index int) string {
	if index == 0 {
		return "Today"
	}
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return parsed.Format("Mon 02 Jan")
}

func atoiOr(value string, fallback int) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return number
}

func textOr(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "—"
	}
	return value
}

func friendlyError(err error) string {
	text := err.Error()
	text = strings.TrimPrefix(text, "failed to connect to weather service: ")
	text = strings.TrimPrefix(text, "failed to build request: ")
	text = strings.TrimPrefix(text, "failed to read response: ")
	if text == "" {
		return "Request failed"
	}
	return "Update failed · " + text
}

func (a *app) placeholderSnapshot() snapshot {
	place := a.pendingPlace
	if strings.TrimSpace(place) == "" {
		place = "Tehran"
	}
	desc := "Fetching current conditions…"
	if a.errText != "" {
		desc = "Weather unavailable"
	}
	return snapshot{
		place:    place,
		desc:     desc,
		humidity: "—",
		wind:     "—",
		pressure: "—",
		vis:      "—",
		uv:       "—",
		sunrise:  "—",
		sunset:   "—",
		icon:     iconCloud,
	}
}

func (a *app) view() snapshot {
	if a.hasData {
		return a.snap
	}
	return a.placeholderSnapshot()
}

func mousePoint(lParam uintptr) (int32, int32) {
	return int32(int16(uint16(lParam))), int32(int16(uint16(lParam >> 16)))
}

func isKeyDown(vk uintptr) bool {
	state, _, _ := pGetKeyState.Call(vk)
	return int16(state) < 0
}

func systemScale() float64 {
	const defaultDPI = 96.0
	if pGetDpiForSystem.Find() == nil {
		if dpi, _, _ := pGetDpiForSystem.Call(); dpi >= 96 {
			return float64(dpi) / defaultDPI
		}
	}
	return 1
}

func placeWindow(width, height int) (int, int) {
	work := rect{right: int32(1280), bottom: int32(720)}
	pSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&work)), 0)
	x := int(work.right) - width - 32
	y := int(work.bottom) - height - 32
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

func setText(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	pSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(textPtr)))
	runtime.KeepAlive(textPtr)
}

func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	length, _, _ := pGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return ""
	}
	buffer := make([]uint16, int(length)+1)
	pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer)
}

func loadIcon() uintptr {
	icon, _, _ := pLoadIconW.Call(0, idiApplication)
	return icon
}

func loadCursor() uintptr {
	cursor, _, _ := pLoadCursorW.Call(0, idcArrow)
	return cursor
}

func loadHandCursor() uintptr {
	cursor, _, _ := pLoadCursorW.Call(0, idcHand)
	return cursor
}

func fatal(message string) {
	title, _ := syscall.UTF16PtrFromString(appName)
	text, _ := syscall.UTF16PtrFromString(message)
	const mbIconError = 0x00000010
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbIconError)
	os.Exit(1)
}
