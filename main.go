package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const baseURL = "https://wttr.in"

func fetchWeather(location string) (string, error) {
	url := fmt.Sprintf("%s/%s?format=3", baseURL, location)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func main() {
	location := "Baku"

	if len(os.Args) > 1 {
		location = strings.Join(os.Args[1:], " ")
	}

	weather, err := fetchWeather(location)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("🌦 Weather:", weather)
}
