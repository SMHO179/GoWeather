//go:build windows

package main

import (
	"math"
	"strings"
	"unsafe"
)

type iconKind int

const (
	iconSun iconKind = iota
	iconCloud
	iconRain
	iconSnow
	iconThunder
	iconFog
)

func iconFromDescription(description string) iconKind {
	d := strings.ToLower(description)
	switch {
	case strings.Contains(d, "thunder") || strings.Contains(d, "storm") || strings.Contains(d, "lightning"):
		return iconThunder
	case strings.Contains(d, "snow") || strings.Contains(d, "sleet") || strings.Contains(d, "blizzard") || strings.Contains(d, "ice") || strings.Contains(d, "freezing"):
		return iconSnow
	case strings.Contains(d, "rain") || strings.Contains(d, "drizzle") || strings.Contains(d, "shower"):
		return iconRain
	case strings.Contains(d, "fog") || strings.Contains(d, "mist") || strings.Contains(d, "haze"):
		return iconFog
	case strings.Contains(d, "cloud") || strings.Contains(d, "overcast"):
		return iconCloud
	default:
		return iconSun
	}
}

func drawWeatherIcon(dc uintptr, kind iconKind, area rect, phase float64) {
	w := float64(area.right - area.left)
	h := float64(area.bottom - area.top)
	if w <= 0 || h <= 0 {
		return
	}
	size := math.Min(w, h)
	cx := float64(area.left) + w/2
	cy := float64(area.top) + h/2

	switch kind {
	case iconCloud:
		drawCloud(dc, cx, cy, size*0.94, math.Sin(phase*0.9)*size*0.018, colCloud, colCloudTop)
	case iconRain:
		drawCloud(dc, cx, cy-size*0.12, size*0.82, math.Sin(phase*0.9)*size*0.015, colCloud, colCloudTop)
		drawRain(dc, cx, cy, size, phase)
	case iconSnow:
		drawCloud(dc, cx, cy-size*0.12, size*0.82, math.Sin(phase*0.9)*size*0.015, colCloud, colCloudTop)
		drawSnow(dc, cx, cy, size, phase)
	case iconThunder:
		drawCloud(dc, cx, cy-size*0.14, size*0.82, math.Sin(phase*0.9)*size*0.015, colCloud, colCloudTop)
		drawBolt(dc, cx, cy, size, phase)
	case iconFog:
		drawFog(dc, cx, cy, size, phase)
	default:
		drawSun(dc, cx, cy, size, phase)
	}
}

func drawSun(dc uintptr, cx, cy, size, phase float64) {
	radius := size * 0.17
	rotation := phase * 0.35
	width := int(math.Max(2, size*0.024))
	for i := 0; i < 8; i++ {
		angle := rotation + float64(i)*math.Pi/4
		inner := radius * 1.75
		outer := radius * 2.45
		strokeLine(dc,
			cx+math.Cos(angle)*inner, cy+math.Sin(angle)*inner,
			cx+math.Cos(angle)*outer, cy+math.Sin(angle)*outer,
			colWarm, width,
		)
	}
	fillCircle(dc, cx, cy, radius, colWarmSoft)
}

func drawCloud(dc uintptr, cx, cy, size, bob float64, base, top uint32) {
	fillEllipse(dc, cx, cy+size*0.27+bob, size*0.32, size*0.045, colCloudDark)
	fillRoundBounds(dc, cx-size*0.31, cy+size*0.02+bob, cx+size*0.31, cy+size*0.24+bob, size*0.11, base)
	fillCircle(dc, cx-size*0.17, cy+size*0.04+bob, size*0.14, base)
	fillCircle(dc, cx+size*0.01, cy-size*0.05+bob, size*0.19, base)
	fillCircle(dc, cx+size*0.18, cy+size*0.05+bob, size*0.125, base)
	fillCircle(dc, cx+size*0.005, cy-size*0.1+bob, size*0.12, top)
	fillCircle(dc, cx-size*0.13, cy+size*0.0+bob, size*0.075, top)
}

func drawRain(dc uintptr, cx, cy, size, phase float64) {
	width := int(math.Max(2, size*0.022))
	for i := 0; i < 3; i++ {
		progress := math.Mod(phase*1.35+float64(i)*0.34, 1)
		x := cx + (float64(i)-1)*size*0.16
		y := cy + size*0.16 + progress*size*0.26
		length := size * 0.075 * (0.35 + 0.65*math.Sin(math.Pi*progress))
		strokeLine(dc, x, y, x-size*0.015, y+length, colRain, width)
	}
}

func drawSnow(dc uintptr, cx, cy, size, phase float64) {
	width := int(math.Max(1, size*0.012))
	for i := 0; i < 3; i++ {
		progress := math.Mod(phase*0.75+float64(i)*0.37, 1)
		x := cx + (float64(i)-1)*size*0.17 + math.Sin(phase*1.4+float64(i))*size*0.02
		y := cy + size*0.17 + progress*size*0.24
		radius := size * 0.037
		for k := 0; k < 3; k++ {
			angle := progress*2 + float64(k)*math.Pi/3
			strokeLine(dc,
				x-math.Cos(angle)*radius, y-math.Sin(angle)*radius,
				x+math.Cos(angle)*radius, y+math.Sin(angle)*radius,
				colSnow, width,
			)
		}
	}
}

func drawBolt(dc uintptr, cx, cy, size, phase float64) {
	flash := math.Mod(phase, 2.4) < 0.2
	boltColor := mixColor(colWarm, colCardTop, 0.2)
	if flash {
		boltColor = colWarmSoft
	}
	points := []point{
		{x: int32(cx + size*0.03), y: int32(cy + size*0.14)},
		{x: int32(cx - size*0.09), y: int32(cy + size*0.38)},
		{x: int32(cx - size*0.01), y: int32(cy + size*0.38)},
		{x: int32(cx - size*0.08), y: int32(cy + size*0.62)},
		{x: int32(cx + size*0.14), y: int32(cy + size*0.30)},
		{x: int32(cx + size*0.04), y: int32(cy + size*0.30)},
		{x: int32(cx + size*0.11), y: int32(cy + size*0.14)},
	}
	brush, _, _ := pCreateSolidBrush.Call(uintptr(boltColor))
	pen, _, _ := pGetStockObject.Call(nullPen)
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	pPolygon.Call(dc, uintptr(unsafe.Pointer(&points[0])), uintptr(len(points)))
	pSelectObject.Call(dc, oldBrush)
	pSelectObject.Call(dc, oldPen)
	pDeleteObject.Call(brush)
}

func drawFog(dc uintptr, cx, cy, size, phase float64) {
	width := int(math.Max(2, size*0.035))
	widths := [4]float64{0.48, 0.64, 0.42, 0.56}
	for i := 0; i < 4; i++ {
		lineWidth := widths[i] * size
		x := cx - lineWidth/2 + math.Sin(phase*0.8+float64(i)*1.4)*size*0.035
		y := cy + (float64(i)-1.5)*size*0.135
		strokeLine(dc, x, y, x+lineWidth, y, mixColor(colTextDim, colCardTop, 0.15), width)
	}
}

func drawLogo(dc uintptr, area rect) {
	fillRoundGradient(dc, area, int((area.right-area.left)/3), colAccent, colAccent2)
	w := float64(area.right - area.left)
	h := float64(area.bottom - area.top)
	size := math.Min(w, h)
	cx := float64(area.left) + w/2
	cy := float64(area.top) + h/2
	radius := size * 0.15
	width := int(math.Max(1, size*0.055))
	for i := 0; i < 8; i++ {
		angle := float64(i) * math.Pi / 4
		strokeLine(dc,
			cx+math.Cos(angle)*radius*1.7, cy+math.Sin(angle)*radius*1.7,
			cx+math.Cos(angle)*radius*2.45, cy+math.Sin(angle)*radius*2.45,
			colText, width,
		)
	}
	fillCircle(dc, cx, cy, radius, colText)
}

func drawSearchIcon(dc uintptr, cx, cy, size float64, color uint32) {
	radius := size * 0.33
	width := int(math.Max(1, size*0.11))
	strokeCircle(dc, cx-1, cy-1, radius, color, width)
	startX := cx - 1 + radius*0.72
	startY := cy - 1 + radius*0.72
	strokeLine(dc, startX, startY, startX+size*0.28, startY+size*0.28, color, width)
}

func drawRefreshIcon(dc uintptr, cx, cy, size float64, color uint32, phase float64) {
	radius := size * 0.34
	width := int(math.Max(2, size*0.1))
	start := phase * 0.9
	sweep := 4.9
	x1 := cx + math.Cos(start)*radius
	y1 := cy + math.Sin(start)*radius
	x2 := cx + math.Cos(start+sweep)*radius
	y2 := cy + math.Sin(start+sweep)*radius

	pen, _, _ := pCreatePen.Call(psSolid, uintptr(width), uintptr(color))
	brush, _, _ := pGetStockObject.Call(nullBrush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	pArc.Call(dc,
		uintptr(int32(math.Round(cx-radius))), uintptr(int32(math.Round(cy-radius))),
		uintptr(int32(math.Round(cx+radius))), uintptr(int32(math.Round(cy+radius))),
		uintptr(int32(math.Round(x1))), uintptr(int32(math.Round(y1))),
		uintptr(int32(math.Round(x2))), uintptr(int32(math.Round(y2))),
	)
	pSelectObject.Call(dc, oldPen)
	pSelectObject.Call(dc, oldBrush)
	pDeleteObject.Call(pen)

	end := start + sweep
	tipX := cx + math.Cos(end)*radius
	tipY := cy + math.Sin(end)*radius
	back := end - 0.7
	strokeLine(dc,
		tipX, tipY,
		tipX-math.Cos(back)*size*0.16, tipY-math.Sin(back)*size*0.16,
		color, width,
	)
}

func drawSpinner(dc uintptr, cx, cy, size, phase float64, color uint32) {
	const dots = 9
	radius := size * 0.34
	dotRadius := math.Max(1.2, size*0.075)
	step := int(phase*10) % dots
	for i := 0; i < dots; i++ {
		angle := float64(i)*2*math.Pi/float64(dots) - math.Pi/2
		x := cx + math.Cos(angle)*radius
		y := cy + math.Sin(angle)*radius
		brightness := 0.15
		if i == step {
			brightness = 1
		} else {
			distance := (i - step + dots) % dots
			brightness = math.Max(0.12, 1-float64(distance)*0.17)
		}
		fillCircle(dc, x, y, dotRadius*brightness+dotRadius*0.35, mixColor(color, colText, brightness))
	}
}

func drawCloseIcon(dc uintptr, cx, cy, size float64, color uint32) {
	width := int(math.Max(1, size*0.11))
	strokeLine(dc, cx-size*0.18, cy-size*0.18, cx+size*0.18, cy+size*0.18, color, width)
	strokeLine(dc, cx+size*0.18, cy-size*0.18, cx-size*0.18, cy+size*0.18, color, width)
}

func drawMinimizeIcon(dc uintptr, cx, cy, size float64, color uint32) {
	width := int(math.Max(1, size*0.11))
	strokeLine(dc, cx-size*0.2, cy+size*0.02, cx+size*0.2, cy+size*0.02, color, width)
}

func drawPinIcon(dc uintptr, cx, cy, size float64, color uint32) {
	width := int(math.Max(1, size*0.11))
	strokeLine(dc, cx, cy+size*0.2, cx, cy-size*0.12, color, width)
	strokeLine(dc, cx-size*0.17, cy, cx, cy-size*0.13, color, width)
	strokeLine(dc, cx, cy-size*0.13, cx+size*0.17, cy, color, width)
	strokeLine(dc, cx-size*0.13, cy+size*0.21, cx+size*0.13, cy+size*0.21, color, width)
}

func fillRoundBounds(dc uintptr, x1, y1, x2, y2, radius float64, color uint32) {
	r := rect{
		left:   int32(math.Round(x1)),
		top:    int32(math.Round(y1)),
		right:  int32(math.Round(x2)),
		bottom: int32(math.Round(y2)),
	}
	if r.right <= r.left || r.bottom <= r.top {
		return
	}
	fillRound(dc, r, int(math.Round(radius)), color)
}
