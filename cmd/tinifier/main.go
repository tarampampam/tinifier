// Main CLI application entrypoint.
package main

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"

	"gh.tarampamp.am/tinifier/v4/internal/cli"
)

// exitFn is a function for application exiting.
var exitFn = os.Exit //nolint:gochecknoglobals

// main CLI application entrypoint.
func main() {
	code, err := run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s%s\n",
			text.Colors{text.BgHiRed, text.FgBlack, text.Bold}.Sprint("  Fatal error  "),
			text.Colors{text.BgBlack, text.FgHiRed}.Sprintf("  %s  ", err.Error()),
		)
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
