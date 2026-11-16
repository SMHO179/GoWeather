//go:build windows

package main

import (
	"fmt"
	"math"
	"time"
)

func (a *app) render(dc uintptr, w, h int32) {
	phase := time.Since(a.started).Seconds()
	snap := a.view()

	a.drawBackground(dc, w, h, phase)
	a.drawHeader(dc, phase)
	a.drawSearch(dc, phase)
	a.drawHero(dc, snap, phase)
	a.drawForecast(dc, snap, phase)
	a.drawFooter(dc, phase)
	if a.loading {
		a.drawLoadingBar(dc, w, phase)
	}

	border := rect{left: 0, top: 0, right: w - 1, bottom: h - 1}
	strokeRound(dc, border, a.px(roundDIP), rgb(31, 43, 64), 1)
}

func (a *app) drawBackground(dc uintptr, w, h int32, phase float64) {
	gradientFill(dc, rect{right: w, bottom: h}, colBgTop, colBgBottom)

	pulse := 0.5 + 0.5*math.Sin(phase*0.5)
	size := int32(a.px(560))
	a.blitGlow(dc, colAccent, rect{
		left:   w - size*2/3,
		top:    -size / 2,
		right:  w + size/3,
		bottom: size / 2,
	}, byte(34+pulse*14))

	violet := int32(a.px(480))
	a.blitGlow(dc, colAccent3, rect{
		left:   -violet / 2,
		top:    h - violet/2,
		right:  violet / 2,
		bottom: h + violet/3,
	}, 26)
}

func (a *app) drawHeader(dc uintptr, phase float64) {
	pad := a.px32(24)
	logoSize := a.px32(32)
	logo := rect{left: pad, top: a.px32(18), right: pad + logoSize, bottom: a.px32(18) + logoSize}
	drawLogo(dc, logo)

	drawText(dc, a.fonts.title, colText, appName, rect{
		left: logo.right + a.px32(12), top: a.px32(17),
		right: logo.right + a.px32(220), bottom: a.px32(36),
	}, dtSingleLine|dtVCenter|dtNoPrefix)

	drawText(dc, a.fonts.small, colTextFaint, "live weather", rect{
		left: logo.right + a.px32(12), top: a.px32(36),
		right: logo.right + a.px32(220), bottom: a.px32(51),
	}, dtSingleLine|dtVCenter|dtNoPrefix)

	// Window buttons.
	pinHover := a.hover[btnPin]
	if a.pinned {
		fillRound(dc, a.lay.pin, a.px(8), mixColor(colBgTop, colAccent, 0.16+0.08*pinHover))
	} else {
		fillRound(dc, a.lay.pin, a.px(8), mixColor(colBgTop, colHoverBg, pinHover))
	}
	pinColor := mixColor(colTextDim, colText, pinHover)
	if a.pinned {
		pinColor = colAccent
	}
	drawPinIcon(dc, centerX(a.lay.pin), centerY(a.lay.pin), a.pxF(18), pinColor)

	minHover := a.hover[btnMinimize]
	fillRound(dc, a.lay.min, a.px(8), mixColor(colBgTop, colHoverBg, minHover))
	drawMinimizeIcon(dc, centerX(a.lay.min), centerY(a.lay.min), a.pxF(18), mixColor(colTextDim, colText, minHover))

	closeHover := a.hover[btnClose]
	fillRound(dc, a.lay.close, a.px(8), mixColor(colBgTop, colCloseBad, closeHover*0.9))
	drawCloseIcon(dc, centerX(a.lay.close), centerY(a.lay.close), a.pxF(18), mixColor(colTextDim, colText, closeHover))
}

func (a *app) drawSearch(dc uintptr, phase float64) {
	field := a.lay.search
	radius := a.px(12)
	fillRound(dc, field, radius, colField)
	borderColor := mixColor(colCardBorder, colAccent, a.focus*0.9)
	width := 1
	if a.focus > 0.5 {
		width = 2
	}
	strokeRound(dc, field, radius, borderColor, width)

	iconColor := mixColor(colTextFaint, colAccent, a.focus)
	drawSearchIcon(dc,
		float64(field.left)+a.pxF(23),
		float64(field.top)+a.pxF(22),
		a.pxF(18), iconColor,
	)

	// Update button.
	button := a.lay.refresh
	if a.pressed == btnRefresh {
		button.top += a.px32(1)
		button.bottom += a.px32(1)
	}
	hover := a.hover[btnRefresh]
	top := mixColor(colAccent, rgb(186, 230, 253), hover*0.85)
	bottom := mixColor(colAccent2, rgb(199, 210, 254), hover*0.85)
	if a.pressed == btnRefresh {
		top = mixColor(top, rgb(14, 116, 144), 0.2)
		bottom = mixColor(bottom, rgb(79, 70, 229), 0.2)
	}
	fillRoundGradient(dc, button, a.px(12), top, bottom)
	strokeRound(dc, button, a.px(12), mixColor(colAccent, rgb(255, 255, 255), 0.12+hover*0.12), 1)

	label := "Update"
	if a.loading {
		label = "Updating"
	}
	textW := float64(textWidth(dc, a.fonts.button, label))
	iconSize := a.pxF(16)
	gap := a.pxF(8)
	total := float64(iconSize) + float64(gap) + textW
	startX := float64(button.left) + (float64(button.right-button.left)-total)/2
	iconCX := startX + float64(iconSize)/2
	iconCY := centerY(button)

	if a.loading {
		drawSpinner(dc, iconCX, iconCY, a.pxF(17), phase*1.6, colButtonInk)
	} else {
		drawRefreshIcon(dc, iconCX, iconCY, a.pxF(17), colButtonInk, phase*0.7)
	}

	drawText(dc, a.fonts.button, colButtonInk, label, rect{
		left:   int32(startX) + int32(iconSize+gap),
		top:    button.top,
		right:  button.right,
		bottom: button.bottom,
	}, dtLeft|dtSingleLine|dtVCenter|dtNoPrefix)
}

func (a *app) drawHero(dc uintptr, snap snapshot, phase float64) {
	card := a.lay.hero
	radius := a.px(16)
	fillRoundGradient(dc, card, radius, colCardTop, colCardBottom)

	pulse := 0.5 + 0.5*math.Sin(phase*0.85)
	glowColor := iconGlowColor(snap.icon)
	withRoundClip(dc, card, radius, func() {
		glowSize := int32(a.px(200)) + int32(pulse*16)
		cx := card.right - a.px32(86)
		cy := card.top + a.px32(88)
		a.blitGlow(dc, glowColor, rect{
			left: cx - glowSize/2, top: cy - glowSize/2,
			right: cx + glowSize/2, bottom: cy + glowSize/2,
		}, byte(70+pulse*40))

		a.blitGlow(dc, colAccent3, rect{
			left:   card.left - a.px32(70),
			top:    card.bottom - a.px32(200),
			right:  card.left + a.px32(200),
			bottom: card.bottom + a.px32(30),
		}, 32)
		// subtle top highlight for glassy depth
		highlight := rect{left: card.left + a.px32(1), top: card.top + a.px32(1), right: card.right - a.px32(1), bottom: card.top + a.px32(2)}
		fillRectColor(dc, highlight, mixColor(colCardTop, rgb(255, 255, 255), 0.06))
	})
	strokeRound(dc, card, radius, colCardBorder, 1)

	innerX := card.left + a.px32(22)
	textRight := card.left + a.px32(250)
	innerTop := card.top

	drawText(dc, a.fonts.place, colText, snap.place, rect{
		left: innerX, top: innerTop + a.px32(16),
		right: textRight, bottom: innerTop + a.px32(44),
	}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)

	drawText(dc, a.fonts.desc, colTextDim, snap.desc, rect{
		left: innerX, top: innerTop + a.px32(46),
		right: textRight, bottom: innerTop + a.px32(66),
	}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)

	observed := ""
	if snap.observed != "" {
		observed = "Observed " + snap.observed
	}
	drawText(dc, a.fonts.small, colTextFaint, observed, rect{
		left: innerX, top: innerTop + a.px32(68),
		right: textRight, bottom: innerTop + a.px32(84),
	}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)

	tempText := "--°"
	if a.hasData {
		tempText = fmt.Sprintf("%d°", int(math.Round(a.tempDisplay)))
	}
	drawText(dc, a.fonts.temp, colText, tempText, rect{
		left: innerX - a.px32(3), top: innerTop + a.px32(82),
		right: innerX + a.px32(250), bottom: innerTop + a.px32(168),
	}, dtLeft|dtSingleLine|dtVCenter|dtNoPrefix)

	summary := ""
	if a.hasData && len(snap.days) > 0 {
		summary = fmt.Sprintf("Feels like %d°    High %d°    Low %d°", snap.feelsC, snap.highC, snap.lowC)
	}
	drawText(dc, a.fonts.label, colTextDim, summary, rect{
		left: innerX, top: innerTop + a.px32(169),
		right: textRight + a.px32(50), bottom: innerTop + a.px32(189),
	}, dtLeft|dtSingleLine|dtVCenter|dtNoPrefix)

	fillRectColor(dc, rect{
		left: innerX, top: innerTop + a.px32(203),
		right: card.right - a.px32(22), bottom: innerTop + a.px32(204),
	}, colCardBorder)

	stats := heroStats(snap, a.hasData, a.loading)
	colW := float64((card.right - a.px32(22) - innerX)) / 3
	for i, stat := range stats {
		x := innerX + int32(float64(i%3)*colW)
		y := innerTop + a.px32(216) + int32(i/3)*a.px32(36)
		drawText(dc, a.fonts.statLbl, colTextFaint, stat.label, rect{
			left: x, top: y, right: x + int32(colW), bottom: y + a.px32(13),
		}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
		drawText(dc, a.fonts.stat, colText, stat.value, rect{
			left: x, top: y + a.px32(13), right: x + int32(colW), bottom: y + a.px32(33),
		}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
	}

	iconArea := rect{
		left: card.right - a.px32(156), top: card.top + a.px32(30),
		right: card.right - a.px32(30), bottom: card.top + a.px32(156),
	}
	drawWeatherIcon(dc, snap.icon, iconArea, phase)
}

type statLine struct {
	label string
	value string
}

func heroStats(snap snapshot, hasData, loading bool) []statLine {
	if !hasData {
		status := "Idle"
		if loading {
			status = "Updating"
		}
		return []statLine{
			{label: "Humidity", value: "—"},
			{label: "Wind", value: "—"},
			{label: "Pressure", value: "—"},
			{label: "UV index", value: "—"},
			{label: "Sunrise", value: "—"},
			{label: "Status", value: status},
		}
	}
	windLabel := "Wind"
	if snap.windDir != "" {
		windLabel = "Wind · " + snap.windDir
	}
	sunrise := snap.sunrise
	if sunrise == "" {
		sunrise = "—"
	}
	sunset := snap.sunset
	if sunset == "" {
		sunset = "—"
	}
	return []statLine{
		{label: "Humidity", value: withUnit(snap.humidity, "%")},
		{label: windLabel, value: withUnit(snap.wind, " km/h")},
		{label: "Pressure", value: withUnit(snap.pressure, " hPa")},
		{label: "UV index", value: snap.uv},
		{label: "Sunrise", value: sunrise},
		{label: "Sunset", value: sunset},
	}
}

func (a *app) drawForecast(dc uintptr, snap snapshot, phase float64) {
	globalMin, globalMax := 0, 0
	if len(snap.days) > 0 {
		globalMin = snap.days[0].minC
		globalMax = snap.days[0].maxC
		for _, day := range snap.days {
			if day.minC < globalMin {
				globalMin = day.minC
			}
			if day.maxC > globalMax {
				globalMax = day.maxC
			}
		}
	}
	span := float64(globalMax - globalMin)
	if span < 1 {
		span = 1
	}

	for i := 0; i < 3; i++ {
		card := a.lay.cards[i]
		radius := a.px(14)
		hover := a.hoverCard[i]

		top := mixColor(colCardTop, rgb(31, 47, 76), hover*0.7)
		bottom := mixColor(colCardBottom, rgb(23, 36, 58), hover*0.7)
		fillRoundGradient(dc, card, radius, top, bottom)
		strokeRound(dc, card, radius, mixColor(colCardBorder, colAccent, hover*0.45), 1)

		if i >= len(snap.days) {
			drawText(dc, a.fonts.cardDesc, colTextFaint, "No forecast available", rect{
				left: card.left + a.px32(18), top: card.top,
				right: card.right - a.px32(18), bottom: card.bottom,
			}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
			continue
		}

		day := snap.days[i]
		drawText(dc, a.fonts.cardDay, colText, day.label, rect{
			left: card.left + a.px32(18), top: card.top + a.px32(12),
			right: card.left + a.px32(220), bottom: card.top + a.px32(32),
		}, dtLeft|dtSingleLine|dtVCenter|dtNoPrefix)

		drawText(dc, a.fonts.cardDesc, colTextDim, day.desc, rect{
			left: card.left + a.px32(18), top: card.top + a.px32(36),
			right: card.right - a.px32(130), bottom: card.top + a.px32(56),
		}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)

		iconArea := rect{
			left: card.right - a.px32(124), top: card.top + a.px32(22),
			right: card.right - a.px32(78), bottom: card.top + a.px32(68),
		}
		drawWeatherIcon(dc, day.icon, iconArea, phase+float64(i)*1.7)

		drawText(dc, a.fonts.cardMax, colText, fmt.Sprintf("%d°", day.maxC), rect{
			left: card.right - a.px32(76), top: card.top + a.px32(12),
			right: card.right - a.px32(18), bottom: card.top + a.px32(38),
		}, dtRight|dtSingleLine|dtVCenter|dtNoPrefix)

		drawText(dc, a.fonts.cardMin, colTextDim, fmt.Sprintf("low %d°", day.minC), rect{
			left: card.right - a.px32(86), top: card.top + a.px32(38),
			right: card.right - a.px32(18), bottom: card.top + a.px32(56),
		}, dtRight|dtSingleLine|dtVCenter|dtNoPrefix)

		barLeft := card.left + a.px32(18)
		barRight := card.right - a.px32(18)
		barTop := card.bottom - a.px32(16)
		track := rect{left: barLeft, top: barTop, right: barRight, bottom: barTop + a.px32(4)}
		fillRound(dc, track, a.px(2), colTrack)

		start := float64(day.minC-globalMin) / span
		end := float64(day.maxC-globalMin) / span
		if end < start {
			start, end = end, start
		}
		barWidth := float64(barRight - barLeft)
		segmentLeft := barLeft + int32(start*barWidth)
		segmentRight := barLeft + int32(end*barWidth)
		if segmentRight-segmentLeft < a.px32(6) {
			segmentRight = segmentLeft + a.px32(6)
		}
		fillRound(dc, rect{left: segmentLeft, top: barTop, right: segmentRight, bottom: barTop + a.px32(4)},
			a.px(2), mixColor(colAccent, colAccent2, float64(i)/2))

		avg := (float64(day.minC+day.maxC)/2 - float64(globalMin)) / span
		dotX := barLeft + int32(avg*barWidth)
		fillCircle(dc, float64(dotX), float64(barTop)+a.pxF(2), a.pyF(3), colText)
	}
}

func (a *app) drawFooter(dc uintptr, phase float64) {
	footer := a.lay.footer
	cy := centerY(footer)
	left := footer.left
	textLeft := left + a.px32(16)

	switch {
	case a.loading:
		drawSpinner(dc, float64(left)+a.pxF(7), float64(cy), a.pxF(13), phase*1.6, colAccent)
		drawText(dc, a.fonts.small, colTextDim, "Updating weather…", rect{
			left: textLeft, top: footer.top, right: footer.right - a.px32(170), bottom: footer.bottom,
		}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
	case a.errText != "":
		fillCircle(dc, float64(left)+a.pxF(7), float64(cy), a.pyF(3), colBad)
		drawText(dc, a.fonts.small, colBad, a.errText, rect{
			left: textLeft, top: footer.top, right: footer.right - a.px32(170), bottom: footer.bottom,
		}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
	case a.hasData:
		pulse := 0.55 + 0.45*math.Sin(phase*1.6)
		fillCircle(dc, float64(left)+a.pxF(7), float64(cy), a.pyF(3)*pulse+a.pyF(1.2), colGood)
		label := "Updated " + a.snap.when.Format("15:04:05")
		drawText(dc, a.fonts.small, colTextDim, label, rect{
			left: textLeft, top: footer.top, right: footer.right - a.px32(170), bottom: footer.bottom,
		}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
	default:
		fillCircle(dc, float64(left)+a.pxF(7), float64(cy), a.pyF(3), colTextFaint)
		drawText(dc, a.fonts.small, colTextDim, "Ready", rect{
			left: textLeft, top: footer.top, right: footer.right - a.px32(170), bottom: footer.bottom,
		}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
	}

	drawText(dc, a.fonts.small, colTextFaint, "powered by wttr.in", rect{
		left: footer.right - a.px32(200), top: footer.top, right: footer.right, bottom: footer.bottom,
	}, dtRight|dtSingleLine|dtVCenter|dtNoPrefix)
}

func (a *app) drawLoadingBar(dc uintptr, w int32, phase float64) {
	segment := float64(a.px(170))
	progress := math.Mod(phase/1.7, 1)
	x := -segment + progress*(float64(w)+segment)
	left := int32(x)
	right := int32(x + segment)
	if left < 0 {
		left = 0
	}
	if right > w {
		right = w
	}
	if right <= left {
		return
	}
	fillRectColor(dc, rect{left: left, top: 0, right: right, bottom: a.px32(2)}, colAccent)
	fillRectColor(dc, rect{left: left, top: a.px32(2), right: right, bottom: a.px32(3)}, mixColor(colAccent, colBgTop, 0.55))
}

func iconGlowColor(kind iconKind) uint32 {
	switch kind {
	case iconSun, iconThunder:
		return rgb(245, 158, 11)
	case iconRain, iconSnow, iconFog:
		return colAccent
	default:
		return colAccent2
	}
}

func withUnit(value, unit string) string {
	if value == "—" || value == "" {
		return "—"
	}
	return value + unit
}

func centerX(r rect) float64 {
	return float64(r.left) + float64(r.right-r.left)/2
}

func centerY(r rect) float64 {
	return float64(r.top) + float64(r.bottom-r.top)/2
}

func (a *app) px32(value int) int32 {
	return int32(a.px(value))
}

func (a *app) pxF(value float64) float64 {
	return value * a.scale
}

func (a *app) pyF(value float64) float64 {
	return value * a.scale
}
