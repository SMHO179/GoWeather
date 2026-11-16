package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/SMHO179/GoWeather/internal/weather"
)

func main() {
	enableWindowsConsole()

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("weather", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	timeout := flags.Duration("timeout", weather.DefaultTimeout, "request timeout (for example 5s or 30s)")
	pause := flags.Bool("pause", false, "wait for Enter before exiting (useful when launched by double-click on Windows)")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "GoWeather - live weather from wttr.in\n\n")
		fmt.Fprintf(flags.Output(), "Usage:\n  weather [flags] [location]\n\n")
		fmt.Fprintf(flags.Output(), "Examples:\n  weather\n  weather London\n  weather \"New York\"\n  weather -timeout 5s Baku\n\nFlags:\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}

	location := weather.NormalizeLocation(strings.Join(flags.Args(), " "))

	client := weather.NewClient()
	client.HTTP.Timeout = *timeout

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	line, err := client.Fetch(ctx, location)
	if err == nil {
		fmt.Printf("Weather in %s\n", location)
		fmt.Println("-------------------------")
		fmt.Println(line)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	if *pause {
		fmt.Fprint(os.Stderr, "\nPress Enter to exit...")
		fmt.Fscanln(os.Stdin)
	}

	return err
}
