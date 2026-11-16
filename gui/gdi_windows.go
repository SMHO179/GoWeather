//go:build windows

package main

import (
	"math"
	"runtime"
	"syscall"
	"unsafe"
)

// Palette (COLORREF: 0x00BBGGRR) — refined Windows 11 dark widget look.
var (
	colBgTop      = rgb(18, 22, 32)
	colBgBottom   = rgb(28, 34, 48)
	colCardTop    = rgb(32, 39, 54)
	colCardBottom = rgb(24, 30, 44)
	colCardBorder = rgb(52, 64, 86)
	colTrack      = rgb(48, 58, 78)
	colField      = rgb(22, 28, 40)
	colText       = rgb(248, 250, 252)
	colTextDim    = rgb(163, 176, 196)
	colTextFaint  = rgb(120, 134, 156)
	colAccent     = rgb(96, 205, 255)
	colAccent2    = rgb(139, 148, 255)
	colAccent3    = rgb(180, 154, 255)
	colWarm       = rgb(251, 191, 36)
	colWarmSoft   = rgb(253, 224, 71)
	colGood       = rgb(74, 222, 128)
	colBad        = rgb(248, 113, 113)
	colCloseBad   = rgb(220, 38, 38)
	colHoverBg    = rgb(46, 56, 76)
	colButtonInk  = rgb(10, 18, 30)
	colCloud      = rgb(203, 213, 225)
	colCloudTop   = rgb(241, 245, 249)
	colCloudDark  = rgb(148, 163, 184)
	colRain       = rgb(125, 211, 252)
	colSnow       = rgb(224, 242, 254)
)

type glowTex struct {
	dc   uintptr
	bmp  uintptr
	size int
}

func (g *glowTex) release() {
	if g == nil {
		return
	}
	if g.bmp != 0 {
		pDeleteObject.Call(g.bmp)
	}
	if g.dc != 0 {
		pDeleteDC.Call(g.dc)
	}
}

func rgb(red, green, blue byte) uint32 {
	return uint32(red) | uint32(green)<<8 | uint32(blue)<<16
}

func mixColor(a, b uint32, t float64) uint32 {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab := float64(a&0xFF), float64((a>>8)&0xFF), float64((a>>16)&0xFF)
	br, bg, bb := float64(b&0xFF), float64((b>>8)&0xFF), float64((b>>16)&0xFF)
	return rgb(
		byte(ar+(br-ar)*t+0.5),
		byte(ag+(bg-ag)*t+0.5),
		byte(ab+(bb-ab)*t+0.5),
	)
}

func gradientFill(dc uintptr, r rect, top, bottom uint32) {
	verts := [2]triVertex{
		{
			x: r.left, y: r.top,
			red: uint16((top & 0xFF) * 0x101), green: uint16(((top >> 8) & 0xFF) * 0x101), blue: uint16(((top >> 16) & 0xFF) * 0x101),
			alpha: 0xFFFF,
		},
		{
			x: r.right, y: r.bottom,
			red: uint16((bottom & 0xFF) * 0x101), green: uint16(((bottom >> 8) & 0xFF) * 0x101), blue: uint16(((bottom >> 16) & 0xFF) * 0x101),
			alpha: 0xFFFF,
		},
	}
	gr := gradientRect{upperLeft: 0, lowerRight: 1}
	pGradientFill.Call(dc, uintptr(unsafe.Pointer(&verts[0])), 2, uintptr(unsafe.Pointer(&gr)), 1, gradientFillRectV)
}

func fillRectColor(dc uintptr, r rect, color uint32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(color))
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), brush)
	pDeleteObject.Call(brush)
}

func fillRound(dc uintptr, r rect, radius int, color uint32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(color))
	pen, _, _ := pGetStockObject.Call(nullPen)
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	pRoundRect.Call(dc, uintptr(r.left), uintptr(r.top), uintptr(r.right), uintptr(r.bottom), uintptr(radius*2), uintptr(radius*2))
	pSelectObject.Call(dc, oldBrush)
	pSelectObject.Call(dc, oldPen)
	pDeleteObject.Call(brush)
}

func strokeRound(dc uintptr, r rect, radius int, color uint32, width int) {
	pen, _, _ := pCreatePen.Call(psSolid, uintptr(width), uintptr(color))
	brush, _, _ := pGetStockObject.Call(nullBrush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	pRoundRect.Call(dc, uintptr(r.left), uintptr(r.top), uintptr(r.right), uintptr(r.bottom), uintptr(radius*2), uintptr(radius*2))
	pSelectObject.Call(dc, oldPen)
	pSelectObject.Call(dc, oldBrush)
	pDeleteObject.Call(pen)
}

func fillRoundGradient(dc uintptr, r rect, radius int, top, bottom uint32) {
	region, _, _ := pCreateRoundRectRgn.Call(
		uintptr(r.left), uintptr(r.top), uintptr(r.right+1), uintptr(r.bottom+1),
		uintptr(radius*2), uintptr(radius*2),
	)
	if region == 0 {
		return
	}
	saved, _, _ := pSaveDC.Call(dc)
	pSelectClipRgn.Call(dc, region)
	gradientFill(dc, r, top, bottom)
	pRestoreDC.Call(dc, saved)
	pDeleteObject.Call(region)
}

func withRoundClip(dc uintptr, r rect, radius int, draw func()) {
	region, _, _ := pCreateRoundRectRgn.Call(
		uintptr(r.left), uintptr(r.top), uintptr(r.right+1), uintptr(r.bottom+1),
		uintptr(radius*2), uintptr(radius*2),
	)
	if region == 0 {
		draw()
		return
	}
	saved, _, _ := pSaveDC.Call(dc)
	pSelectClipRgn.Call(dc, region)
	draw()
	pRestoreDC.Call(dc, saved)
	pDeleteObject.Call(region)
}

func drawText(dc, font uintptr, color uint32, text string, r rect, flags uint32) {
	if text == "" || font == 0 {
		return
	}
	old, _, _ := pSelectObject.Call(dc, font)
	pSetTextColor.Call(dc, uintptr(color))
	pSetBkMode.Call(dc, transparentBkMode)
	ptr, err := syscall.UTF16PtrFromString(text)
	if err == nil {
		pDrawTextW.Call(dc, uintptr(unsafe.Pointer(ptr)), ^uintptr(0), uintptr(unsafe.Pointer(&r)), uintptr(flags))
	}
	pSelectObject.Call(dc, old)
	runtime.KeepAlive(ptr)
}

func textWidth(dc, font uintptr, text string) int32 {
	if text == "" || font == 0 {
		return 0
	}
	units := syscall.StringToUTF16(text)
	old, _, _ := pSelectObject.Call(dc, font)
	var extent size
	pGetTextExtentPoint32W.Call(dc, uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)-1), uintptr(unsafe.Pointer(&extent)))
	pSelectObject.Call(dc, old)
	runtime.KeepAlive(units)
	return extent.cx
}

func strokeLine(dc uintptr, x1, y1, x2, y2 float64, color uint32, width int) {
	pen, _, _ := pCreatePen.Call(psSolid, uintptr(width), uintptr(color))
	old, _, _ := pSelectObject.Call(dc, pen)
	pMoveToEx.Call(dc, uintptr(int32(math.Round(x1))), uintptr(int32(math.Round(y1))), 0)
	pLineTo.Call(dc, uintptr(int32(math.Round(x2))), uintptr(int32(math.Round(y2))))
	pSelectObject.Call(dc, old)
	pDeleteObject.Call(pen)
}

func fillEllipse(dc uintptr, cx, cy, rx, ry float64, color uint32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(color))
	pen, _, _ := pGetStockObject.Call(nullPen)
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	pEllipse.Call(dc,
		uintptr(int32(math.Round(cx-rx))), uintptr(int32(math.Round(cy-ry))),
		uintptr(int32(math.Round(cx+rx))), uintptr(int32(math.Round(cy+ry))),
	)
	pSelectObject.Call(dc, oldBrush)
	pSelectObject.Call(dc, oldPen)
	pDeleteObject.Call(brush)
}

func strokeCircle(dc uintptr, cx, cy, radius float64, color uint32, width int) {
	pen, _, _ := pCreatePen.Call(psSolid, uintptr(width), uintptr(color))
	brush, _, _ := pGetStockObject.Call(nullBrush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	pEllipse.Call(dc,
		uintptr(int32(math.Round(cx-radius))), uintptr(int32(math.Round(cy-radius))),
		uintptr(int32(math.Round(cx+radius))), uintptr(int32(math.Round(cy+radius))),
	)
	pSelectObject.Call(dc, oldPen)
	pSelectObject.Call(dc, oldBrush)
	pDeleteObject.Call(pen)
}

func fillCircle(dc uintptr, cx, cy, radius float64, color uint32) {
	fillEllipse(dc, cx, cy, radius, radius, color)
}

func newGlowTexture(color uint32) *glowTex {
	screen, _, _ := pGetDC.Call(0)
	if screen == 0 {
		return nil
	}
	defer pReleaseDC.Call(0, screen)

	dc, _, _ := pCreateCompatibleDC.Call(screen)
	if dc == 0 {
		return nil
	}

	const dimension = 192
	info := bitmapInfo{}
	info.header.size = uint32(unsafe.Sizeof(info.header))
	info.header.width = dimension
	info.header.height = -dimension
	info.header.planes = 1
	info.header.bitCount = 32
	info.header.compression = biRGB

	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&info)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		pDeleteDC.Call(dc)
		return nil
	}
	pSelectObject.Call(dc, bmp)

	pixels := unsafe.Slice((*byte)(bits), dimension*dimension*4)
	red := float64(color & 0xFF)
	green := float64((color >> 8) & 0xFF)
	blue := float64((color >> 16) & 0xFF)
	center := float64(dimension) / 2

	for y := 0; y < dimension; y++ {
		for x := 0; x < dimension; x++ {
			dx := (float64(x) + 0.5 - center) / center
			dy := (float64(y) + 0.5 - center) / center
			distance := math.Hypot(dx, dy)
			if distance >= 1 {
				continue
			}
			falloff := 1 - distance
			falloff *= falloff
			alpha := falloff * 0.85
			idx := (y*dimension + x) * 4
			pixels[idx] = byte(blue * alpha)
			pixels[idx+1] = byte(green * alpha)
			pixels[idx+2] = byte(red * alpha)
			pixels[idx+3] = byte(alpha * 255)
		}
	}

	return &glowTex{dc: dc, bmp: bmp, size: dimension}
}

func (a *app) glow(color uint32) *glowTex {
	if tex, ok := a.glows[color]; ok {
		return tex
	}
	tex := newGlowTexture(color)
	if tex != nil {
		a.glows[color] = tex
	}
	return tex
}

func (a *app) blitGlow(dc uintptr, color uint32, r rect, strength byte) {
	tex := a.glow(color)
	if tex == nil {
		return
	}
	blend := uint32(strength)<<16 | uint32(acSrcAlpha)<<24
	pAlphaBlend.Call(dc,
		uintptr(r.left), uintptr(r.top), uintptr(r.right-r.left), uintptr(r.bottom-r.top),
		tex.dc, 0, 0, uintptr(tex.size), uintptr(tex.size),
		uintptr(blend),
	)
}
