// Main CLI application entrypoint.
package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"

	"github.com/tarampampam/tinifier/v4/internal/cli"
	"github.com/tarampampam/tinifier/v4/internal/logger"
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

		println(left.Sprintf("  %s  ", "Error") + right.Sprintf("  %s  ", err)) //nolint:forbidigo
	}

	exitFn(code)
}

// run this CLI application.
// Exit codes documentation: <https://tldp.org/LDP/abs/html/exitcodes.html>
func run() (int, error) {
	log := logger.New(logger.DebugLevel) // JUST FOR A TEST
	log.Debug("debug level", "asd", 123, struct{}{})
	log.Info("info level", "asd", 123, struct{}{})
	log.Warn("warn level", "asd", 123, struct{}{})
	log.Error("error level", "asd", 123, struct{}{})

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
