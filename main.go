package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultLocation = "Baku"
	weatherService  = "https://wttr.in"
	requestTimeout  = 10 * time.Second
)

var httpClient = &http.Client{
	Timeout: requestTimeout,
}

func buildWeatherURL(location string) string {
	return fmt.Sprintf("%s/%s?format=3", weatherService, location)
}

func fetchWeather(location string) (string, error) {
	response, err := httpClient.Get(buildWeatherURL(location))
	if err != nil {
		return "", fmt.Errorf("failed to connect to weather service: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather service returned %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}

func getLocationFromArgs() string {
	if len(os.Args) <= 1 {
		return defaultLocation
	}

	location := strings.Join(os.Args[1:], " ")
	location = strings.TrimSpace(location)

	if location == "" {
		return defaultLocation
	}

	return location
}

func main() {
	location := getLocationFromArgs()

	weather, err := fetchWeather(location)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Weather in %s\n", location)
	fmt.Println("-------------------------")
	fmt.Println(weather)
}
