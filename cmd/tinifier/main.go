// Main CLI application entrypoint.
package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"

	"github.com/tarampampam/tinifier/v4/internal/cli"
)

// exitFn is a function for application exiting.
var exitFn = os.Exit //nolint:gochecknoglobals

// main CLI application entrypoint.
func main() {
	code, err := run()
	if err != nil {
		var (
			left  = color.New(color.BgHiRed, color.FgBlack, color.Bold)
			right = color.New(color.FgHiRed, color.BgBlack)
		)

		println(left.Sprintf("  %s  ", "Error") + right.Sprintf("  %s  ", err))
	}

	exitFn(code)
}

// run this CLI application.
// Exit codes documentation: <https://tldp.org/LDP/abs/html/exitcodes.html>
func run() (int, error) {
	const dotenvFileName = ".env" // dotenv (.env) file name

	// load .env file (if file exists; useful for the local app development)
	if stat, dotenvErr := os.Stat(dotenvFileName); dotenvErr == nil && !stat.IsDir() {
		if err := godotenv.Load(dotenvFileName); err != nil {
			return 1, errors.Wrap(err, dotenvFileName+" file error")
		}
	}

	if err := (cli.NewApp()).Run(os.Args); err != nil {
		return 1, err
	}

	return 0, nil
}
