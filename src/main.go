package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

const (
	AppVersion = "0.0.1"
	envApiKey  = "TINYPNG_API_KEY"
)

type Options struct {
	Verbose     bool   `short:"v" description:"Show verbose debug information"`
	ShowVersion bool   `short:"V" long:"version" description:"Show version and exit"`
	ApiKey      string `short:"k" long:"api-key" description:"API key <https://tinypng.com/dashboard/api>"`
}

var (
	options   Options
	parser    = flags.NewParser(&options, flags.Default)
	errorsLog = log.New(os.Stderr, "", 0)
	apiKey    string
)

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			parser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}

	// Show application version and exit, if flag `-V` passed
	if options.ShowVersion == true {
		fmt.Printf("Version: %s\n", AppVersion)
		os.Exit(0)
	}

	// Check
	if key, err := getApiKey(options); err != nil {
		errorsLog.Println(err)
		os.Exit(1)
	} else {
		apiKey = key
	}

	fmt.Println(apiKey)

	// @todo: Write code
}

// Extract tinypng.com API KEY from passed options or environment variable (as fallback)
func getApiKey(opts Options) (apiKey string, err error) {
	if optsKey := options.ApiKey; len(optsKey) >= 1 {
		return optsKey, nil
	} else if env := os.Getenv(envApiKey); len(env) >= 1 {
		return env, nil
	}

	return "", errors.New("tinypng.com API key is not provided")
}
