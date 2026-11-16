// Package weather fetches live weather data from wttr.in.
//
// It needs no API key and has no third-party dependencies, so it builds the
// same way on Linux, macOS and Windows.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultLocation is used when the user does not provide one.
	DefaultLocation = "tehran"
	// DefaultService is the public wttr.in endpoint.
	DefaultService = "https://wttr.in"
	// DefaultTimeout keeps a stalled request from hanging the CLI or the GUI.
	DefaultTimeout = 10 * time.Second

	maxResponseBytes = 4 << 20 // 4 MiB, the j1 payload is a few hundred KiB.
	userAgent        = "GoWeather/1.0 (+https://github.com/SMHO179/GoWeather)"
)

// Client talks to a wttr.in compatible service.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a client with sensible defaults.
func NewClient() *Client {
	return &Client{
		BaseURL: DefaultService,
		HTTP:    &http.Client{Timeout: DefaultTimeout},
	}
}

// NormalizeLocation trims whitespace and falls back to DefaultLocation.
func NormalizeLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return DefaultLocation
	}
	return location
}

// BuildURL returns the wttr.in URL for a location and output format.
func (c *Client) BuildURL(location, format string) string {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultService
	}
	return fmt.Sprintf("%s/%s?format=%s", base, url.PathEscape(NormalizeLocation(location)), url.QueryEscape(format))
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: DefaultTimeout}
}

func (c *Client) get(ctx context.Context, location, format string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BuildURL(location, format), nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "text/plain")

	response, err := c.httpClient().Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to connect to weather service: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather service returned %s", response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return "", fmt.Errorf("weather service returned an empty response")
	}
	return text, nil
}

// Fetch returns the short one-line weather summary (wttr.in format=3).
func (c *Client) Fetch(ctx context.Context, location string) (string, error) {
	return c.get(ctx, NormalizeLocation(location), "3")
}

// FetchForecast downloads and parses the full JSON forecast (wttr.in format=j1).
func (c *Client) FetchForecast(ctx context.Context, location string) (*Forecast, error) {
	body, err := c.get(ctx, NormalizeLocation(location), "j1")
	if err != nil {
		return nil, err
	}

	var forecast Forecast
	if err := json.Unmarshal([]byte(body), &forecast); err != nil {
		return nil, fmt.Errorf("failed to parse weather data: %w", err)
	}
	return &forecast, nil
}

// WeatherDesc is wttr.in's condition text wrapper.
type WeatherDesc struct {
	Value string `json:"value"`
}

// CurrentCondition holds the "now" block of a j1 response.
type CurrentCondition struct {
	TempC            string        `json:"temp_C"`
	FeelsLikeC       string        `json:"FeelsLikeC"`
	Humidity         string        `json:"humidity"`
	WindKmph         string        `json:"windspeedKmph"`
	WindDir          string        `json:"winddir16Point"`
	Pressure         string        `json:"pressure"`
	UVIndex          string        `json:"uvIndex"`
	Visibility       string        `json:"visibility"`
	ObservationTime  string        `json:"observation_time"`
	LocalObsDateTime string        `json:"localObsDateTime"`
	WeatherDesc      []WeatherDesc `json:"weatherDesc"`
}

// Description returns the first non-empty condition description.
func (c CurrentCondition) Description() string {
	for _, desc := range c.WeatherDesc {
		if strings.TrimSpace(desc.Value) != "" {
			return strings.TrimSpace(desc.Value)
		}
	}
	return "Unknown"
}

// AreaName is wttr.in's localized name wrapper.
type AreaName struct {
	Value string `json:"value"`
}

// NearestArea identifies the location the service resolved the query to.
type NearestArea struct {
	AreaName []AreaName `json:"areaName"`
	Region   []AreaName `json:"region"`
	Country  []AreaName `json:"country"`
}

// HourlyForecast is a single hourly slot.
type HourlyForecast struct {
	TempC       string        `json:"tempC"`
	WeatherDesc []WeatherDesc `json:"weatherDesc"`
}

// Description returns the first non-empty condition description.
func (h HourlyForecast) Description() string {
	for _, desc := range h.WeatherDesc {
		if strings.TrimSpace(desc.Value) != "" {
			return strings.TrimSpace(desc.Value)
		}
	}
	return ""
}

// DailyForecast is one day of the three day forecast.
type DailyForecast struct {
	Date      string           `json:"date"`
	MaxTempC  string           `json:"maxtempC"`
	MinTempC  string           `json:"mintempC"`
	AvgTempC  string           `json:"avgtempC"`
	Hourly    []HourlyForecast `json:"hourly"`
	Astronomy []struct {
		Sunrise string `json:"sunrise"`
		Sunset  string `json:"sunset"`
	} `json:"astronomy"`
}

// Description picks a representative condition for the day (around noon).
func (d DailyForecast) Description() string {
	if len(d.Hourly) == 0 {
		return "Unknown"
	}
	index := len(d.Hourly) / 2
	if desc := d.Hourly[index].Description(); desc != "" {
		return desc
	}
	for _, hour := range d.Hourly {
		if desc := hour.Description(); desc != "" {
			return desc
		}
	}
	return "Unknown"
}

// Forecast is the subset of wttr.in's j1 response that the GUI displays.
type Forecast struct {
	CurrentCondition []CurrentCondition `json:"current_condition"`
	NearestArea      []NearestArea      `json:"nearest_area"`
	Weather          []DailyForecast    `json:"weather"`
	Request          []struct {
		Query string `json:"query"`
		Type  string `json:"type"`
	} `json:"request"`
}

func firstValue(values []AreaName) string {
	for _, value := range values {
		if strings.TrimSpace(value.Value) != "" {
			return strings.TrimSpace(value.Value)
		}
	}
	return ""
}

// Place returns a human readable location such as "Tehran, Iran".
func (f *Forecast) Place() string {
	if len(f.NearestArea) > 0 {
		area := f.NearestArea[0]
		parts := make([]string, 0, 3)
		name := firstValue(area.AreaName)
		region := firstValue(area.Region)
		country := firstValue(area.Country)

		if name != "" {
			parts = append(parts, name)
		}
		if region != "" && !strings.EqualFold(region, name) {
			parts = append(parts, region)
		}
		if country != "" && !strings.EqualFold(country, name) {
			parts = append(parts, country)
		}
		if len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
	}
	if len(f.Request) > 0 && strings.TrimSpace(f.Request[0].Query) != "" {
		return strings.TrimSpace(f.Request[0].Query)
	}
	return "Unknown location"
}

func orDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return strings.TrimSpace(value)
}

func formatDay(date string) string {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return parsed.Format("Mon 02 Jan")
}

// Summary renders the forecast as plain text for the CLI and the GUI.
func (f *Forecast) Summary() string {
	var b strings.Builder

	b.WriteString(f.Place())
	b.WriteString("\n")

	if len(f.CurrentCondition) > 0 {
		current := f.CurrentCondition[0]
		observed := current.LocalObsDateTime
		if strings.TrimSpace(observed) == "" {
			observed = current.ObservationTime
		}
		if strings.TrimSpace(observed) != "" {
			fmt.Fprintf(&b, "Observed: %s\n", strings.TrimSpace(observed))
		}

		b.WriteString(strings.Repeat("-", 52))
		b.WriteString("\n")
		fmt.Fprintf(&b, "Now:        %s, %s°C (feels like %s°C)\n", current.Description(), orDash(current.TempC), orDash(current.FeelsLikeC))
		fmt.Fprintf(&b, "Humidity:   %s%%\n", orDash(current.Humidity))
		fmt.Fprintf(&b, "Wind:       %s km/h %s\n", orDash(current.WindKmph), orDash(current.WindDir))
		fmt.Fprintf(&b, "Pressure:   %s hPa\n", orDash(current.Pressure))
		fmt.Fprintf(&b, "Visibility: %s km\n", orDash(current.Visibility))
		fmt.Fprintf(&b, "UV index:   %s\n", orDash(current.UVIndex))
	}

	if len(f.Weather) > 0 {
		b.WriteString("\nForecast\n")
		b.WriteString(strings.Repeat("-", 52))
		b.WriteString("\n")
		for _, day := range f.Weather {
			desc := day.Description()
			if len([]rune(desc)) > 24 {
				desc = string([]rune(desc)[:24])
			}
			fmt.Fprintf(&b, "%-12s  %-24s  %s..%s°C\n",
				formatDay(day.Date), desc, orDash(day.MinTempC), orDash(day.MaxTempC))
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}
