package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const baseURL = "https://wttr.in"

var client = &http.Client{
	Timeout: 10 * time.Second,
}

func fetchWeather(location string) (string, error) {
	url := fmt.Sprintf("%s/%s?format=3", baseURL, location)

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather service returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

func main() {
	location := "Baku"

	if len(os.Args) > 1 {
		location = strings.Join(os.Args[1:], " ")
	}

	location = strings.TrimSpace(location)

	weather, err := fetchWeather(location)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Printf("Weather in %s:\n%s\n", location, weather)
}
